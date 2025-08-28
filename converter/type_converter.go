package converter

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/php-any/generator/core"
)

// TypeConverterImpl 类型转换器实现
type TypeConverterImpl struct {
	context    *core.GeneratorContext
	strategies map[string]core.ConversionStrategy
}

// 创建新的类型转换器
func NewTypeConverter(ctx *core.GeneratorContext) *TypeConverterImpl {
	converter := &TypeConverterImpl{
		context:    ctx,
		strategies: make(map[string]core.ConversionStrategy),
	}

	// 注册默认策略
	converter.registerDefaultStrategies()

	return converter
}

// ConvertType 转换类型
func (tc *TypeConverterImpl) ConvertType(typeInfo *core.TypeInfo) (string, error) {
	if typeInfo == nil {
		return "", core.NewGeneratorError(core.ErrCodeTypeConversion, "cannot convert nil type", nil)
	}

	// 获取转换策略
	strategy := tc.getStrategy(typeInfo)
	if strategy == nil {
		return "", core.NewGeneratorError(core.ErrCodeTypeConversion,
			fmt.Sprintf("no conversion strategy for type: %s", typeInfo.TypeName), nil)
	}

	// 创建转换上下文
	ctx := &core.ConversionContext{
		Type:      typeInfo,
		Context:   tc.context,
		Converter: tc,
	}

	// 执行转换
	result, err := strategy.Convert(ctx)
	if err != nil {
		return "", core.NewGeneratorError(core.ErrCodeTypeConversion,
			fmt.Sprintf("failed to convert type %s: %v", typeInfo.TypeName, err), nil)
	}

	return result, nil
}

// ConvertParameter 转换参数
func (tc *TypeConverterImpl) ConvertParameter(param core.ParameterInfo) (string, error) {
	if param.Type == nil {
		return "", core.NewGeneratorError(core.ErrCodeTypeConversion, "cannot convert nil parameter", nil)
	}

	// 获取参数转换策略
	strategy := tc.getParameterStrategy(param.Type)
	if strategy == nil {
		return "", core.NewGeneratorError(core.ErrCodeTypeConversion,
			fmt.Sprintf("no parameter conversion strategy for type: %s", param.Type.TypeName), nil)
	}

	// 创建转换上下文
	ctx := &core.ConversionContext{
		Type:      param.Type,
		Name:      param.Name,
		Index:     param.Index,
		Context:   tc.context,
		Converter: tc,
	}

	// 执行参数转换
	result, err := strategy.Convert(ctx)
	if err != nil {
		return "", core.NewGeneratorError(core.ErrCodeTypeConversion,
			fmt.Sprintf("failed to convert parameter %s: %v", param.Name, err), nil)
	}

	return result, nil
}

// ConvertReturn 转换返回值
func (tc *TypeConverterImpl) ConvertReturn(ret core.TypeInfo) (string, error) {
	// 获取返回值转换策略
	strategy := tc.getReturnValueStrategy(&ret)
	if strategy == nil {
		return "", core.NewGeneratorError(core.ErrCodeTypeConversion,
			fmt.Sprintf("no return value conversion strategy for type: %s", ret.TypeName), nil)
	}

	// 创建转换上下文
	ctx := &core.ConversionContext{
		Type:      &ret,
		Context:   tc.context,
		Converter: tc,
	}

	// 执行返回值转换
	result, err := strategy.Convert(ctx)
	if err != nil {
		return "", core.NewGeneratorError(core.ErrCodeTypeConversion,
			fmt.Sprintf("failed to convert return value: %v", err), nil)
	}

	return result, nil
}

// ConvertField 转换字段
func (tc *TypeConverterImpl) ConvertField(field core.FieldInfo) (string, error) {
	if field.Type == nil {
		return "", core.NewGeneratorError(core.ErrCodeTypeConversion, "cannot convert nil field", nil)
	}

	// 获取字段转换策略
	strategy := tc.getStrategy(field.Type)
	if strategy == nil {
		return "", core.NewGeneratorError(core.ErrCodeTypeConversion,
			fmt.Sprintf("no field conversion strategy for type: %s", field.Type.TypeName), nil)
	}

	// 创建转换上下文
	ctx := &core.ConversionContext{
		Type:      field.Type,
		Name:      field.Name,
		Context:   tc.context,
		Converter: tc,
	}

	// 执行字段转换
	result, err := strategy.Convert(ctx)
	if err != nil {
		return "", core.NewGeneratorError(core.ErrCodeTypeConversion,
			fmt.Sprintf("failed to convert field %s: %v", field.Name, err), nil)
	}

	return result, nil
}

