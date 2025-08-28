package utils

import (
	"strings"
	"unicode"
)

// NamingUtils 命名工具函数集合
type NamingUtils struct{}

// NewNamingUtils 创建新的命名工具实例
func NewNamingUtils() *NamingUtils {
	return &NamingUtils{}
}

// UpperFirst 首字母大写
func (nu *NamingUtils) UpperFirst(s string) string {
	if len(s) == 0 {
		return s
	}
	return string(unicode.ToUpper(rune(s[0]))) + s[1:]
}

// LowerFirst 首字母小写
func (nu *NamingUtils) LowerFirst(s string) string {
	if len(s) == 0 {
		return s
	}
	return string(unicode.ToLower(rune(s[0]))) + s[1:]
}

// ToCamelCase 转换为驼峰命名
func (nu *NamingUtils) ToCamelCase(s string) string {
	if len(s) == 0 {
		return s
	}

	// 处理下划线分隔的字符串
	if strings.Contains(s, "_") {
		parts := strings.Split(s, "_")
		result := ""
		for _, part := range parts {
			if len(part) > 0 {
				result += nu.UpperFirst(part)
			}
		}
		return result
	}

	// 处理连字符分隔的字符串
	if strings.Contains(s, "-") {
		parts := strings.Split(s, "-")
		result := ""
		for _, part := range parts {
			if len(part) > 0 {
				result += nu.UpperFirst(part)
			}
		}
		return result
	}

	// 处理点分隔的字符串
	if strings.Contains(s, ".") {
		parts := strings.Split(s, ".")
		result := ""
		for _, part := range parts {
			if len(part) > 0 {
				result += nu.UpperFirst(part)
			}
		}
		return result
	}

	// 处理空格分隔的字符串
	if strings.Contains(s, " ") {
		parts := strings.Split(s, " ")
		result := ""
		for _, part := range parts {
			if len(part) > 0 {
				result += nu.UpperFirst(part)
			}
		}
		return result
	}

	// 如果已经是驼峰命名，直接返回
	return s
}

// ToSnakeCase 转换为下划线命名
func (nu *NamingUtils) ToSnakeCase(s string) string {
	if len(s) == 0 {
		return s
	}

	var result strings.Builder
	for i, r := range s {
		if unicode.IsUpper(r) {
			if i > 0 {
				result.WriteRune('_')
			}
			result.WriteRune(unicode.ToLower(r))
		} else {
			result.WriteRune(r)
		}
	}

	return result.String()
}

// ToKebabCase 转换为连字符命名
func (nu *NamingUtils) ToKebabCase(s string) string {
	if len(s) == 0 {
		return s
	}

	var result strings.Builder
	for i, r := range s {
		if unicode.IsUpper(r) {
			if i > 0 {
				result.WriteRune('-')
			}
			result.WriteRune(unicode.ToLower(r))
		} else {
			result.WriteRune(r)
		}
	}

	return result.String()
}

// ToPascalCase 转换为帕斯卡命名（首字母大写的驼峰命名）
func (nu *NamingUtils) ToPascalCase(s string) string {
	return nu.UpperFirst(nu.ToCamelCase(s))
}

// ToLowerCamelCase 转换为小驼峰命名（首字母小写的驼峰命名）
func (nu *NamingUtils) ToLowerCamelCase(s string) string {
	return nu.LowerFirst(nu.ToCamelCase(s))
}

// SanitizeIdentifier 清理标识符，确保符合Go命名规范
func (nu *NamingUtils) SanitizeIdentifier(s string) string {
	if len(s) == 0 {
		return "_"
	}

	// 如果以数字开头，添加下划线前缀
	if unicode.IsDigit(rune(s[0])) {
		s = "_" + s
	}

	// 替换非法字符为下划线
	var result strings.Builder
	for _, r := range s {
		if unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_' {
			result.WriteRune(r)
		} else {
			result.WriteRune('_')
		}
	}

	return result.String()
}

