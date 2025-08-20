package generator

import (
	"path/filepath"
	"reflect"
	"strings"
)

// lowerFirst 将标识符首字母小写
func lowerFirst(s string) string {
	if s == "" {
		return s
	}
	runes := []rune(s)
	runes[0] = []rune(strings.ToLower(string(runes[0])))[0]
	return string(runes)
}

// upperFirst 将标识符首字母大写
func upperFirst(s string) string {
	if s == "" {
		return s
	}
	runes := []rune(s)
	runes[0] = []rune(strings.ToUpper(string(runes[0])))[0]
	return string(runes)
}

// pkgBaseName 从完整包路径提取最终包名
func pkgBaseName(pkgPath string) string {
	if pkgPath == "" {
		return "main"
	}
	base := filepath.Base(pkgPath)
	if base == "." || base == string(filepath.Separator) {
		return pkgPath
	}
	return base
}

// suggestParamName 基于类型与上下文给出较合理的参数名（不得写死具体包/函数）
// fullName 示例："database/sql.Open" 或 "database/sql.DB.QueryContext"
// simpleName 示例："Open"、"QueryContext"
func suggestParamName(index int, fullName string, simpleName string, t reflect.Type) string {
	// 解引用指针
	base := t
	if base.Kind() == reflect.Pointer {
		base = base.Elem()
	}

	// 变长/切片优先视为 args
	if t.Kind() == reflect.Slice {
		return "args"
	}

	// 依据类型名称的通用命名（不依赖包名）
	typeName := base.Name()
	switch typeName {
	case "Context":
		return "ctx"
	case "Duration":
		return "d"
	}
	if strings.HasSuffix(typeName, "Options") {
		return "opts"
	}

	// 根据函数/方法名语义的通用启发
	nameLower := strings.ToLower(simpleName)
	if base.Kind() == reflect.String {
		if strings.Contains(nameLower, "query") || strings.Contains(nameLower, "prepare") || strings.Contains(nameLower, "exec") {
			return "query"
		}
	}
	if base.Kind() == reflect.Int || base.Kind() == reflect.Int64 {
		if strings.Contains(nameLower, "setmax") || strings.Contains(nameLower, "set") {
			return "n"
		}
	}

	// 默认
	return "param" + strconvItoa(index)
}

// 轻量 itoa，避免引入 strconv
func strconvItoa(n int) string {
	if n == 0 {
		return "0"
	}
	digits := []byte{}
	sign := false
	if n < 0 {
		sign = true
		n = -n
	}
	for n > 0 {
		d := n % 10
		digits = append(digits, byte('0'+d))
		n /= 10
	}
	// reverse
	for i, j := 0, len(digits)-1; i < j; i, j = i+1, j-1 {
		digits[i], digits[j] = digits[j], digits[i]
	}
	if sign {
		return "-" + string(digits)
	}
	return string(digits)
}
