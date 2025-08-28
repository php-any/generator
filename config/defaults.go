package config

import "github.com/php-any/generator/core"

// NewDefaultConfig 提供导出默认配置（转发 core 层）
func NewDefaultConfig() *core.GeneratorConfig {
	return core.NewDefaultConfig()
}
