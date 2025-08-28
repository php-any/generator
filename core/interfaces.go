package core

import (
	"reflect"
)

// TypeAnalyzer 类型分析器接口
type TypeAnalyzer interface {
	AnalyzeType(t reflect.Type) (*TypeInfo, error)
	AnalyzeFunction(fn any) (*FunctionInfo, error)
	AnalyzeMethod(m reflect.Method) (*MethodInfo, error)
	GetCachedType(key string) (*TypeInfo, bool)
	CacheType(key string, info *TypeInfo)
	// 新增：配置相关方法
	ApplyConfig(config interface{}) error
	IsTypeAllowed(typeInfo *TypeInfo) bool
}

// PackageAnalyzer 包分析器接口
type PackageAnalyzer interface {
	AnalyzeImports(types []*TypeInfo) ([]ImportInfo, error)
	ResolveAlias(pkgPath string) string
	CollectDependencies(typeInfo *TypeInfo) ([]ImportInfo, error)
	// 新增：配置相关方法
	GetPackagePrefix(pkgPath string) string
	GetMappedPackage(sourcePkg string) string
}

// CodeGenerator 代码生成器接口
type CodeGenerator interface {
	Generate(ctx *GeneratorContext, info interface{}) (string, error)
	GenerateFunction(ctx *GeneratorContext, fn *FunctionInfo) (string, error)
	GenerateClass(ctx *GeneratorContext, class *TypeInfo) (string, error)
	GenerateMethod(ctx *GeneratorContext, method *MethodInfo) (string, error)
	// 新增：配置相关方法
	ApplyPackageConfig(ctx *GeneratorContext, pkgPath string) error
}

// TypeConverter 类型转换器接口
type TypeConverter interface {
	ConvertParameter(param ParameterInfo) (string, error)
	ConvertReturn(ret TypeInfo) (string, error)
	ConvertField(field FieldInfo) (string, error)
	GetConversionStrategy(t *TypeInfo) (ConversionStrategy, error)
	// 新增：配置相关方法
	ApplyTypeConfig(ctx *GeneratorContext, typeInfo *TypeInfo) error
}

// FileEmitter 文件输出器接口
type FileEmitter interface {
	EmitFile(pkgName, fileName, content string) error
	EmitLoadFile(pkgName string, functions []string, classes []string) error
	CreateDirectory(path string) error
	FileExists(path string) bool
}

// ConversionStrategy 转换策略接口
type ConversionStrategy interface {
	CanConvert(t *TypeInfo) bool
	Convert(ctx *ConversionContext) (string, error)
	GetPriority() int
}

// ConversionContext 转换上下文
type ConversionContext struct {
	Type      *TypeInfo
	Index     int
	Name      string
	Context   *GeneratorContext
	Converter TypeConverter
}

// TemplateGenerator 模板生成器接口
type TemplateGenerator interface {
	GenerateFunction(data *TemplateData) (string, error)
	GenerateClass(data *TemplateData) (string, error)
	GenerateMethod(data *TemplateData) (string, error)
	LoadTemplate(name string) (interface{}, error)
}

// TemplateData 模板数据
type TemplateData struct {
	PackageName  string
	ClassName    string
	FunctionName string
	Parameters   []ParameterTemplateData
	Returns      []ReturnTemplateData
	Fields       []FieldTemplateData
	Methods      []MethodTemplateData
	Imports      []ImportTemplateData
	Context      *GeneratorContext
	// 新增：配置相关数据
	Config        interface{}
	PackageConfig *PackageConfig
}

// PackageConfig 包配置
type PackageConfig struct {
	Prefix        string
	MappedTo      string
	IsBlacklisted bool
}

// ParameterTemplateData 参数模板数据
type ParameterTemplateData struct {
	Name  string
	Type  string
	Index int
}

// ReturnTemplateData 返回值模板数据
type ReturnTemplateData struct {
	Type  string
	Index int
}

// FieldTemplateData 字段模板数据
type FieldTemplateData struct {
	Name       string
	Type       string
	IsExported bool
	Tag        string
}

// MethodTemplateData 方法模板数据
type MethodTemplateData struct {
	Name       string
	ClassName  string
	Parameters []ParameterTemplateData
	Returns    []ReturnTemplateData
	IsVariadic bool
	IsExported bool
}

// ImportTemplateData 导入模板数据
type ImportTemplateData struct {
	Path     string
	Alias    string
	Used     bool
	Priority int
}

// Registry 注册器接口
type Registry interface {
	RegisterFunction(pkgName, functionName string) error
	RegisterClass(pkgName, className string) error
	RegisterMethod(pkgName, className, methodName string) error
	GetRegisteredFunctions(pkgName string) []string
	GetRegisteredClasses(pkgName string) []string
	GetRegisteredMethods(pkgName, className string) []string
	ClearPackage(pkgName string) error
	Clear() error
}

// Cache 缓存接口
type Cache interface {
	Get(key string) (interface{}, bool)
	Set(key string, value interface{}) error
	Delete(key string) error
	Clear() error
	GetStats() *CacheStats
}

// CacheStats 缓存统计
type CacheStats struct {
	HitCount  int64
	MissCount int64
	Size      int64
	MaxSize   int64
}

// Validator 验证器接口
type Validator interface {
	Validate(value interface{}) []ValidationError
	ValidateConfig(config interface{}) []ValidationError
}

// Logger 日志接口
type Logger interface {
	Debug(msg string, args ...interface{})
	Info(msg string, args ...interface{})
	Warning(msg string, args ...interface{})
	Error(msg string, args ...interface{})
	SetLevel(level string)
	SetVerbose(verbose bool)
}

// MetricsCollector 指标收集器接口
type MetricsCollector interface {
	IncrementCounter(name string, value int64)
	SetGauge(name string, value float64)
	RecordHistogram(name string, value float64)
	GetMetrics() map[string]interface{}
	Reset()
}

// ProgressTracker 进度跟踪器接口
type ProgressTracker interface {
	SetTotal(total int)
	UpdateProgress(current int, message string)
	Finish(message string) error
	IsCancelled() bool
}

// GeneratorPlugin 生成器插件接口
type GeneratorPlugin interface {
	Name() string
	Version() string
	Initialize(ctx *GeneratorContext) error
	BeforeGenerate(ctx *GeneratorContext, info interface{}) error
	AfterGenerate(ctx *GeneratorContext, info interface{}, result string) error
	Cleanup(ctx *GeneratorContext) error
}

// PluginManager 插件管理器接口
type PluginManager interface {
	RegisterPlugin(plugin GeneratorPlugin) error
	UnregisterPlugin(name string) error
	GetPlugin(name string) (GeneratorPlugin, bool)
	GetAllPlugins() []GeneratorPlugin
	InitializePlugins(ctx *GeneratorContext) error
	ExecuteBeforeGenerate(ctx *GeneratorContext, info interface{}) error
	ExecuteAfterGenerate(ctx *GeneratorContext, info interface{}, result string) error
	CleanupPlugins(ctx *GeneratorContext) error
}
