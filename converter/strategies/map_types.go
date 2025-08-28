package strategies

import (
	"reflect"

	"github.com/php-any/generator/core"
)

// MapTypeStrategy 映射类型转换策略
type MapTypeStrategy struct {
	*BaseStrategy
}

// NewMapTypeStrategy 创建新的映射类型策略
func NewMapTypeStrategy() *MapTypeStrategy {
	return &MapTypeStrategy{
		BaseStrategy: NewBaseStrategy("MapType", 30),
	}
}

// CanConvert 检查是否可以转换映射类型
func (mts *MapTypeStrategy) CanConvert(t *core.TypeInfo) bool {
	if t == nil || t.Type == nil {
		return false
	}

	return t.Type.Kind() == reflect.Map
}

// Convert 转换映射类型
func (mts *MapTypeStrategy) Convert(ctx *ConversionContext) (string, error) {
	if ctx.Type == nil || ctx.Type.Type == nil {
		return "", core.NewGeneratorError(core.ErrCodeTypeConversion, "type is nil", nil)
	}

	// 获取键和值类型
	keyType := ctx.Type.Type.Key()
	valueType := ctx.Type.Type.Elem()

	// 转换键类型
	keyTypeStr := mts.getDefaultTypeString(keyType)

	// 转换值类型
	valueTypeStr := mts.getDefaultTypeString(valueType)

	return "map[" + keyTypeStr + "]" + valueTypeStr, nil
}

// getDefaultTypeString 获取默认类型字符串
func (mts *MapTypeStrategy) getDefaultTypeString(t reflect.Type) string {
	if t == nil {
		return "interface{}"
	}

	switch t.Kind() {
	case reflect.Bool:
		return "bool"
	case reflect.Int:
		return "int"
	case reflect.Int8:
		return "int8"
	case reflect.Int16:
		return "int16"
	case reflect.Int32:
		return "int32"
	case reflect.Int64:
		return "int64"
	case reflect.Uint:
		return "uint"
	case reflect.Uint8:
		return "uint8"
	case reflect.Uint16:
		return "uint16"
	case reflect.Uint32:
		return "uint32"
	case reflect.Uint64:
		return "uint64"
	case reflect.Uintptr:
		return "uintptr"
	case reflect.Float32:
		return "float32"
	case reflect.Float64:
		return "float64"
	case reflect.Complex64:
		return "complex64"
	case reflect.Complex128:
		return "complex128"
	case reflect.String:
		return "string"
	case reflect.Interface:
		return "interface{}"
	case reflect.Ptr:
		return "*" + mts.getDefaultTypeString(t.Elem())
	case reflect.Struct:
		if t.PkgPath() != "" {
			return t.PkgPath() + "." + t.Name()
		}
		return t.Name()
	default:
		return t.String()
	}
}