// RegisterStrategy 注册转换策略
func (tc *TypeConverterImpl) RegisterStrategy(name string, strategy core.ConversionStrategy) {
	tc.strategies[name] = strategy
}

// GetStrategy 获取策略
func (tc *TypeConverterImpl) GetStrategy(name string) core.ConversionStrategy {
	return tc.strategies[name]
}

// GetConversionStrategy 获取转换策略
func (tc *TypeConverterImpl) GetConversionStrategy(t *core.TypeInfo) (core.ConversionStrategy, error) {
	if t == nil {
		return nil, core.NewGeneratorError(core.ErrCodeTypeConversion, "cannot get strategy for nil type", nil)
	}

	strategy := tc.getStrategy(t)
	if strategy == nil {
		return nil, core.NewGeneratorError(core.ErrCodeTypeConversion,
			fmt.Sprintf("no conversion strategy for type: %s", t.TypeName), nil)
	}

	return strategy, nil
}

// ApplyTypeConfig 应用类型配置
func (tc *TypeConverterImpl) ApplyTypeConfig(ctx *core.GeneratorContext, typeInfo *core.TypeInfo) error {
	if typeInfo == nil || ctx == nil {
		return nil
	}

	// 这里可以应用配置到类型信息上
	// 例如设置包前缀、检查黑名单等
	// 具体实现根据需求而定

	return nil
}

// 获取类型转换策略
func (tc *TypeConverterImpl) getStrategy(typeInfo *core.TypeInfo) core.ConversionStrategy {
	// 根据类型特征选择策略
	if typeInfo.IsBasicType() {
		return tc.strategies["basic"]
	}

	if typeInfo.IsStruct {
		return tc.strategies["struct"]
	}

	if typeInfo.IsPointer {
		return tc.strategies["pointer"]
	}

	if typeInfo.IsSliceType() {
		return tc.strategies["slice"]
	}

	if typeInfo.IsMapType() {
		return tc.strategies["map"]
	}

	if typeInfo.IsChanType() {
		return tc.strategies["channel"]
	}

	if typeInfo.IsInterface {
		return tc.strategies["interface"]
	}

	if typeInfo.IsFunction {
		return tc.strategies["function"]
	}

	// 默认策略
	return tc.strategies["default"]
}

// 获取参数转换策略
func (tc *TypeConverterImpl) getParameterStrategy(typeInfo *core.TypeInfo) core.ConversionStrategy {
	// 参数转换可能需要特殊处理
	if typeInfo.IsStruct && typeInfo.PackagePath != "" {
		return tc.strategies["parameter_struct"]
	}

	return tc.getStrategy(typeInfo)
}

// 获取返回值转换策略
func (tc *TypeConverterImpl) getReturnValueStrategy(typeInfo *core.TypeInfo) core.ConversionStrategy {
	// 返回值转换可能需要特殊处理
	if typeInfo.IsStruct && typeInfo.PackagePath != "" {
		return tc.strategies["return_struct"]
	}

	return tc.getStrategy(typeInfo)
}

// 注册默认策略
func (tc *TypeConverterImpl) registerDefaultStrategies() {
	// 基本类型策略
	tc.RegisterStrategy("basic", &BasicTypeStrategy{})

	// 结构体策略
	tc.RegisterStrategy("struct", &StructTypeStrategy{})
	tc.RegisterStrategy("parameter_struct", &ParameterStructStrategy{})
	tc.RegisterStrategy("return_struct", &ReturnStructStrategy{})

	// 指针策略
	tc.RegisterStrategy("pointer", &PointerTypeStrategy{})

	// 切片策略
	tc.RegisterStrategy("slice", &SliceTypeStrategy{})

	// 映射策略
	tc.RegisterStrategy("map", &MapTypeStrategy{})

	// 通道策略
	tc.RegisterStrategy("channel", &ChannelTypeStrategy{})

	// 接口策略
	tc.RegisterStrategy("interface", &InterfaceTypeStrategy{})

	// 函数策略
	tc.RegisterStrategy("function", &FunctionTypeStrategy{})

	// 默认策略
	tc.RegisterStrategy("default", &DefaultTypeStrategy{})
}

