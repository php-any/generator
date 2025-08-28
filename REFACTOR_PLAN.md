# Generator 重构方案文档

## 1. 项目概述

### 1.1 项目简介

Generator 是一个基于反射的 Go 代码生成器，用于为 `github.com/php-any/origami` 生态自动生成 Class/Method/Function 代理（Wrapper）。

### 1.2 核心功能

- 输入：构造函数（返回 `*struct` 的函数）或结构体类型
- 输出：生成 origami 兼容的代理类、方法和函数
- 支持递归代理生成和跨包类型处理

## 2. 当前问题分析

### 2.1 代码结构问题

#### 2.1.1 职责混乱

- `generate.go` 既是入口又包含具体生成逻辑
- 单个文件承担过多职责，难以维护和测试
- 缺乏清晰的层次划分

#### 2.1.2 代码重复

- `func_builder.go`、`method_builder.go`、`class_builder.go` 中存在大量相似的参数处理逻辑
- 类型转换、包导入、错误处理逻辑重复
- 缺乏统一的抽象层

#### 2.1.3 函数过长

- `buildMethodFileBody` 函数超过 400 行
- `buildClassFileBody` 函数超过 300 行
- 单个函数承担过多任务，难以理解和修改

### 2.2 类型处理问题

#### 2.2.1 反射使用不当

- 过多的反射操作影响性能
- 缺乏类型信息缓存机制
- 反射逻辑分散在多个文件中

#### 2.2.2 包导入混乱

- 导入逻辑分散在各处，难以统一管理
- 缺乏包别名的一致性处理
- 跨包依赖处理不够优雅

#### 2.2.3 类型转换分散

- 相同的类型转换逻辑在多个文件中重复
- 缺乏统一的类型处理策略
- 错误处理不统一

### 2.3 错误处理问题

#### 2.3.1 错误信息不一致

- 不同文件的错误信息格式不统一
- 缺乏错误分类和错误码
- 调试信息不够详细

#### 2.3.2 错误处理分散

- 错误处理逻辑散布在代码中
- 缺乏统一的错误恢复机制
- 错误传播路径不清晰

### 2.4 性能问题

#### 2.4.1 重复分析

- 相同类型被重复分析多次
- 缺乏分析结果缓存
- 反射操作频繁

#### 2.4.2 文件 I/O 效率

- 文件生成缺乏并行处理
- 缺乏增量更新机制
- 文件写入效率不高

### 2.5 配置管理问题

#### 2.5.1 配置选项不足

- 缺乏对特定包的黑名单配置
- 缺乏包级别的命名前缀配置
- 缺乏依赖包映射配置

#### 2.5.2 配置灵活性差

- 配置选项硬编码在代码中
- 缺乏外部配置文件支持
- 配置验证机制不完善

## 3. 重构目标

### 3.1 架构目标

- 建立清晰的分层架构
- 实现单一职责原则
- 提供可扩展的插件化设计

### 3.2 质量目标

- 提高代码可维护性
- 增强代码可读性
- 改善错误处理机制

### 3.3 性能目标

- 减少重复的反射操作
- 实现类型分析缓存
- 支持并行代码生成

### 3.4 功能目标

- 保持向后兼容性
- 增强类型处理能力
- 提供更好的调试支持
- 增强配置管理能力

## 4. 新架构设计

### 4.1 目录结构