// GetPackageName 从包路径中提取包名
func (nu *NamingUtils) GetPackageName(pkgPath string) string {
	if len(pkgPath) == 0 {
		return ""
	}

	// 处理标准库包
	if !strings.Contains(pkgPath, "/") {
		return pkgPath
	}

	// 处理第三方包
	parts := strings.Split(pkgPath, "/")
	if len(parts) == 0 {
		return ""
	}

	lastPart := parts[len(parts)-1]

	// 处理版本后缀（如 v2, v3）
	if strings.HasPrefix(lastPart, "v") && len(lastPart) > 1 {
		for i := 1; i < len(lastPart); i++ {
			if !unicode.IsDigit(rune(lastPart[i])) {
				lastPart = lastPart[:i]
				break
			}
		}
	}

	return lastPart
}

// GetShortPackageName 获取包的短名称（用于导入别名）
func (nu *NamingUtils) GetShortPackageName(pkgPath string) string {
	pkgName := nu.GetPackageName(pkgPath)

	// 如果包名太长，使用缩写
	if len(pkgName) > 10 {
		// 尝试使用包名的前几个字符
		words := nu.splitCamelCase(pkgName)
		if len(words) > 0 {
			var result strings.Builder
			for _, word := range words {
				if len(word) > 0 {
					result.WriteRune(rune(word[0]))
				}
			}
			return strings.ToLower(result.String())
		}
	}

	return strings.ToLower(pkgName)
}

// splitCamelCase 分割驼峰命名的字符串
func (nu *NamingUtils) splitCamelCase(s string) []string {
	var words []string
	var current strings.Builder

	for i, r := range s {
		if unicode.IsUpper(r) && i > 0 {
			if current.Len() > 0 {
				words = append(words, current.String())
				current.Reset()
			}
		}
		current.WriteRune(r)
	}

	if current.Len() > 0 {
		words = append(words, current.String())
	}

	return words
}

// GenerateMethodName 生成方法名称
func (nu *NamingUtils) GenerateMethodName(className, methodName string) string {
	if len(methodName) == 0 {
		return className
	}

	// 如果方法名已经是完整的方法名，直接返回
	if strings.HasPrefix(methodName, className) {
		return methodName
	}

	// 否则组合类名和方法名
	return className + nu.UpperFirst(methodName)
}

// GenerateClassName 生成类名称
func (nu *NamingUtils) GenerateClassName(typeName string) string {
	if len(typeName) == 0 {
		return "Class"
	}

	return nu.UpperFirst(typeName) + "Class"
}

// GenerateMethodClassName 生成方法类名称
func (nu *NamingUtils) GenerateMethodClassName(methodName string) string {
	if len(methodName) == 0 {
		return "Method"
	}

	return nu.UpperFirst(methodName) + "Method"
}

// GenerateFunctionName 生成函数名称
func (nu *NamingUtils) GenerateFunctionName(funcName string) string {
	if len(funcName) == 0 {
		return "Function"
	}

	return nu.UpperFirst(funcName) + "Function"
}

// IsReservedWord 检查是否为Go保留字
func (nu *NamingUtils) IsReservedWord(word string) bool {
	reservedWords := map[string]bool{
		"break": true, "case": true, "chan": true, "const": true, "continue": true,
		"default": true, "defer": true, "else": true, "fallthrough": true, "for": true,
		"func": true, "go": true, "goto": true, "if": true, "import": true,
		"interface": true, "map": true, "package": true, "range": true, "return": true,
		"select": true, "struct": true, "switch": true, "type": true, "var": true,
	}

	return reservedWords[strings.ToLower(word)]
}

// EscapeReservedWord 转义保留字
func (nu *NamingUtils) EscapeReservedWord(word string) string {
	if nu.IsReservedWord(word) {
		return word + "_"
	}
	return word
}