// BasicTypeStrategy 基本类型转换策略
type BasicTypeStrategy struct{}

func (s *BasicTypeStrategy) CanConvert(t *core.TypeInfo) bool {
	return t.IsBasicType()
}

func (s *BasicTypeStrategy) Convert(ctx *core.ConversionContext) (string, error) {
	switch ctx.Type.TypeName {
	case "string":
		return "string", nil
	case "int", "int8", "int16", "int32", "int64":
		return "int", nil
	case "uint", "uint8", "uint16", "uint32", "uint64":
		return "uint", nil
	case "float32", "float64":
		return "float", nil
	case "bool":
		return "bool", nil
	case "error":
		return "error", nil
	default:
		// 对于包类型，使用别名
		if ctx.Type.PackagePath != "" {
			alias := s.getPackageAlias(ctx.Type.PackagePath)
			return fmt.Sprintf("%s.%s", alias, ctx.Type.TypeName), nil
		}
		return ctx.Type.TypeName, nil
	}
}

func (s *BasicTypeStrategy) GetPriority() int {
	return 100
}

func (s *BasicTypeStrategy) getPackageAlias(pkgPath string) string {
	if pkgPath == "" {
		return ""
	}
	// 根据 origami 规则，使用 src 后缀
	parts := strings.Split(pkgPath, "/")
	baseName := parts[len(parts)-1]
	return baseName + "src"
}

// StructTypeStrategy 结构体类型转换策略
type StructTypeStrategy struct{}

func (s *StructTypeStrategy) CanConvert(t *core.TypeInfo) bool {
	return t.IsStruct
}

func (s *StructTypeStrategy) Convert(ctx *core.ConversionContext) (string, error) {
	if ctx.Type.PackagePath == "" {
		return ctx.Type.TypeName, nil
	}

	// 获取包别名
	alias := s.getPackageAlias(ctx.Type.PackagePath)
	return fmt.Sprintf("%s.%s", alias, ctx.Type.TypeName), nil
}

func (s *StructTypeStrategy) GetPriority() int {
	return 80
}

func (s *StructTypeStrategy) getPackageAlias(pkgPath string) string {
	if pkgPath == "" {
		return ""
	}
	return "origamisrc"
}

// ParameterStructStrategy 参数结构体策略
type ParameterStructStrategy struct {
	StructTypeStrategy
}

func (s *ParameterStructStrategy) GetPriority() int {
	return 85
}

// ReturnStructStrategy 返回值结构体策略
type ReturnStructStrategy struct {
	StructTypeStrategy
}

func (s *ReturnStructStrategy) GetPriority() int {
	return 85
}

// PointerTypeStrategy 指针类型转换策略
type PointerTypeStrategy struct{}

func (s *PointerTypeStrategy) CanConvert(t *core.TypeInfo) bool {
	return t.IsPointer
}

func (s *PointerTypeStrategy) Convert(ctx *core.ConversionContext) (string, error) {
	// 获取元素类型
	elemType := ctx.Type.GetElementType()
	if elemType == nil {
		return "", fmt.Errorf("pointer type has no element type")
	}

	// 递归转换元素类型
	elemTypeStr, err := ctx.Converter.ConvertParameter(core.ParameterInfo{Type: elemType, Name: ctx.Name})
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("*%s", elemTypeStr), nil
}

func (s *PointerTypeStrategy) GetPriority() int {
	return 60
}

// SliceTypeStrategy 切片类型转换策略
type SliceTypeStrategy struct{}

func (s *SliceTypeStrategy) CanConvert(t *core.TypeInfo) bool {
	return t.IsSliceType()
}

func (s *SliceTypeStrategy) Convert(ctx *core.ConversionContext) (string, error) {
	// 获取元素类型
	elemType := ctx.Type.GetElementType()
	if elemType == nil {
		return "", fmt.Errorf("slice type has no element type")
	}

	// 递归转换元素类型
	elemTypeStr, err := ctx.Converter.ConvertParameter(core.ParameterInfo{Type: elemType, Name: ctx.Name})
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("[]%s", elemTypeStr), nil
}

