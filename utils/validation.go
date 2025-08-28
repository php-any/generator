package utils

import (
	"encoding/json"
	"fmt"
	"reflect"
	"regexp"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"
)

// ValidationUtils 验证工具函数集合
type ValidationUtils struct{}

// NewValidationUtils 创建新的验证工具实例
func NewValidationUtils() *ValidationUtils {
	return &ValidationUtils{}
}

// ValidationError 验证错误
type ValidationError struct {
	Field   string
	Message string
	Value   interface{}
}

// Error 实现 error 接口
func (ve *ValidationError) Error() string {
	return fmt.Sprintf("validation error in field '%s': %s (value: %v)", ve.Field, ve.Message, ve.Value)
}

// ValidationResult 验证结果
type ValidationResult struct {
	IsValid bool
	Errors  []*ValidationError
}

// NewValidationResult 创建新的验证结果
func NewValidationResult() *ValidationResult {
	return &ValidationResult{
		IsValid: true,
		Errors:  make([]*ValidationError, 0),
	}
}

// AddError 添加验证错误
func (vr *ValidationResult) AddError(field, message string, value interface{}) {
	vr.IsValid = false
	vr.Errors = append(vr.Errors, &ValidationError{
		Field:   field,
		Message: message,
		Value:   value,
	})
}

// GetErrors 获取所有错误
func (vr *ValidationResult) GetErrors() []*ValidationError {
	return vr.Errors
}

// IsEmpty 检查字符串是否为空
func (vu *ValidationUtils) IsEmpty(value string) bool {
	return strings.TrimSpace(value) == ""
}

// IsNotEmpty 检查字符串是否非空
func (vu *ValidationUtils) IsNotEmpty(value string) bool {
	return !vu.IsEmpty(value)
}

// IsValidEmail 检查是否为有效邮箱
func (vu *ValidationUtils) IsValidEmail(email string) bool {
	emailRegex := regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)
	return emailRegex.MatchString(email)
}

// IsValidURL 检查是否为有效URL
func (vu *ValidationUtils) IsValidURL(url string) bool {
	urlRegex := regexp.MustCompile(`^(https?|ftp)://[^\s/$.?#].[^\s]*$`)
	return urlRegex.MatchString(url)
}

// IsValidPhone 检查是否为有效电话号码
func (vu *ValidationUtils) IsValidPhone(phone string) bool {
	phoneRegex := regexp.MustCompile(`^[\+]?[1-9][\d]{0,15}$`)
	return phoneRegex.MatchString(phone)
}

// IsValidIP 检查是否为有效IP地址
func (vu *ValidationUtils) IsValidIP(ip string) bool {
	ipRegex := regexp.MustCompile(`^((25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)\.){3}(25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)$`)
	return ipRegex.MatchString(ip)
}

// IsValidIPv6 检查是否为有效IPv6地址
func (vu *ValidationUtils) IsValidIPv6(ip string) bool {
	ipv6Regex := regexp.MustCompile(`^([0-9a-fA-F]{1,4}:){7}[0-9a-fA-F]{1,4}$`)
	return ipv6Regex.MatchString(ip)
}

// IsValidDomain 检查是否为有效域名
func (vu *ValidationUtils) IsValidDomain(domain string) bool {
	domainRegex := regexp.MustCompile(`^[a-zA-Z0-9]([a-zA-Z0-9\-]{0,61}[a-zA-Z0-9])?(\.[a-zA-Z0-9]([a-zA-Z0-9\-]{0,61}[a-zA-Z0-9])?)*$`)
	return domainRegex.MatchString(domain)
}

// IsValidPath 检查是否为有效路径
func (vu *ValidationUtils) IsValidPath(path string) bool {
	// 检查路径是否包含非法字符
	illegalChars := regexp.MustCompile(`[<>:"|?*]`)
	if illegalChars.MatchString(path) {
		return false
	}

	// 检查路径长度
	if len(path) > 260 { // Windows 路径长度限制
		return false
	}

	return true
}