```
generator/
├── core/                    # 核心接口和类型定义
│   ├── types.go            # 统一类型定义
│   ├── context.go          # 生成上下文
│   ├── errors.go           # 统一错误处理
│   └── interfaces.go       # 核心接口定义
├── analyzer/               # 类型分析层
│   ├── type_analyzer.go    # 类型分析器
│   ├── package_analyzer.go # 包分析器
│   ├── method_analyzer.go  # 方法分析器
│   ├── field_analyzer.go   # 字段分析器
│   └── function_analyzer.go # 函数分析器
├── generator/              # 代码生成层
│   ├── base_gen.go         # 基础生成器
│   ├── function_gen.go     # 函数生成器
│   ├── class_gen.go        # 类生成器
│   ├── method_gen.go       # 方法生成器
│   └── template_gen.go     # 模板生成器
├── converter/              # 类型转换层
│   ├── param_converter.go  # 参数转换器
│   ├── return_converter.go # 返回值转换器
│   ├── type_converter.go   # 通用类型转换器
│   └── strategies/         # 转换策略
│       ├── basic_types.go  # 基础类型转换
│       ├── struct_types.go # 结构体类型转换
│       ├── interface_types.go # 接口类型转换
│       ├── func_types.go   # 函数类型转换
│       └── slice_types.go  # 切片类型转换
├── emitter/                # 输出层
│   ├── file_emitter.go     # 文件输出器
│   ├── load_emitter.go     # load.go 生成器
│   ├── code_manager.go     # 代码生成器-管理import等信息，防止漏导入和重复生成
│   ├── registry.go         # 注册管理器
│   └── cache.go            # 缓存管理器-防止循环引用和生成
├── utils/                  # 工具函数
│   ├── reflection.go       # 反射工具
│   ├── naming.go          # 命名工具
│   ├── path.go            # 路径工具
│   └── validation.go      # 验证工具
├── templates/              # 代码模板
│   ├── function.tmpl      # 函数模板
│   ├── class.tmpl         # 类模板
│   └── method.tmpl        # 方法模板
├── config/                 # 配置管理
│   ├── options.go         # 配置选项
│   ├── defaults.go        # 默认配置
│   ├── validator.go       # 配置验证器
│   ├── loader.go          # 配置加载器
│   ├── blacklist.go       # 黑名单管理
│   └── package_mapping.go # 包映射管理
└── main.go                 # 主入口
```

### 4.2 核心接口设计

#### 4.2.1 生成上下文

```go
// core/context.go
type GeneratorContext struct {
    Options        *GenOptions
    GeneratedTypes map[string]bool
    TypeCache      map[string]*TypeInfo
    ErrorHandler   ErrorHandler
    Metrics        *Metrics
    ConfigManager  ConfigManager
}

type GenOptions struct {
    OutputRoot     string
    NamePrefix     string
    MaxDepth       int
    Parallel       bool
    Verbose        bool
    TemplatePath   string
    ConfigFile     string
}
```

#### 4.2.2 配置管理接口

```go
// config/options.go
type ConfigManager interface {
    LoadConfig(path string) error
    ValidateConfig() error
    IsPackageBlacklisted(pkgPath string) bool
    GetPackagePrefix(pkgPath string) string
    GetPackageMapping(sourcePkg string) (string, bool)
    GetGlobalPrefix() string
}

type GeneratorConfig struct {
    // 全局配置
    GlobalPrefix string `json:"global_prefix" yaml:"global_prefix"`
    OutputRoot   string `json:"output_root" yaml:"output_root"`
    MaxDepth     int    `json:"max_depth" yaml:"max_depth"`
    Parallel     bool   `json:"parallel" yaml:"parallel"`
    Verbose      bool   `json:"verbose" yaml:"verbose"`

    // 黑名单配置
    Blacklist BlacklistConfig `json:"blacklist" yaml:"blacklist"`

    // 包前缀配置
    PackagePrefixes map[string]string `json:"package_prefixes" yaml:"package_prefixes"`

    // 依赖包映射配置
    PackageMappings map[string]string `json:"package_mappings" yaml:"package_mappings"`

    // 高级配置
    Advanced AdvancedConfig `json:"advanced" yaml:"advanced"`
}

type BlacklistConfig struct {
    // 包路径黑名单
    Packages []string `json:"packages" yaml:"packages"`

    // 类型名称黑名单
    Types []string `json:"types" yaml:"types"`

    // 方法名称黑名单
    Methods []string `json:"methods" yaml:"methods"`

    // 使用正则表达式匹配
    UseRegex bool `json:"use_regex" yaml:"use_regex"`

    // 正则表达式模式
    Patterns []string `json:"patterns" yaml:"patterns"`
}

type AdvancedConfig struct {
    // 是否生成调试信息
    Debug bool `json:"debug" yaml:"debug"`

    // 是否保留原始注释
    KeepComments bool `json:"keep_comments" yaml:"keep_comments"`

    // 是否生成测试文件
    GenerateTests bool `json:"generate_tests" yaml:"generate_tests"`

    // 自定义模板路径
    TemplatePath string `json:"template_path" yaml:"template_path"`

    // 缓存配置
    Cache CacheConfig `json:"cache" yaml:"cache"`
}

type CacheConfig struct {
    // 是否启用缓存
    Enabled bool `json:"enabled" yaml:"enabled"`

    // 缓存目录
    Directory string `json:"directory" yaml:"directory"`

    // 缓存过期时间（秒）
    TTL int64 `json:"ttl" yaml:"ttl"`

    // 最大缓存大小（MB）
    MaxSize int64 `json:"max_size" yaml:"max_size"`
}
```

