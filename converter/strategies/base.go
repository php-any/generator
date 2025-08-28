package strategies

import (
	"github.com/php-any/generator/core"
)

// ConversionStrategy 转换策略接口
type ConversionStrategy interface {
	// CanConvert 检查是否可以转换指定类型
	CanConvert(t *core.TypeInfo) bool

	// Convert 执行类型转换
	Convert(ctx *ConversionContext) (string, error)

	// GetPriority 获取策略优先级（数字越小优先级越高）
	GetPriority() int

	// GetName 获取策略名称
	GetName() string
}

// ConversionContext 转换上下文
type ConversionContext struct {
	// Type 要转换的类型信息
	Type *core.TypeInfo

	// Index 参数或返回值的索引
	Index int

	// Name 参数或字段的名称
	Name string

	// Context 生成器上下文
	Context *core.GeneratorContext

	// Converter 类型转换器实例
	Converter interface{}

	// Options 转换选项
	Options *ConversionOptions
}

// ConversionOptions 转换选项
type ConversionOptions struct {
	// 是否生成指针类型
	UsePointer bool

	// 是否生成接口类型
	UseInterface bool

	// 包别名映射
	PackageAliases map[string]string

	// 包映射
	PackageMappings map[string]string

	// 类型映射
	TypeMappings map[string]string

	// 是否包含包前缀
	IncludePackagePrefix bool
}

// NewConversionOptions 创建新的转换选项
func NewConversionOptions() *ConversionOptions {
	return &ConversionOptions{
		PackageAliases:       make(map[string]string),
		TypeMappings:         make(map[string]string),
		IncludePackagePrefix: true,
	}
}

// BaseStrategy 基础策略实现
type BaseStrategy struct {
	name     string
	priority int
}

// NewBaseStrategy 创建新的基础策略
func NewBaseStrategy(name string, priority int) *BaseStrategy {
	return &BaseStrategy{
		name:     name,
		priority: priority,
	}
}

// GetName 获取策略名称
func (bs *BaseStrategy) GetName() string {
	return bs.name
}

// GetPriority 获取策略优先级
func (bs *BaseStrategy) GetPriority() int {
	return bs.priority
}

// StrategyRegistry 策略注册器
type StrategyRegistry struct {
	strategies []ConversionStrategy
}

// NewStrategyRegistry 创建新的策略注册器
func NewStrategyRegistry() *StrategyRegistry {
	return &StrategyRegistry{
		strategies: make([]ConversionStrategy, 0),
	}
}

// RegisterStrategy 注册转换策略
func (sr *StrategyRegistry) RegisterStrategy(strategy ConversionStrategy) {
	sr.strategies = append(sr.strategies, strategy)

	// 按优先级排序
	sr.sortStrategies()
}

// GetStrategy 获取适合的策略
func (sr *StrategyRegistry) GetStrategy(t *core.TypeInfo) (ConversionStrategy, error) {
	for _, strategy := range sr.strategies {
		if strategy.CanConvert(t) {
			return strategy, nil
		}
	}

	return nil, core.NewGeneratorError(core.ErrCodeTypeConversion,
		"no suitable conversion strategy found for type: "+t.TypeName, nil)
}

// GetAllStrategies 获取所有策略
func (sr *StrategyRegistry) GetAllStrategies() []ConversionStrategy {
	return sr.strategies
}

// sortStrategies 按优先级排序策略
func (sr *StrategyRegistry) sortStrategies() {
	// 使用简单的冒泡排序
	for i := 0; i < len(sr.strategies)-1; i++ {
		for j := 0; j < len(sr.strategies)-i-1; j++ {
			if sr.strategies[j].GetPriority() > sr.strategies[j+1].GetPriority() {
				sr.strategies[j], sr.strategies[j+1] = sr.strategies[j+1], sr.strategies[j]
			}
		}
	}
}

// StrategyFactory 策略工厂
type StrategyFactory struct {
	registry *StrategyRegistry
}

// NewStrategyFactory 创建新的策略工厂
func NewStrategyFactory() *StrategyFactory {
	return &StrategyFactory{
		registry: NewStrategyRegistry(),
	}
}

// RegisterDefaultStrategies 注册默认策略
func (sf *StrategyFactory) RegisterDefaultStrategies() {
	// 注册基础类型策略
	sf.registry.RegisterStrategy(NewBasicTypeStrategy())

	// 注册结构体类型策略
	sf.registry.RegisterStrategy(NewStructTypeStrategy())

	// 注册接口类型策略
	sf.registry.RegisterStrategy(NewInterfaceTypeStrategy())

	// 注册函数类型策略
	sf.registry.RegisterStrategy(NewFunctionTypeStrategy())

	// 注册切片类型策略
	sf.registry.RegisterStrategy(NewSliceTypeStrategy())

	// 注册映射类型策略
	sf.registry.RegisterStrategy(NewMapTypeStrategy())

	// 注册通道类型策略
	sf.registry.RegisterStrategy(NewChannelTypeStrategy())

	// 注册指针类型策略
	sf.registry.RegisterStrategy(NewPointerTypeStrategy())
}

// GetStrategy 获取策略
func (sf *StrategyFactory) GetStrategy(t *core.TypeInfo) (ConversionStrategy, error) {
	return sf.registry.GetStrategy(t)
}

// GetRegistry 获取策略注册器
func (sf *StrategyFactory) GetRegistry() *StrategyRegistry {
	return sf.registry
}
