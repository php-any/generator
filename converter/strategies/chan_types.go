package strategies

import (
	"reflect"

	"github.com/php-any/generator/core"
)

// ChannelTypeStrategy 通道类型转换策略
type ChannelTypeStrategy struct {
	*BaseStrategy
}

// NewChannelTypeStrategy 创建新的通道类型策略
func NewChannelTypeStrategy() *ChannelTypeStrategy {
	return &ChannelTypeStrategy{
		BaseStrategy: NewBaseStrategy("ChannelType", 35),
	}
}

// CanConvert 检查是否可以转换通道类型
func (cts *ChannelTypeStrategy) CanConvert(t *core.TypeInfo) bool {
	if t == nil || t.Type == nil {
		return false
	}

	return t.Type.Kind() == reflect.Chan
}

// Convert 转换通道类型
func (cts *ChannelTypeStrategy) Convert(ctx *ConversionContext) (string, error) {
	if ctx.Type == nil || ctx.Type.Type == nil {
		return "", core.NewGeneratorError(core.ErrCodeTypeConversion, "type is nil", nil)
	}

	// 获取元素类型
	elemType := ctx.Type.Type.Elem()

	// 获取通道方向
	chanDir := ctx.Type.Type.ChanDir()

	// 转换元素类型
	elemTypeStr := cts.getDefaultTypeString(elemType)

	// 根据方向生成通道类型
	switch chanDir {
	case reflect.RecvDir:
		return "<-chan " + elemTypeStr, nil
	case reflect.SendDir:
		return "chan<- " + elemTypeStr, nil
	default:
		return "chan " + elemTypeStr, nil
	}
}

// getDefaultTypeString 获取默认类型字符串
func (cts *ChannelTypeStrategy) getDefaultTypeString(t reflect.Type) string {
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
		return "*" + cts.getDefaultTypeString(t.Elem())
	case reflect.Struct:
		if t.PkgPath() != "" {
			return t.PkgPath() + "." + t.Name()
		}
		return t.Name()
	default:
		return t.String()
	}
}
