## generator

一个基于反射的 Go 代码生成器，用于为 `github.com/php-any/origami` 生态自动生成 Class/Method/Function 代理（Wrapper）。

- **输入**: 一个“构造函数”（任意返回 `*struct` 的函数，如 `sql.Open`、`log.New`，或你自己的函数）
- **输出**: 在指定输出根目录（通常为 `origami/`）下，为对应包生成：
  - 顶级函数代理：`<func>_func.go`
  - 类文件：`<Type>_class.go`
  - 方法代理：`<Type>_<method>_method.go`
  - 自动汇总注册：`load.go`（提供 `Load(vm data.VM)`）

这些文件遵循 Origami 的 `data`/`node` 接口规范，可直接在 VM 中注册并使用。

### 运行环境

- Go 1.20+（go.mod 当前写为 `go 1.24`，实际使用 1.20+ 即可）
- 依赖 `github.com/php-any/origami`（`data` 与 `node` 包）

### 安装

本模块已作为 `github.com/php-any/generator` 发布并可直接安装：

```bash
go get github.com/php-any/generator
```

随后在代码中以 `import "github.com/php-any/generator"` 使用。

### 快速开始

以下示例展示如何为标准库 `database/sql` 的 `Open` 生成代理（会自动为返回的 `*sql.DB` 及其方法生成类与方法代理）。

```go
package main

import (
  "database/sql"
  "fmt"
  "os"

  _ "github.com/go-sql-driver/mysql"
  "github.com/php-any/generator"
)

// 入口：根据 array 中配置的构造函数，生成 origami 目录下的类与方法包装代码
func main() {
  array := []any{
    sql.Open,
  }

  outRoot := "origami"
  for _, elem := range array {
    if err := generator.GenerateFromConstructor(elem, generator.GenOptions{OutputRoot: outRoot}); err != nil {
      fmt.Fprintln(os.Stderr, "生成失败:", err)
      continue
    }
    fmt.Println("生成完成 ->", outRoot)
  }
}

```

运行后将生成：

- `origami/sql/open_func.go` 顶级函数代理
- `origami/sql/db_class.go` 类定义（含属性、方法索引等）
- `origami/sql/db_querycontext_method.go` 等方法代理文件（按需）
- `origami/sql/load.go` 自动注册入口

你可以对任意返回 `*struct` 的构造函数重复调用 `GenerateFromConstructor`；生成器会做去重缓存，避免递归死循环和重复输出。

### 如何在 Origami VM 中加载

生成的 `load.go` 会导出一个 `Load(vm data.VM)`：

```go
// 假设在 origami/sql 下
func Load(vm data.VM) {
    // 自动将 open 函数与 DB 类注册到 VM
}
```

你的宿主程序可在适当时机调用：

```go
import (
    "github.com/php-any/origami/data"
    sqlpkg "your-module/origami/sql" // 以你的模块路径为准
)

func boot(vm data.VM) {
    sqlpkg.Load(vm)
}
```

### 生成产物与约定

- **命名与目录**
  - 输出根目录为 `GenOptions.OutputRoot`，推荐设为 `origami`
  - 按源包短名分子目录：例如 `database/sql` → `origami/sql`
  - 类文件：`<Type>_class.go`（如 `db_class.go` → 类名 `DB`）
  - 方法文件：`<Type>_<method>_method.go`
  - 函数文件：`<func>_func.go`
  - 注册文件：`load.go`（自动扫描当前目录，把类与函数统一注册）
- **类名特殊处理**
  - `db_class.go` → `DB`
  - `txoptions_class.go` → `TxOptions`
  - 其余使用下划线到驼峰的通用转换

### 参数/返回值/错误处理规则（与 Origami 约定一致）

- **参数提取**（在 `Call(ctx data.Context)` 内）：
  - `string` → `*data.StringValue`
  - `int`/`int64` → `*data.IntValue`（`int64` 通过 `int → int64` 转换）
  - `bool` → `*data.BoolValue`
  - `[]T` → `*data.ArrayValue`
  - `interface{}`/具名接口 → `*data.AnyValue`，必要时做具体类型断言
  - `*Struct` → 从代理类 `*data.ClassValue` 取出其 `source` 指针再调用原方法
  - `context.Context` 会按接口处理，不会被当作 `*Struct` 生成代理
- **返回值封装**：
  - 无返回：`return nil, nil`
  - 单返回：`data.NewAnyValue(ret0)`
  - 多返回：`data.NewAnyValue([]any{...})`
  - 单返回且为 `*Struct`：返回 `data.NewClassValue(New<Class>ClassFrom(ret0), ctx)`，自动把原始指针包裹为类实例
- **错误处理**：
  - 若原函数/方法最后一个返回值为 `error`，非 `nil` 时返回 `data.NewErrorThrow(nil, err)`

### 参数命名策略

- 运行时通过 `runtime.FuncForPC` + 源码解析尝试提取真实参数名
- 若无法提取（如源码不可用），退回 `param0/param1/...`
- 部分常见语义会被猜测命名，例如 `Context` → `ctx`，`*Options` → `opts`，变长/切片 → `args`

### 跨包类型与递归生成

- 当某个方法的参数或返回值为 `*Struct` 时：
  - 会为目标结构体生成对应的类与方法代理（跨包同样生效）
  - 为避免循环与重复生成，内部维护全局去重缓存

### 常见问题（FAQ）

- Q: 为什么有些参数名是 `param0/param1`？
  - A: 生成器需要访问到被包装函数/方法的源码才能提取到真实参数名；当源码不可达时会退回占位名。
- Q: 可以多次对同一构造函数调用生成吗？
  - A: 可以，生成器有去重缓存；重复类型会被跳过。
- Q: 可以只生成函数代理而不生成类吗？
  - A: 可以。如果构造函数不返回 `*struct`，仅会生成函数代理文件。
- Q: 如何自定义输出目录？
  - A: 通过 `GenOptions{OutputRoot: "..."}` 指定，通常推荐 `origami`。

### 参与贡献

欢迎 Issue 与 PR。提交前请确保：

- 代码 `go fmt` 通过（本项目在写文件时会尝试 `go/format`，失败则写原始内容以便排错）
- 遵循 Origami 的 `data/node` 接口约定与错误处理约定

---

如需更多上下文或边界条件说明，请参考源码中的 `generate.go`、`method_builder.go`、`func_builder.go` 与 `class_builder.go`。
