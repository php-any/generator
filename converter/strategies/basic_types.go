package strategies

import (
	"reflect"

	"github.com/php-any/generator/core"
)

// BasicTypeStrategy 基础类型转换策略
type BasicTypeStrategy struct {
	*BaseStrategy
}

// NewBasicTypeStrategy 创建新的基础类型策略
func NewBasicTypeStrategy() *BasicTypeStrategy {
	return &BasicTypeStrategy{
		BaseStrategy: NewBaseStrategy("BasicType", 1),
	}
}

// CanConvert 检查是否可以转换基础类型
func (bts *BasicTypeStrategy) CanConvert(t *core.TypeInfo) bool {
	if t == nil || t.Type == nil {
		return false
	}

	// 检查是否为基本类型
	return bts.isBasicType(t.Type.Kind())
}

// Convert 转换基础类型
func (bts *BasicTypeStrategy) Convert(ctx *ConversionContext) (string, error) {
	if ctx.Type == nil || ctx.Type.Type == nil {
		return "", core.NewGeneratorError(core.ErrCodeTypeConversion, "type is nil", nil)
	}

	kind := ctx.Type.Type.Kind()

	// 根据类型种类返回对应的Go类型
	switch kind {
	case reflect.Bool:
		return "bool", nil
	case reflect.Int:
		return "int", nil
	case reflect.Int8:
		return "int8", nil
	case reflect.Int16:
		return "int16", nil
	case reflect.Int32:
		return "int32", nil
	case reflect.Int64:
		return "int64", nil
	case reflect.Uint:
		return "uint", nil
	case reflect.Uint8:
		return "uint8", nil
	case reflect.Uint16:
		return "uint16", nil
	case reflect.Uint32:
		return "uint32", nil
	case reflect.Uint64:
		return "uint64", nil
	case reflect.Uintptr:
		return "uintptr", nil
	case reflect.Float32:
		return "float32", nil
	case reflect.Float64:
		return "float64", nil
	case reflect.Complex64:
		return "complex64", nil
	case reflect.Complex128:
		return "complex128", nil
	case reflect.String:
		return "string", nil
	case reflect.Interface:
		return "interface{}", nil
	default:
		return "", core.NewGeneratorError(core.ErrCodeTypeConversion,
			"unsupported basic type: "+kind.String(), nil)
	}
}

// isBasicType 检查是否为基本类型
func (bts *BasicTypeStrategy) isBasicType(kind reflect.Kind) bool {
	switch kind {
	case reflect.Bool, reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr,
		reflect.Float32, reflect.Float64, reflect.Complex64, reflect.Complex128,
		reflect.String, reflect.Interface:
		return true
	default:
		return false
	}
}

// NumericTypeStrategy 数值类型转换策略
type NumericTypeStrategy struct {
	*BaseStrategy
}

// NewNumericTypeStrategy 创建新的数值类型策略
func NewNumericTypeStrategy() *NumericTypeStrategy {
	return &NumericTypeStrategy{
		BaseStrategy: NewBaseStrategy("NumericType", 2),
	}
}

// CanConvert 检查是否可以转换数值类型
func (nts *NumericTypeStrategy) CanConvert(t *core.TypeInfo) bool {
	if t == nil || t.Type == nil {
		return false
	}

	return nts.isNumericType(t.Type.Kind())
}

// Convert 转换数值类型
func (nts *NumericTypeStrategy) Convert(ctx *ConversionContext) (string, error) {
	if ctx.Type == nil || ctx.Type.Type == nil {
		return "", core.NewGeneratorError(core.ErrCodeTypeConversion, "type is nil", nil)
	}

	kind := ctx.Type.Type.Kind()

	// 根据类型种类返回对应的Go类型
	switch kind {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return "int", nil
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		return "uint", nil
	case reflect.Float32, reflect.Float64:
		return "float64", nil
	case reflect.Complex64, reflect.Complex128:
		return "complex128", nil
	default:
		return "", core.NewGeneratorError(core.ErrCodeTypeConversion,
			"unsupported numeric type: "+kind.String(), nil)
	}
}

// isNumericType 检查是否为数值类型
func (nts *NumericTypeStrategy) isNumericType(kind reflect.Kind) bool {
	switch kind {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr,
		reflect.Float32, reflect.Float64, reflect.Complex64, reflect.Complex128:
		return true
	default:
		return false
	}
}

// StringTypeStrategy 字符串类型转换策略
type StringTypeStrategy struct {
	*BaseStrategy
}

// NewStringTypeStrategy 创建新的字符串类型策略
func NewStringTypeStrategy() *StringTypeStrategy {
	return &StringTypeStrategy{
		BaseStrategy: NewBaseStrategy("StringType", 3),
	}
}

// CanConvert 检查是否可以转换字符串类型
func (sts *StringTypeStrategy) CanConvert(t *core.TypeInfo) bool {
	if t == nil || t.Type == nil {
		return false
	}

	return t.Type.Kind() == reflect.String
}

// Convert 转换字符串类型
func (sts *StringTypeStrategy) Convert(ctx *ConversionContext) (string, error) {
	return "string", nil
}

// BoolTypeStrategy 布尔类型转换策略
type BoolTypeStrategy struct {
	*BaseStrategy
}

// NewBoolTypeStrategy 创建新的布尔类型策略
func NewBoolTypeStrategy() *BoolTypeStrategy {
	return &BoolTypeStrategy{
		BaseStrategy: NewBaseStrategy("BoolType", 4),
	}
}

// CanConvert 检查是否可以转换布尔类型
func (bts *BoolTypeStrategy) CanConvert(t *core.TypeInfo) bool {
	if t == nil || t.Type == nil {
		return false
	}

	return t.Type.Kind() == reflect.Bool
}

// Convert 转换布尔类型
func (bts *BoolTypeStrategy) Convert(ctx *ConversionContext) (string, error) {
	return "bool", nil
}
