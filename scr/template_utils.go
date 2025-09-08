package scr

import (
	"fmt"
	"reflect"
	"strings"
)

// getTypeString 获取类型的字符串表示
func getTypeString(t reflect.Type, fileCache *FileCache) string {
	if t == nil {
		return "interface{}"
	}

	switch t.Kind() {
	case reflect.Ptr:
		return "*" + getTypeString(t.Elem(), fileCache)
	case reflect.Slice:
		return "[]" + getTypeString(t.Elem(), fileCache)
	case reflect.Array:
		return fmt.Sprintf("[%d]%s", t.Len(), getTypeString(t.Elem(), fileCache))
	case reflect.Map:
		return fmt.Sprintf("map[%s]%s", getTypeString(t.Key(), fileCache), getTypeString(t.Elem(), fileCache))
	case reflect.Chan:
		return "chan " + getTypeString(t.Elem(), fileCache)
	case reflect.Func:
		return "interface{}"
	case reflect.Struct, reflect.Interface:
		if t.PkgPath() != "" {
			// 对于标准库类型，直接使用包名
			if isStandardLibrary(t.PkgPath()) {
				return t.PkgPath() + "." + t.Name()
			}
			// 对于第三方包，使用包名
			pkgName := pkgBaseName(t.PkgPath())
			return pkgName + "." + t.Name()
		}
		return t.Name()
	default:
		return t.Name()
	}
}

// isStandardLibrary 检查包路径是否属于标准库
func isStandardLibrary(pkgPath string) bool {
	// 标准库包路径不包含域名，直接以包名开头
	// 例如: "context", "fmt", "time", "net/http" 等
	return !strings.Contains(pkgPath, ".") && !strings.Contains(pkgPath, "/")
}

// lowerFirst 将名称首字母小写
func lowerFirst(s string) string {
	if s == "" {
		return s
	}
	r := []rune(s)
	r[0] = []rune(strings.ToLower(string(r[0])))[0]
	return string(r)
}