// IsValidFileName 检查是否为有效文件名
func (vu *ValidationUtils) IsValidFileName(filename string) bool {
	// 检查文件名是否包含非法字符
	illegalChars := regexp.MustCompile(`[<>:"/\\|?*]`)
	if illegalChars.MatchString(filename) {
		return false
	}

	// 检查文件名长度
	if len(filename) > 255 {
		return false
	}

	// 检查是否为保留名称
	reservedNames := map[string]bool{
		"CON": true, "PRN": true, "AUX": true, "NUL": true,
		"COM1": true, "COM2": true, "COM3": true, "COM4": true, "COM5": true,
		"COM6": true, "COM7": true, "COM8": true, "COM9": true,
		"LPT1": true, "LPT2": true, "LPT3": true, "LPT4": true, "LPT5": true,
		"LPT6": true, "LPT7": true, "LPT8": true, "LPT9": true,
	}

	if reservedNames[strings.ToUpper(filename)] {
		return false
	}

	return true
}

// IsValidPackageName 检查是否为有效包名
func (vu *ValidationUtils) IsValidPackageName(pkgName string) bool {
	if vu.IsEmpty(pkgName) {
		return false
	}

	// 包名必须以字母开头
	if !regexp.MustCompile(`^[a-zA-Z]`).MatchString(pkgName) {
		return false
	}

	// 包名只能包含字母、数字和下划线
	if !regexp.MustCompile(`^[a-zA-Z][a-zA-Z0-9_]*$`).MatchString(pkgName) {
		return false
	}

	// 包名不能是Go关键字
	keywords := map[string]bool{
		"break": true, "case": true, "chan": true, "const": true, "continue": true,
		"default": true, "defer": true, "else": true, "fallthrough": true, "for": true,
		"func": true, "go": true, "goto": true, "if": true, "import": true,
		"interface": true, "map": true, "package": true, "range": true, "return": true,
		"select": true, "struct": true, "switch": true, "type": true, "var": true,
	}

	if keywords[pkgName] {
		return false
	}

	return true
}

// IsValidTypeName 检查是否为有效类型名
func (vu *ValidationUtils) IsValidTypeName(typeName string) bool {
	return vu.IsValidPackageName(typeName)
}

// IsValidFunctionName 检查是否为有效函数名
func (vu *ValidationUtils) IsValidFunctionName(funcName string) bool {
	return vu.IsValidPackageName(funcName)
}

// IsValidMethodName 检查是否为有效方法名
func (vu *ValidationUtils) IsValidMethodName(methodName string) bool {
	return vu.IsValidPackageName(methodName)
}

// IsValidFieldName 检查是否为有效字段名
func (vu *ValidationUtils) IsValidFieldName(fieldName string) bool {
	return vu.IsValidPackageName(fieldName)
}

// IsValidVariableName 检查是否为有效变量名
func (vu *ValidationUtils) IsValidVariableName(varName string) bool {
	return vu.IsValidPackageName(varName)
}

// IsValidConstantName 检查是否为有效常量名
func (vu *ValidationUtils) IsValidConstantName(constName string) bool {
	return vu.IsValidPackageName(constName)
}

// IsValidStructTag 检查是否为有效结构体标签
func (vu *ValidationUtils) IsValidStructTag(tag string) bool {
	if vu.IsEmpty(tag) {
		return true // 空标签是有效的
	}

	// 结构体标签必须用反引号包围
	if !strings.HasPrefix(tag, "`") || !strings.HasSuffix(tag, "`") {
		return false
	}

	// 移除反引号
	tag = strings.Trim(tag, "`")

	// 检查标签内容
	if !regexp.MustCompile(`^[a-zA-Z0-9_\-:"\s,=]+$`).MatchString(tag) {
		return false
	}

	return true
}

// IsValidJSON 检查是否为有效JSON
func (vu *ValidationUtils) IsValidJSON(jsonStr string) bool {
	var js interface{}
	return json.Unmarshal([]byte(jsonStr), &js) == nil
}

// IsValidYAML 检查是否为有效YAML
func (vu *ValidationUtils) IsValidYAML(yamlStr string) bool {
	var y interface{}
	return yaml.Unmarshal([]byte(yamlStr), &y) == nil
}

// IsValidRegex 检查是否为有效正则表达式
func (vu *ValidationUtils) IsValidRegex(regexStr string) bool {
	_, err := regexp.Compile(regexStr)
	return err == nil
}

// IsValidUUID 检查是否为有效UUID
func (vu *ValidationUtils) IsValidUUID(uuid string) bool {
	uuidRegex := regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)
	return uuidRegex.MatchString(uuid)
}