func (s *SliceTypeStrategy) GetPriority() int {
	return 70
}

// MapTypeStrategy 映射类型转换策略
type MapTypeStrategy struct{}

func (s *MapTypeStrategy) CanConvert(t *core.TypeInfo) bool {
	return t.IsMapType()
}

func (s *MapTypeStrategy) Convert(ctx *core.ConversionContext) (string, error) {
	// 获取键类型和值类型
	keyType := ctx.Type.GetKeyType()
	elemType := ctx.Type.GetElementType()

	if keyType == nil || elemType == nil {
		return "", fmt.Errorf("map type has no key or element type")
	}

	// 递归转换键类型和值类型
	keyTypeStr, err := ctx.Converter.ConvertParameter(core.ParameterInfo{Type: keyType, Name: ctx.Name})
	if err != nil {
		return "", err
	}

	elemTypeStr, err := ctx.Converter.ConvertParameter(core.ParameterInfo{Type: elemType, Name: ctx.Name})
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("map[%s]%s", keyTypeStr, elemTypeStr), nil
}

func (s *MapTypeStrategy) GetPriority() int {
	return 50
}

// ChannelTypeStrategy 通道类型转换策略
type ChannelTypeStrategy struct{}

func (s *ChannelTypeStrategy) CanConvert(t *core.TypeInfo) bool {
	return t.IsChanType()
}

func (s *ChannelTypeStrategy) Convert(ctx *core.ConversionContext) (string, error) {
	// 获取元素类型
	elemType := ctx.Type.GetElementType()
	if elemType == nil {
		return "", fmt.Errorf("channel type has no element type")
	}

	// 递归转换元素类型
	elemTypeStr, err := ctx.Converter.ConvertParameter(core.ParameterInfo{Type: elemType, Name: ctx.Name})
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("chan %s", elemTypeStr), nil
}

func (s *ChannelTypeStrategy) GetPriority() int {
	return 40
}

// InterfaceTypeStrategy 接口类型转换策略
type InterfaceTypeStrategy struct{}

func (s *InterfaceTypeStrategy) CanConvert(t *core.TypeInfo) bool {
	return t.IsInterface
}

func (s *InterfaceTypeStrategy) Convert(ctx *core.ConversionContext) (string, error) {
	if ctx.Type.PackagePath == "" {
		return ctx.Type.TypeName, nil
	}

	// 获取包别名
	alias := s.getPackageAlias(ctx.Type.PackagePath)
	return fmt.Sprintf("%s.%s", alias, ctx.Type.TypeName), nil
}

func (s *InterfaceTypeStrategy) GetPriority() int {
	return 90
}

func (s *InterfaceTypeStrategy) getPackageAlias(pkgPath string) string {
	if pkgPath == "" {
		return ""
	}
	return "origamisrc"
}

// FunctionTypeStrategy 函数类型转换策略
type FunctionTypeStrategy struct{}

func (s *FunctionTypeStrategy) CanConvert(t *core.TypeInfo) bool {
	return t.IsFunction
}

func (s *FunctionTypeStrategy) Convert(ctx *core.ConversionContext) (string, error) {
	if ctx.Type.Type == nil {
		return "func", nil
	}

	t := ctx.Type.Type
	if t.Kind() != reflect.Func {
		return "func", nil
	}

	// 生成简化的函数签名
	var signature strings.Builder
	signature.WriteString("func(")

	// 参数数量
	signature.WriteString(fmt.Sprintf("%d params", t.NumIn()))

	// 返回值数量
	if t.NumOut() > 0 {
		signature.WriteString(fmt.Sprintf(", %d returns", t.NumOut()))
	}

	signature.WriteString(")")

	return signature.String(), nil
}

func (s *FunctionTypeStrategy) GetPriority() int {
	return 30
}

// DefaultTypeStrategy 默认类型转换策略
type DefaultTypeStrategy struct{}

func (s *DefaultTypeStrategy) CanConvert(t *core.TypeInfo) bool {
	return true
}

func (s *DefaultTypeStrategy) Convert(ctx *core.ConversionContext) (string, error) {
	return ctx.Type.TypeName, nil
}

func (s *DefaultTypeStrategy) GetPriority() int {
	return 0
}
