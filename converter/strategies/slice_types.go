package strategies

import (
	"reflect"
	"strings"

	"github.com/php-any/generator/core"
)

// SliceTypeStrategy 切片类型转换策略
type SliceTypeStrategy struct {
	*BaseStrategy
}

// NewSliceTypeStrategy 创建新的切片类型策略
func NewSliceTypeStrategy() *SliceTypeStrategy {
	return &SliceTypeStrategy{
		BaseStrategy: NewBaseStrategy("SliceType", 25),
	}
}

// CanConvert 检查是否可以转换切片类型
func (sts *SliceTypeStrategy) CanConvert(t *core.TypeInfo) bool {
	if t == nil || t.Type == nil {
		return false
	}

	return t.Type.Kind() == reflect.Slice
}

// Convert 转换切片类型
func (sts *SliceTypeStrategy) Convert(ctx *ConversionContext) (string, error) {
	if ctx.Type == nil || ctx.Type.Type == nil {
		return "", core.NewGeneratorError(core.ErrCodeTypeConversion, "type is nil", nil)
	}

	// 获取元素类型
	elemType := ctx.Type.Type.Elem()

	// 创建元素类型信息
	elemTypeInfo := &core.TypeInfo{
		Type:        elemType,
		PackagePath: elemType.PkgPath(),
		PackageName: sts.getPackageName(elemType.PkgPath()),
		TypeName:    elemType.Name(),
		IsPointer:   elemType.Kind() == reflect.Ptr,
		IsInterface: elemType.Kind() == reflect.Interface,
		IsStruct:    elemType.Kind() == reflect.Struct,
		IsFunction:  elemType.Kind() == reflect.Func,
		CacheKey:    elemType.String(),
	}

	// 递归转换元素类型
	elemContext := &ConversionContext{
		Type:      elemTypeInfo,
		Index:     ctx.Index,
		Name:      ctx.Name,
		Context:   ctx.Context,
		Converter: ctx.Converter,
		Options:   ctx.Options,
	}

	// 获取适合的策略
	if ctx.Converter != nil {
		if converter, ok := ctx.Converter.(interface {
			GetStrategy(t *core.TypeInfo) (interface{}, error)
		}); ok {
			if strategy, err := converter.GetStrategy(elemTypeInfo); err == nil {
				if convStrategy, ok := strategy.(ConversionStrategy); ok {
					if elemTypeStr, err := convStrategy.Convert(elemContext); err == nil {
						return "[]" + elemTypeStr, nil
					}
				}
			}
		}
	}

	// 如果无法获取策略，使用默认转换
	elemTypeStr := sts.getDefaultTypeString(elemType)
	return "[]" + elemTypeStr, nil
}

// getPackageName 获取包名
func (sts *SliceTypeStrategy) getPackageName(pkgPath string) string {
	if pkgPath == "" {
		return ""
	}

	// 提取包名
	parts := []string{}
	for _, part := range strings.Split(pkgPath, "/") {
		if part != "" {
			parts = append(parts, part)
		}
	}

	if len(parts) == 0 {
		return ""
	}

	return parts[len(parts)-1]
}

// getDefaultTypeString 获取默认类型字符串
func (sts *SliceTypeStrategy) getDefaultTypeString(t reflect.Type) string {
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
		return "*" + sts.getDefaultTypeString(t.Elem())
	case reflect.Struct:
		if t.PkgPath() != "" {
			return t.PkgPath() + "." + t.Name()
		}
		return t.Name()
	default:
		return t.String()
	}
}