#### 4.2.3 配置验证器

```go
// config/validator.go
type ConfigValidator interface {
    ValidateConfig(config *GeneratorConfig) []ValidationError
    ValidateBlacklist(config *BlacklistConfig) []ValidationError
    ValidatePackageMappings(mappings map[string]string) []ValidationError
}

type ValidationError struct {
    Field   string
    Message string
    Value   interface{}
}

// config/blacklist.go
type BlacklistManager interface {
    IsPackageBlacklisted(pkgPath string) bool
    IsTypeBlacklisted(typeName string) bool
    IsMethodBlacklisted(methodName string) bool
    AddPackageToBlacklist(pkgPath string) error
    RemovePackageFromBlacklist(pkgPath string) error
    ClearBlacklist()
}

// config/package_mapping.go
type PackageMappingManager interface {
    GetMapping(sourcePkg string) (targetPkg string, exists bool)
    SetMapping(sourcePkg, targetPkg string) error
    RemoveMapping(sourcePkg string) error
    GetAllMappings() map[string]string
    ValidateMapping(sourcePkg, targetPkg string) error
}
```

#### 4.2.4 类型信息

```go
// core/types.go
type TypeInfo struct {
    Type        reflect.Type
    PackagePath string
    PackageName string
    TypeName    string
    IsPointer   bool
    IsInterface bool
    IsStruct    bool
    IsFunction  bool
    Fields      []FieldInfo
    Methods     []MethodInfo
    Imports     []ImportInfo
    CacheKey    string
    // 新增：配置相关信息
    Config      *TypeConfig
}

type TypeConfig struct {
    // 是否被黑名单阻止
    IsBlacklisted bool

    // 包前缀
    PackagePrefix string

    // 映射的目标包
    MappedPackage string

    // 自定义配置
    Custom map[string]interface{}
}

type FieldInfo struct {
    Name       string
    Type       *TypeInfo
    IsExported bool
    Tag        reflect.StructTag
}

type MethodInfo struct {
    Name        string
    Parameters  []ParameterInfo
    Returns     []TypeInfo
    IsVariadic  bool
    IsExported  bool
    Receiver    *TypeInfo
}

type ParameterInfo struct {
    Name string
    Type *TypeInfo
    Index int
}
```

#### 4.2.5 分析器接口

```go
// core/interfaces.go
type TypeAnalyzer interface {
    AnalyzeType(t reflect.Type) (*TypeInfo, error)
    AnalyzeFunction(fn any) (*FunctionInfo, error)
    AnalyzeMethod(m reflect.Method) (*MethodInfo, error)
    GetCachedType(key string) (*TypeInfo, bool)
    CacheType(key string, info *TypeInfo)
    // 新增：配置相关方法
    ApplyConfig(config *GeneratorConfig) error
    IsTypeAllowed(typeInfo *TypeInfo) bool
}

type PackageAnalyzer interface {
    AnalyzeImports(types []*TypeInfo) ([]ImportInfo, error)
    ResolveAlias(pkgPath string) string
    CollectDependencies(typeInfo *TypeInfo) ([]ImportInfo, error)
    // 新增：配置相关方法
    GetPackagePrefix(pkgPath string) string
    GetMappedPackage(sourcePkg string) string
}

type CodeGenerator interface {
    Generate(ctx *GeneratorContext, info interface{}) (string, error)
    GenerateFunction(ctx *GeneratorContext, fn *FunctionInfo) (string, error)
    GenerateClass(ctx *GeneratorContext, class *TypeInfo) (string, error)
    GenerateMethod(ctx *GeneratorContext, method *MethodInfo) (string, error)
    // 新增：配置相关方法
    ApplyPackageConfig(ctx *GeneratorContext, pkgPath string) error
}

type TypeConverter interface {
    ConvertParameter(param ParameterInfo) (string, error)
    ConvertReturn(ret TypeInfo) (string, error)
    ConvertField(field FieldInfo) (string, error)
    GetConversionStrategy(t *TypeInfo) (ConversionStrategy, error)
    // 新增：配置相关方法
    ApplyTypeConfig(ctx *GeneratorContext, typeInfo *TypeInfo) error
}

type FileEmitter interface {
    EmitFile(pkgName, fileName, content string) error
    EmitLoadFile(pkgName string, functions []string, classes []string) error
    CreateDirectory(path string) error
    FileExists(path string) bool
}
```

