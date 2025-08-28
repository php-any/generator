package strategies

import (
	"reflect"
	"strings"

	"github.com/php-any/generator/core"
)

// FunctionTypeStrategy 函数类型转换策略
type FunctionTypeStrategy struct {
	*BaseStrategy
}

// NewFunctionTypeStrategy 创建新的函数类型策略
func NewFunctionTypeStrategy() *FunctionTypeStrategy {
	return &FunctionTypeStrategy{
		BaseStrategy: NewBaseStrategy("FunctionType", 20),
	}
}

// CanConvert 检查是否可以转换函数类型
func (fts *FunctionTypeStrategy) CanConvert(t *core.TypeInfo) bool {
	if t == nil || t.Type == nil {
		return false
	}

	return t.IsFunction
}

// Convert 转换函数类型
func (fts *FunctionTypeStrategy) Convert(ctx *ConversionContext) (string, error) {
	if ctx.Type == nil || ctx.Type.Type == nil {
		return "", core.NewGeneratorError(core.ErrCodeTypeConversion, "type is nil", nil)
	}

	// 生成函数签名
	return fts.generateFunctionSignature(ctx.Type.Type), nil
}

// generateFunctionSignature 生成函数签名
func (fts *FunctionTypeStrategy) generateFunctionSignature(t reflect.Type) string {
	if t.Kind() != reflect.Func {
		return "func"
	}

	var signature strings.Builder
	signature.WriteString("func(")

	// 处理参数
	numIn := t.NumIn()
	for i := 0; i < numIn; i++ {
		if i > 0 {
			signature.WriteString(", ")
		}
		paramType := t.In(i)
		signature.WriteString(fts.getTypeString(paramType))
	}

	signature.WriteString(")")

	// 处理返回值
	numOut := t.NumOut()
	if numOut > 0 {
		if numOut == 1 {
			signature.WriteString(" ")
			signature.WriteString(fts.getTypeString(t.Out(0)))
		} else {
			signature.WriteString(" (")
			for i := 0; i < numOut; i++ {
				if i > 0 {
					signature.WriteString(", ")
				}
				signature.WriteString(fts.getTypeString(t.Out(i)))
			}
			signature.WriteString(")")
		}
	}

	return signature.String()
}

// getTypeString 获取类型的字符串表示
func (fts *FunctionTypeStrategy) getTypeString(t reflect.Type) string {
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
		return "*" + fts.getTypeString(t.Elem())
	case reflect.Slice:
		return "[]" + fts.getTypeString(t.Elem())
	case reflect.Array:
		return "[" + string(rune(t.Len())) + "]" + fts.getTypeString(t.Elem())
	case reflect.Map:
		return "map[" + fts.getTypeString(t.Key()) + "]" + fts.getTypeString(t.Elem())
	case reflect.Chan:
		switch t.ChanDir() {
		case reflect.RecvDir:
			return "<-" + fts.getTypeString(t.Elem())
		case reflect.SendDir:
			return "chan<- " + fts.getTypeString(t.Elem())
		default:
			return "chan " + fts.getTypeString(t.Elem())
		}
	case reflect.Func:
		return "func" // 避免递归
	case reflect.Struct:
		if t.PkgPath() != "" {
			return t.PkgPath() + "." + t.Name()
		}
		return t.Name()
	default:
		return t.String()
	}
}
