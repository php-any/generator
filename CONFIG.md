# 配置文件说明

代码生成器支持 YAML 格式的配置文件，用于自定义生成行为。

## 配置文件格式

代码生成器使用 YAML 格式的配置文件：

```bash
./generator-bin -config config.yaml
```

## 配置项说明

### 基本配置

| 配置项          | 类型   | 默认值      | 说明             |
| --------------- | ------ | ----------- | ---------------- |
| `global_prefix` | string | "origami"   | 全局包前缀       |
| `output_root`   | string | "generated" | 输出目录         |
| `max_depth`     | int    | 3           | 最大递归深度     |
| `parallel`      | bool   | false       | 是否并行处理     |
| `verbose`       | bool   | false       | 是否显示详细日志 |

### 黑名单配置 (`blacklist`)

用于排除不需要生成的包、类型或方法。

```yaml
blacklist:
  packages:
    - internal/*
    - vendor/*
  types:
    - context.Context
    - time.Time
  methods:
    - String
    - Error
  use_regex: false
  patterns: []
```

- `packages`: 包路径黑名单，支持通配符 `*`
- `types`: 类型名称黑名单
- `methods`: 方法名称黑名单
- `use_regex`: 是否使用正则表达式匹配
- `patterns`: 正则表达式模式列表

### 包前缀配置 (`package_prefixes`)

为特定包设置前缀，用于控制生成的代理类命名。

```yaml
package_prefixes:
  github.com/php-any/origami/application: appsrc
  github.com/php-any/origami/window: windowsrc
```

### 包映射配置 (`package_mappings`)

最重要的配置项，用于定义源包到目标包的映射关系。这是生成正确 import 语句的关键。

#### 默认映射规则

代码生成器内置了以下默认映射规则：

```yaml
package_mappings:
  applicationsrc: github.com/php-any/origami/application
  contextsrc: github.com/php-any/origami/context
  eventssrc: github.com/php-any/origami/events
  httpsrc: github.com/php-any/origami/http
  windowsrc: github.com/php-any/origami/window
  menusrc: github.com/php-any/origami/menu
  dialogsrc: github.com/php-any/origami/dialog
  clipboardsrc: github.com/php-any/origami/clipboard
  keyboardsrc: github.com/php-any/origami/keyboard
  mousesrc: github.com/php-any/origami/mouse
  devicesrc: github.com/php-any/origami/device
  storagesrc: github.com/php-any/origami/storage
  networkingsrc: github.com/php-any/origami/networking
  securitysrc: github.com/php-any/origami/security
  uilibsrc: github.com/php-any/origami/uilib
```

#### 自定义映射

你可以添加自定义的包映射：

```yaml
package_mappings:
  customsrc: github.com/example/custom
  myappsrc: github.com/your-org/your-app
```

### 高级配置 (`advanced`)

```yaml
advanced:
  debug: false
  keep_comments: true
  generate_tests: false
  template_path: ./templates
  cache:
    enabled: true
    directory: ./cache
    ttl: 3600
    max_size: 1000
```

- `debug`: 是否生成调试信息
- `keep_comments`: 是否保留原始注释
- `generate_tests`: 是否生成测试文件
- `template_path`: 自定义模板路径
- `cache`: 缓存配置
  - `enabled`: 是否启用缓存
  - `directory`: 缓存目录
  - `ttl`: 缓存过期时间（秒）
  - `max_size`: 缓存最大条目数

## 配置文件示例

参考以下文件：

- `config.example.yaml` - YAML 格式示例

## 使用方式

1. 复制示例配置文件：

   ```bash
   cp config.example.yaml config.yaml
   ```

2. 根据需要修改配置：

   ```bash
   vim config.yaml
   ```

3. 运行生成器：
   ```bash
   ./generator-bin -config config.yaml
   ```

## 配置优先级

1. 命令行参数（最高优先级）
2. 配置文件
3. 默认配置（最低优先级）

配置项会按照优先级进行合并，高优先级的配置会覆盖低优先级的配置。
