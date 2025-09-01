package scr

import "reflect"

// 返回生成类型
func parseTypes(t reflect.Type) string {
	switch t.Kind() {
	case reflect.Struct:
		return "class"
	case reflect.Interface:
		return "class"
	case reflect.Func:
		return "func"
	case reflect.Ptr:
		// 检查指针指向的元素类型
		if t.Elem() != nil {
			return parseTypes(t.Elem())
		}
		return ""
	default:
		return ""
	}
}

// isTypeNeedsProxy 检查类型是否需要生成代理类
func isTypeNeedsProxy(t reflect.Type) bool {
	switch t.Kind() {
	case reflect.Struct:
		return true
	case reflect.Interface:
		return true
	case reflect.Func:
		return true
	case reflect.Ptr:
		if t.Kind() == reflect.Ptr && t.Elem() != nil && t.Elem().Kind() == reflect.Struct {
			return true
		}
		return false
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64, reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr, reflect.Float32, reflect.Float64, reflect.Complex64, reflect.Complex128, reflect.Bool, reflect.String:
		return false
	default:
		return false
	}
}
