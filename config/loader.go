package config

import (
	"fmt"
	"os"

	"github.com/php-any/generator/core"
	"gopkg.in/yaml.v3"
)

// LoadYAML 加载 YAML 配置
func LoadYAML(path string) (*core.GeneratorConfig, error) {
	if path == "" {
		return nil, fmt.Errorf("config path is empty")
	}
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var cfg core.GeneratorConfig
	if err := yaml.Unmarshal(b, &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}
