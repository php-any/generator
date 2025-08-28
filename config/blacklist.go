package config

import "github.com/php-any/generator/core"

// IsBlacklisted 包/类型/方法黑名单总入口
func IsBlacklisted(cfg *core.GeneratorConfig, pkgPath, typeName, methodName string) bool {
	if cfg == nil {
		return false
	}
	if pkgPath != "" && cfg.IsPackageBlacklisted(pkgPath) {
		return true
	}
	if typeName != "" && cfg.IsTypeBlacklisted(typeName) {
		return true
	}
	if methodName != "" && cfg.IsMethodBlacklisted(methodName) {
		return true
	}
	return false
}