### 4.3 策略模式设计

#### 4.3.1 转换策略

```go
// converter/strategies/base.go
type ConversionStrategy interface {
    CanConvert(t *TypeInfo) bool
    Convert(ctx *ConversionContext) (string, error)
    GetPriority() int
}

type ConversionContext struct {
    Type       *TypeInfo
    Index      int
    Name       string
    Context    *GeneratorContext
    Converter  TypeConverter
}
```

#### 4.3.2 具体策略实现

```go
// converter/strategies/basic_types.go
type BasicTypeStrategy struct{}

func (s *BasicTypeStrategy) CanConvert(t *TypeInfo) bool {
    return t.Type.PkgPath() == "" && isBasicType(t.Type.Kind())
}

func (s *BasicTypeStrategy) Convert(ctx *ConversionContext) (string, error) {
    // 实现基础类型转换逻辑
}

// converter/strategies/struct_types.go
type StructTypeStrategy struct{}

func (s *StructTypeStrategy) CanConvert(t *TypeInfo) bool {
    return t.IsStruct
}

func (s *StructTypeStrategy) Convert(ctx *ConversionContext) (string, error) {
    // 实现结构体类型转换逻辑
}
```

### 4.4 模板化设计

#### 4.4.1 模板定义

```go
// generator/template_gen.go
type TemplateData struct {
    PackageName    string
    ClassName      string
    FunctionName   string
    Parameters     []ParameterTemplateData
    Returns        []ReturnTemplateData
    Fields         []FieldTemplateData
    Methods        []MethodTemplateData
    Imports        []ImportTemplateData
    Context        *GeneratorContext
    // 新增：配置相关数据
    Config         *GeneratorConfig
    PackageConfig  *PackageConfig
}

type PackageConfig struct {
    Prefix   string
    MappedTo string
    IsBlacklisted bool
}

type TemplateGenerator interface {
    GenerateFunction(data *TemplateData) (string, error)
    GenerateClass(data *TemplateData) (string, error)
    GenerateMethod(data *TemplateData) (string, error)
    LoadTemplate(name string) (*template.Template, error)
}
```

### 4.5 配置示例

#### 4.5.1 JSON 配置文件示例

```json
{
  "global_prefix": "origami",
  "output_root": "generated",
  "max_depth": 3,
  "parallel": true,
  "verbose": false,

  "blacklist": {
    "packages": ["internal/*", "vendor/*", "github.com/private/*"],
    "types": ["InternalType", "PrivateStruct"],
    "methods": ["internalMethod", "privateFunc"],
    "use_regex": true,
    "patterns": [".*internal.*", ".*private.*"]
  },

  "package_prefixes": {
    "database/sql": "dbsql",
    "net/http": "nethttp",
    "encoding/json": "json",
    "github.com/your-org/your-pkg": "yourpkg"
  },

  "package_mappings": {
    "database/sql": "github.com/your-org/origami-sql",
    "net/http": "github.com/your-org/origami-http",
    "encoding/json": "github.com/your-org/origami-json"
  },

  "advanced": {
    "debug": false,
    "keep_comments": true,
    "generate_tests": false,
    "template_path": "./templates",
    "cache": {
      "enabled": true,
      "directory": "./cache",
      "ttl": 3600,
      "max_size": 100
    }
  }
}
```

#### 4.5.2 YAML 配置文件示例

