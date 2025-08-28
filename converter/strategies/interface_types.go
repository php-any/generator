package strategies

import (
	"strings"
	"unicode"

	"github.com/php-any/generator/core"
)

// InterfaceTypeStrategy 接口类型转换策略
type InterfaceTypeStrategy struct {
	*BaseStrategy
}

// NewInterfaceTypeStrategy 创建新的接口类型策略
func NewInterfaceTypeStrategy() *InterfaceTypeStrategy {
	return &InterfaceTypeStrategy{
		BaseStrategy: NewBaseStrategy("InterfaceType", 15),
	}
}

// CanConvert 检查是否可以转换接口类型
func (its *InterfaceTypeStrategy) CanConvert(t *core.TypeInfo) bool {
	if t == nil || t.Type == nil {
		return false
	}

	return t.IsInterface
}

// Convert 转换接口类型
func (its *InterfaceTypeStrategy) Convert(ctx *ConversionContext) (string, error) {
	if ctx.Type == nil || ctx.Type.Type == nil {
		return "", core.NewGeneratorError(core.ErrCodeTypeConversion, "type is nil", nil)
	}

	// 获取包路径和类型名
	pkgPath := ctx.Type.PackagePath
	typeName := ctx.Type.TypeName

	// 如果包路径为空，直接返回类型名
	if pkgPath == "" {
		return typeName, nil
	}

	// 检查是否有包映射
	if ctx.Options != nil && ctx.Options.PackageMappings != nil {
		if mappedPkg, exists := ctx.Options.PackageMappings[pkgPath]; exists {
			pkgPath = mappedPkg
		}
	}

	// 检查是否有包别名
	if ctx.Options != nil && ctx.Options.PackageAliases != nil {
		if alias, exists := ctx.Options.PackageAliases[pkgPath]; exists {
			return alias + "." + typeName, nil
		}
	}

	// 生成包别名
	alias := its.generatePackageAlias(pkgPath)

	// 如果包含包前缀，添加前缀
	if ctx.Options != nil && ctx.Options.IncludePackagePrefix {
		return alias + "." + typeName, nil
	}

	return typeName, nil
}

// generatePackageAlias 生成包别名
func (its *InterfaceTypeStrategy) generatePackageAlias(pkgPath string) string {
	// 处理标准库包
	switch pkgPath {
	case "net/http":
		return "httpsrc"
	case "context":
		return "contextsrc"
	case "time":
		return "timesrc"
	case "fmt":
		return "fmtsrc"
	case "os":
		return "ossrc"
	case "io":
		return "iosrc"
	case "encoding/json":
		return "jsonsrc"
	case "encoding/xml":
		return "xmlsrc"
	case "database/sql":
		return "sqlsrc"
	case "net/url":
		return "urlsrc"
	case "mime/multipart":
		return "multipartsrc"
	case "crypto/tls":
		return "tlssrc"
	default:
		// 对于第三方包，使用包名的缩写
		return its.getShortPackageName(pkgPath)
	}
}

// getShortPackageName 获取包的短名称
func (its *InterfaceTypeStrategy) getShortPackageName(pkgPath string) string {
	// 提取包名
	parts := []string{}
	for _, part := range strings.Split(pkgPath, "/") {
		if part != "" {
			parts = append(parts, part)
		}
	}

	if len(parts) == 0 {
		return "pkg"
	}

	// 使用最后一个部分作为包名
	lastPart := parts[len(parts)-1]

	// 处理版本后缀
	if strings.HasPrefix(lastPart, "v") && len(lastPart) > 1 {
		for i := 1; i < len(lastPart); i++ {
			if !unicode.IsDigit(rune(lastPart[i])) {
				lastPart = lastPart[:i]
				break
			}
		}
	}

	return lastPart + "src"
}