// IsValidCreditCard 检查是否为有效信用卡号
func (vu *ValidationUtils) IsValidCreditCard(cardNumber string) bool {
	// 移除空格和连字符
	cardNumber = regexp.MustCompile(`[\s\-]`).ReplaceAllString(cardNumber, "")

	// 检查长度
	if len(cardNumber) < 13 || len(cardNumber) > 19 {
		return false
	}

	// 检查是否只包含数字
	if !regexp.MustCompile(`^\d+$`).MatchString(cardNumber) {
		return false
	}

	// Luhn 算法验证
	sum := 0
	alternate := false

	for i := len(cardNumber) - 1; i >= 0; i-- {
		digit := int(cardNumber[i] - '0')

		if alternate {
			digit *= 2
			if digit > 9 {
				digit = (digit % 10) + 1
			}
		}

		sum += digit
		alternate = !alternate
	}

	return sum%10 == 0
}

// IsValidISBN 检查是否为有效ISBN
func (vu *ValidationUtils) IsValidISBN(isbn string) bool {
	// 移除连字符和空格
	isbn = regexp.MustCompile(`[\s\-]`).ReplaceAllString(isbn, "")

	// 检查长度
	if len(isbn) != 10 && len(isbn) != 13 {
		return false
	}

	// 检查是否只包含数字（ISBN-10 最后一位可能是 X）
	if len(isbn) == 10 {
		if !regexp.MustCompile(`^\d{9}[\dX]$`).MatchString(isbn) {
			return false
		}
	} else {
		if !regexp.MustCompile(`^\d{13}$`).MatchString(isbn) {
			return false
		}
	}

	// 验证校验和
	if len(isbn) == 10 {
		return vu.validateISBN10(isbn)
	} else {
		return vu.validateISBN13(isbn)
	}
}

// validateISBN10 验证ISBN-10
func (vu *ValidationUtils) validateISBN10(isbn string) bool {
	sum := 0
	for i := 0; i < 9; i++ {
		sum += int(isbn[i]-'0') * (10 - i)
	}

	lastChar := isbn[9]
	if lastChar == 'X' {
		sum += 10
	} else {
		sum += int(lastChar - '0')
	}

	return sum%11 == 0
}

// validateISBN13 验证ISBN-13
func (vu *ValidationUtils) validateISBN13(isbn string) bool {
	sum := 0
	for i := 0; i < 12; i++ {
		digit := int(isbn[i] - '0')
		if i%2 == 0 {
			sum += digit
		} else {
			sum += digit * 3
		}
	}

	checkDigit := int(isbn[12] - '0')
	expectedCheckDigit := (10 - (sum % 10)) % 10

	return checkDigit == expectedCheckDigit
}

// ValidateStruct 验证结构体
func (vu *ValidationUtils) ValidateStruct(obj interface{}) *ValidationResult {
	result := NewValidationResult()

	val := reflect.ValueOf(obj)
	if val.Kind() == reflect.Ptr {
		val = val.Elem()
	}

	if val.Kind() != reflect.Struct {
		result.AddError("", "object is not a struct", obj)
		return result
	}

	typ := val.Type()
	for i := 0; i < val.NumField(); i++ {
		field := val.Field(i)
		fieldType := typ.Field(i)

		// 获取字段标签
		tag := fieldType.Tag.Get("validate")
		if tag == "" {
			continue
		}

		// 解析验证规则
		rules := strings.Split(tag, ",")
		for _, rule := range rules {
			rule = strings.TrimSpace(rule)

			switch {
			case rule == "required":
				if field.IsZero() {
					result.AddError(fieldType.Name, "field is required", field.Interface())
				}
			case strings.HasPrefix(rule, "min="):
				minStr := strings.TrimPrefix(rule, "min=")
				if min, err := strconv.Atoi(minStr); err == nil {
					if field.Len() < min {
						result.AddError(fieldType.Name, fmt.Sprintf("field length must be at least %d", min), field.Interface())
					}
				}
			case strings.HasPrefix(rule, "max="):
				maxStr := strings.TrimPrefix(rule, "max=")
				if max, err := strconv.Atoi(maxStr); err == nil {
					if field.Len() > max {
						result.AddError(fieldType.Name, fmt.Sprintf("field length must be at most %d", max), field.Interface())
					}
				}
			case rule == "email":
				if field.Kind() == reflect.String && !vu.IsValidEmail(field.String()) {
					result.AddError(fieldType.Name, "field must be a valid email", field.Interface())
				}
			case rule == "url":
				if field.Kind() == reflect.String && !vu.IsValidURL(field.String()) {
					result.AddError(fieldType.Name, "field must be a valid URL", field.Interface())
				}
			}
		}
	}

	return result
}