```yaml
global_prefix: origami
output_root: generated
max_depth: 3
parallel: true
verbose: false

blacklist:
  packages:
    - "internal/*"
    - "vendor/*"
    - "github.com/private/*"
  types:
    - "InternalType"
    - "PrivateStruct"
  methods:
    - "internalMethod"
    - "privateFunc"
  use_regex: true
  patterns:
    - ".*internal.*"
    - ".*private.*"

package_prefixes:
  database/sql: dbsql
  net/http: nethttp
  encoding/json: json
  github.com/your-org/your-pkg: yourpkg

package_mappings:
  database/sql: github.com/your-org/origami-sql
  net/http: github.com/your-org/origami-http
  encoding/json: github.com/your-org/origami-json

advanced:
  debug: false
  keep_comments: true
  generate_tests: false
  template_path: ./templates
  cache:
    enabled: true
    directory: ./cache
    ttl: 3600
    max_size: 100
```

## 5. 实施计划

### 5.1 第一阶段：基础架构（1-2 周）

#### 5.1.1 目标

- 建立新的目录结构
- 定义核心接口和类型
- 实现基础的类型分析器
- **新增：实现配置管理系统**

#### 5.1.2 具体任务

1. **创建目录结构**

   - 创建新的包结构
   - 迁移现有文件到新位置
   - 建立模块依赖关系

2. **定义核心类型**

   - 实现 `TypeInfo`、`MethodInfo` 等核心类型
   - 定义分析器和生成器接口
   - 建立错误处理机制
   - **新增：定义配置相关类型**

3. **实现基础分析器**

   - 实现 `TypeAnalyzer` 接口
   - 添加类型缓存机制
   - 实现包路径解析
   - **新增：集成配置管理**

4. **实现配置管理系统**
   - 实现 `ConfigManager` 接口
   - 实现配置加载和验证
   - 实现黑名单管理
   - 实现包映射管理

#### 5.1.3 验收标准

- 新的目录结构建立完成
- 核心接口定义完整
- 基础类型分析功能正常
- **配置管理系统功能完整**

### 5.2 第二阶段：转换器重构（1-2 周）

#### 5.2.1 目标

- 实现统一的类型转换器
- 重构参数和返回值处理
- 实现策略模式
- **新增：集成配置支持**

#### 5.2.2 具体任务

1. **实现转换策略**

   - 实现各种类型的转换策略
   - 建立策略注册机制
   - 实现策略优先级排序
   - **新增：策略支持配置过滤**

2. **重构转换逻辑**

   - 统一参数转换接口
   - 统一返回值转换接口
   - 实现错误统一处理
   - **新增：转换时应用配置**

3. **优化转换性能**
   - 添加转换结果缓存
   - 实现批量转换优化
   - 减少重复计算
   - **新增：缓存配置相关结果**

#### 5.2.3 验收标准

- 所有类型转换策略实现完成
- 转换逻辑统一且高效
- 错误处理机制完善
- **配置支持完整**

### 5.3 第三阶段：生成器重构（1-2 周）

#### 5.3.1 目标

- 实现模板化代码生成
- 重构函数、类、方法生成器
- 优化代码生成性能
- **新增：模板支持配置变量**

#### 5.3.2 具体任务

1. **实现模板系统**

   - 设计代码生成模板
   - 实现模板解析引擎
   - 支持模板自定义
   - **新增：模板支持配置变量**

2. **重构生成器**

   - 实现 `FunctionGenerator`
   - 实现 `ClassGenerator`
   - 实现 `MethodGenerator`
   - **新增：生成器应用配置**

3. **优化生成性能**
   - 实现并行代码生成
   - 添加增量更新机制
   - 优化文件写入性能
   - **新增：配置缓存机制**

#### 5.3.3 验收标准

- 模板系统功能完整
- 代码生成器重构完成
- 性能有明显提升
- **配置集成完整**

### 5.4 第四阶段：输出层重构（1 周）

#### 5.4.1 目标

- 重构文件输出系统
- 优化注册管理机制
- 完善缓存系统
- **新增：输出支持配置过滤**

#### 5.4.2 具体任务

1. **重构文件输出**

   - 实现 `FileEmitter` 接口
   - 优化文件写入逻辑
   - 添加文件验证机制
   - **新增：输出前应用配置**

2. **优化注册管理**

   - 重构 `Registry` 实现
   - 优化注册性能
   - 添加注册验证
   - **新增：注册时过滤黑名单**

3. **完善缓存系统**
   - 实现多级缓存
   - 添加缓存失效机制
   - 优化缓存性能
   - **新增：缓存配置信息**

