package config

// 这个文件现在主要用于配置加载和验证的工具函数
// 主要的配置类型定义已经移到 core/types.go 中

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/php-any/generator/core"
	"gopkg.in/yaml.v3"
)

// CreateDefaultConfig 创建默认配置（包装core包的函数）
func CreateDefaultConfig() *core.GeneratorConfig {
	return core.NewDefaultConfig()
}

// LoadConfigFromFile 从文件加载配置
func LoadConfigFromFile(path string) (*core.GeneratorConfig, error) {
	if path == "" {
		return nil, fmt.Errorf("配置文件路径不能为空")
	}

	// 检查文件是否存在
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return nil, fmt.Errorf("配置文件不存在: %s", path)
	}

	// 读取文件内容
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("读取配置文件失败: %v", err)
	}

	// 根据文件扩展名选择解析方式
	ext := strings.ToLower(filepath.Ext(path))
	var config *core.GeneratorConfig

	switch ext {
	case ".yaml", ".yml":
		config, err = parseYAMLConfig(content)
	default:
		return nil, fmt.Errorf("不支持的配置文件格式: %s，只支持 YAML 格式", ext)
	}

	if err != nil {
		return nil, fmt.Errorf("解析 YAML 配置文件失败: %v", err)
	}

	// 验证配置
	if err := ValidateConfigFile(config); err != nil {
		return nil, fmt.Errorf("配置验证失败: %v", err)
	}

	return config, nil
}

// ValidateConfigFile 验证配置文件
func ValidateConfigFile(config *core.GeneratorConfig) error {
	if config == nil {
		return fmt.Errorf("配置不能为空")
	}

	var errors []string

	// 验证基本配置
	if config.MaxDepth <= 0 {
		errors = append(errors, "max_depth 必须大于 0")
	}

	if config.OutputRoot == "" {
		errors = append(errors, "output_root 不能为空")
	}

	// 验证黑名单配置
	if err := validateBlacklistConfig(config.Blacklist); err != nil {
		errors = append(errors, fmt.Sprintf("黑名单配置错误: %v", err))
	}

	// 验证包前缀配置
	if err := validatePackagePrefixes(config.PackagePrefixes); err != nil {
		errors = append(errors, fmt.Sprintf("包前缀配置错误: %v", err))
	}

	// 验证包映射配置
	if err := validatePackageMappings(config.PackageMappings); err != nil {
		errors = append(errors, fmt.Sprintf("包映射配置错误: %v", err))
	}

	// 验证缓存配置
	if err := validateCacheConfig(config.Advanced.Cache); err != nil {
		errors = append(errors, fmt.Sprintf("缓存配置错误: %v", err))
	}

	if len(errors) > 0 {
		return fmt.Errorf("配置验证失败:\n%s", strings.Join(errors, "\n"))
	}

	return nil
}

// 解析YAML配置
func parseYAMLConfig(content []byte) (*core.GeneratorConfig, error) {
	var config core.GeneratorConfig
	if err := yaml.Unmarshal(content, &config); err != nil {
		return nil, fmt.Errorf("YAML解析失败: %v", err)
	}
	return &config, nil
}

// 验证黑名单配置
func validateBlacklistConfig(blacklist core.BlacklistConfig) error {
	// 验证包黑名单
	for i, pkg := range blacklist.Packages {
		if strings.TrimSpace(pkg) == "" {
			return fmt.Errorf("包黑名单[%d]不能为空", i)
		}
	}

	// 验证类型黑名单
	for i, typ := range blacklist.Types {
		if strings.TrimSpace(typ) == "" {
			return fmt.Errorf("类型黑名单[%d]不能为空", i)
		}
	}

	// 验证方法黑名单
	for i, method := range blacklist.Methods {
		if strings.TrimSpace(method) == "" {
			return fmt.Errorf("方法黑名单[%d]不能为空", i)
		}
	}

	return nil
}

// 验证包前缀配置
func validatePackagePrefixes(prefixes map[string]string) error {
	for pkg, prefix := range prefixes {
		if strings.TrimSpace(pkg) == "" {
			return fmt.Errorf("包路径不能为空")
		}
		if strings.TrimSpace(prefix) == "" {
			return fmt.Errorf("包 %s 的前缀不能为空", pkg)
		}
	}
	return nil
}

// 验证包映射配置
func validatePackageMappings(mappings map[string]string) error {
	for from, to := range mappings {
		if strings.TrimSpace(from) == "" {
			return fmt.Errorf("源包路径不能为空")
		}
		if strings.TrimSpace(to) == "" {
			return fmt.Errorf("目标包路径不能为空")
		}
		if from == to {
			return fmt.Errorf("源包和目标包不能相同: %s", from)
		}
	}
	return nil
}

// 验证缓存配置
func validateCacheConfig(cache core.CacheConfig) error {
	if cache.Enabled && cache.MaxSize <= 0 {
		return fmt.Errorf("缓存启用时最大大小必须大于 0")
	}
	if cache.Enabled && cache.TTL <= 0 {
		return fmt.Errorf("缓存启用时TTL必须大于 0")
	}
	return nil
}
