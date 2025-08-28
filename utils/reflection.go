package utils

import (
	"reflect"
	"strings"
)

// ReflectionUtils 反射工具函数集合
type ReflectionUtils struct{}

// NewReflectionUtils 创建新的反射工具实例
func NewReflectionUtils() *ReflectionUtils {
	return &ReflectionUtils{}
}

// GetTypeName 获取类型的完整名称
func (ru *ReflectionUtils) GetTypeName(t reflect.Type) string {
	if t == nil {
		return ""
	}

	if t.PkgPath() == "" {
		return t.Name()
	}

	return t.PkgPath() + "." + t.Name()
}

// GetElementType 获取指针或切片的元素类型
func (ru *ReflectionUtils) GetElementType(t reflect.Type) reflect.Type {
	if t == nil {
		return nil
	}

	switch t.Kind() {
	case reflect.Ptr, reflect.Slice, reflect.Array, reflect.Chan:
		return t.Elem()
	default:
		return t
	}
}

// IsExported 检查字段或方法是否导出
func (ru *ReflectionUtils) IsExported(name string) bool {
	return len(name) > 0 && strings.ToUpper(name[:1]) == name[:1]
}

// IsBasicType 检查是否为基本类型
func (ru *ReflectionUtils) IsBasicType(t reflect.Type) bool {
	if t == nil {
		return false
	}

	switch t.Kind() {
	case reflect.Bool, reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64,
		reflect.Float32, reflect.Float64, reflect.Complex64, reflect.Complex128,
		reflect.String:
		return true
	default:
		return false
	}
}

// IsNumericType 检查是否为数值类型
func (ru *ReflectionUtils) IsNumericType(t reflect.Type) bool {
	if t == nil {
		return false
	}

	switch t.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64,
		reflect.Float32, reflect.Float64, reflect.Complex64, reflect.Complex128:
		return true
	default:
		return false
	}
}

// GetMethodReceiverType 获取方法的接收者类型
func (ru *ReflectionUtils) GetMethodReceiverType(m reflect.Method) reflect.Type {
	if m.Type.NumIn() == 0 {
		return nil
	}
	return m.Type.In(0)
}

// GetMethodParameters 获取方法的参数类型
func (ru *ReflectionUtils) GetMethodParameters(m reflect.Method) []reflect.Type {
	var params []reflect.Type
	for i := 1; i < m.Type.NumIn(); i++ { // 跳过接收者
		params = append(params, m.Type.In(i))
	}
	return params
}

// GetMethodReturns 获取方法的返回值类型
func (ru *ReflectionUtils) GetMethodReturns(m reflect.Method) []reflect.Type {
	var returns []reflect.Type
	for i := 0; i < m.Type.NumOut(); i++ {
		returns = append(returns, m.Type.Out(i))
	}
	return returns
}

// IsMethodVariadic 检查方法是否为可变参数
func (ru *ReflectionUtils) IsMethodVariadic(m reflect.Method) bool {
	return m.Type.IsVariadic()
}

// GetStructFields 获取结构体的字段信息
func (ru *ReflectionUtils) GetStructFields(t reflect.Type) []reflect.StructField {
	if t == nil || t.Kind() != reflect.Struct {
		return nil
	}

	var fields []reflect.StructField
	for i := 0; i < t.NumField(); i++ {
		fields = append(fields, t.Field(i))
	}
	return fields
}

// GetInterfaceMethods 获取接口的方法信息
func (ru *ReflectionUtils) GetInterfaceMethods(t reflect.Type) []reflect.Method {
	if t == nil || t.Kind() != reflect.Interface {
		return nil
	}

	var methods []reflect.Method
	for i := 0; i < t.NumMethod(); i++ {
		methods = append(methods, t.Method(i))
	}
	return methods
}

// GetStructMethods 获取结构体的方法信息
func (ru *ReflectionUtils) GetStructMethods(t reflect.Type) []reflect.Method {
	if t == nil {
		return nil
	}

	var methods []reflect.Method

	// 获取值接收者的方法
	if t.Kind() == reflect.Struct {
		for i := 0; i < t.NumMethod(); i++ {
			methods = append(methods, t.Method(i))
		}
	}

	// 获取指针接收者的方法
	ptrType := reflect.PtrTo(t)
	for i := 0; i < ptrType.NumMethod(); i++ {
		method := ptrType.Method(i)
		// 避免重复添加同名方法
		if !ru.containsMethod(methods, method.Name) {
			methods = append(methods, method)
		}
	}

	return methods
}

// containsMethod 检查方法列表中是否包含指定名称的方法
func (ru *ReflectionUtils) containsMethod(methods []reflect.Method, name string) bool {
	for _, m := range methods {
		if m.Name == name {
			return true
		}
	}
	return false
}

// GetTypeSize 获取类型的大小（字节）
func (ru *ReflectionUtils) GetTypeSize(t reflect.Type) int64 {
	if t == nil {
		return 0
	}
	return int64(t.Size())
}

// IsTypeComparable 检查类型是否可比较
func (ru *ReflectionUtils) IsTypeComparable(t reflect.Type) bool {
	if t == nil {
		return false
	}
	return t.Comparable()
}

// GetTypeKind 获取类型的种类
func (ru *ReflectionUtils) GetTypeKind(t reflect.Type) reflect.Kind {
	if t == nil {
		return reflect.Invalid
	}
	return t.Kind()
}
