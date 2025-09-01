package scr

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/php-any/origami/data"
)

// isExportedName 检查名称是否为导出名称（首字母大写）
func isExportedName(name string) bool {
	if name == "" {
		return false
	}
	return strings.ToUpper(name[:1]) == name[:1]
}

// isPtrToStruct 检查是否为指向结构体的指针
func isPtrToStruct(t reflect.Type) bool {
	return t.Kind() == reflect.Pointer && t.Elem() != nil && t.Elem().Kind() == reflect.Struct
}

// isBuiltinErrorType 判断是否为内建 error 接口
func isBuiltinErrorType(t reflect.Type) bool {
	return t != nil && t.Kind() == reflect.Interface && t.PkgPath() == "" && t.Name() == "error"
}

// checkMethodRecursiveGeneration 检查方法的参数和返回值是否需要递归生成
// 用于 buildClass 中的方法检查
func checkMethodRecursiveGeneration(m reflect.Method, cache *GroupCache) {
	// 检查方法返回值
	for oi := 0; oi < m.Type.NumOut(); oi++ {
		outType := m.Type.Out(oi)
		if isBuiltinErrorType(outType) { // 跳过内建 error
			continue
		}
		if isTypeNeedsProxy(outType) {
			_ = generateFromType(outType, cache, nil)
		}
	}

	// 检查方法参数（跳过第一个参数，通常是接收者）
	for ii := 1; ii < m.Type.NumIn(); ii++ {
		paramType := m.Type.In(ii)
		if isPtrToStruct(paramType) {
			_ = generateFromType(paramType, cache, nil)
		}
	}
}

// checkFunctionRecursiveGeneration 检查函数的参数和返回值是否需要递归生成
// 用于 buildFunc 中的函数检查
func checkFunctionRecursiveGeneration(t reflect.Type, cache *GroupCache) {
	// 检查函数返回值
	for i := 0; i < t.NumOut(); i++ {
		outType := t.Out(i)
		if isBuiltinErrorType(outType) { // 跳过内建 error
			continue
		}
		if isTypeNeedsProxy(outType) {
			_ = generateFromType(outType, cache, nil)
		}
	}

	// 检查函数参数
	for i := 0; i < t.NumIn(); i++ {
		paramType := t.In(i)
		if isPtrToStruct(paramType) {
			_ = generateFromType(paramType, cache, nil)
		}
	}
}

// ConvertFromIndex 泛型函数，从 Context 中安全获取指定索引的参数并转换为目标类型
func ConvertFromIndex[T any](ctx data.Context, index int) (T, error) {
	var zero T

	value, has := ctx.GetIndexValue(index)
	if !has {
		return zero, fmt.Errorf("缺少参数, index: %d", index)
	}

	// 尝试类型断言
	if converted, ok := value.(T); ok {
		return converted, nil
	}

	// 如果直接断言失败，尝试通过反射进行转换
	targetType := reflect.TypeOf(zero)
	valueType := reflect.TypeOf(value)

	// 如果类型完全匹配
	if valueType == targetType {
		return value.(T), nil
	}

	// 处理指针类型转换
	if targetType.Kind() == reflect.Ptr && valueType.Kind() == reflect.Ptr {
		if targetType.Elem() == valueType.Elem() {
			return value.(T), nil
		}
	}

	// 处理接口类型转换
	if targetType.Kind() == reflect.Interface {
		if valueType.Implements(targetType) {
			return value.(T), nil
		}
	}

	return zero, fmt.Errorf("参数类型转换失败, index: %d, 期望类型: %s, 实际类型: %s",
		index, targetType.String(), valueType.String())
}