#### 5.4.3 验收标准

- 文件输出系统稳定可靠
- 注册管理机制完善
- 缓存系统性能良好
- **配置过滤功能完整**

### 5.5 第五阶段：测试和优化（1 周）

#### 5.5.1 目标

- 完善测试覆盖
- 性能优化
- 文档更新
- **新增：配置功能测试**

#### 5.5.2 具体任务

1. **完善测试**

   - 添加单元测试
   - 添加集成测试
   - 添加性能测试
   - **新增：配置功能测试**

2. **性能优化**

   - 优化内存使用
   - 优化 CPU 使用
   - 优化 I/O 性能
   - **新增：配置加载优化**

3. **文档更新**
   - 更新 API 文档
   - 编写使用指南
   - 更新示例代码
   - **新增：配置文档**

#### 5.5.3 验收标准

- 测试覆盖率 > 80%
- 性能指标达到预期
- 文档完整且准确
- **配置功能测试完整**

## 6. 风险评估与应对

### 6.1 技术风险

#### 6.1.1 向后兼容性风险

- **风险**：重构可能破坏现有 API
- **应对**：保持主要接口不变，逐步迁移
- **缓解**：提供迁移指南和兼容性层

#### 6.1.2 性能风险

- **风险**：新架构可能影响性能
- **应对**：持续性能测试和优化
- **缓解**：添加性能监控和基准测试

#### 6.1.3 复杂性风险

- **风险**：新架构可能过于复杂
- **应对**：保持简单设计原则
- **缓解**：充分的文档和示例

#### 6.1.4 **配置复杂性风险**

- **风险**：配置系统可能过于复杂
- **应对**：提供默认配置和配置向导
- **缓解**：配置验证和错误提示

### 6.2 项目风险

#### 6.2.1 时间风险

- **风险**：重构时间可能超出预期
- **应对**：分阶段实施，及时调整计划
- **缓解**：设置里程碑和检查点

#### 6.2.2 质量风险

- **风险**：重构可能引入新 bug
- **应对**：充分的测试和代码审查
- **缓解**：渐进式重构，及时验证

### 6.3 应对策略

#### 6.3.1 渐进式重构

- 保持现有功能稳定
- 逐步迁移到新架构
- 及时验证和反馈

#### 6.3.2 充分测试

- 保持现有测试通过
- 添加新的测试用例
- 持续集成测试

#### 6.3.3 文档先行

- 详细记录重构过程
- 及时更新文档
- 提供迁移指南

#### 6.3.4 **配置管理策略**

- 提供配置模板和示例
- 实现配置验证和错误提示
- 支持配置热重载

## 7. 成功标准

### 7.1 技术指标

- 代码复杂度降低 30%
- 测试覆盖率 > 80%
- 性能提升 20%
- 内存使用减少 15%
- **配置系统响应时间 < 100ms**

### 7.2 质量指标

- 代码重复率 < 5%
- 函数长度 < 100 行
- 圈复杂度 < 10
- 错误率降低 50%
- **配置错误率 < 1%**

### 7.3 维护性指标

- 新功能开发时间减少 30%
- Bug 修复时间减少 40%
- 代码审查时间减少 25%
- 文档完整性 > 90%
- **配置维护时间减少 60%**

### 7.4 **新增：配置指标**

- 支持配置格式：JSON, YAML, TOML
- 配置验证准确率 > 99%
- 配置热重载支持
- 配置模板覆盖率 > 95%

## 8. 总结

本次重构将显著提升 Generator 项目的可维护性、可扩展性和性能。通过建立清晰的分层架构，实现单一职责原则，采用策略模式和模板化设计，将使代码更加模块化和易于维护。

**特别强调配置管理的重要性**：

- 黑名单功能可以精确控制生成的包和类型
- 包前缀配置可以避免命名冲突
- 依赖包映射可以支持自定义包路径
- 配置系统提供灵活的定制能力

重构过程将分五个阶段进行，每个阶段都有明确的目标和验收标准。我们将采用渐进式重构策略，确保在重构过程中保持系统的稳定性，并通过充分的测试和文档来降低风险。

最终，我们期望通过这次重构，使 Generator 项目能够更好地支持未来的功能扩展，为 `github.com/php-any/origami` 生态提供更强大、更可靠的代码生成能力。
