package config

import "github.com/php-any/generator/core"

// ResolvePackageMapping 返回映射后的包路径，若无映射则返回原始路径
func ResolvePackageMapping(cfg *core.GeneratorConfig, sourcePkg string) string {
	if cfg == nil {
		return sourcePkg
	}
	if target, ok := cfg.GetPackageMapping(sourcePkg); ok {
		return target
	}
	return sourcePkg
}
