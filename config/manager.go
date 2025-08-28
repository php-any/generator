package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/php-any/generator/core"
	"gopkg.in/yaml.v3"
)

// ConfigManagerImpl 配置管理器实现
type ConfigManagerImpl struct {
	config       *core.GeneratorConfig
	errorHandler core.ErrorHandler
}

// 创建新的配置管理器
func NewConfigManager(errorHandler core.ErrorHandler) *ConfigManagerImpl {
	return &ConfigManagerImpl{
		config:       core.NewDefaultConfig(),
		errorHandler: errorHandler,
	}
}

// LoadConfig 加载配置文件
func (cm *ConfigManagerImpl) LoadConfig(path string) error {
	if path == "" {
		// 使用默认配置
		return nil
	}

	ext := strings.ToLower(filepath.Ext(path))
	var config *core.GeneratorConfig
	var err error

	switch ext {
	case ".yaml", ".yml":
		config, err = cm.loadYAMLConfig(path)
	default:
		return fmt.Errorf("unsupported config file format: %s, only YAML format is supported", ext)
	}

	if err != nil {
		return core.NewGeneratorError(core.ErrCodeConfigInvalid, "failed to load config", err)
	}

	// 合并配置
	cm.config.Merge(config)
	return nil
}

// ValidateConfig 验证配置
func (cm *ConfigManagerImpl) ValidateConfig() error {
	validator := NewConfigValidator(cm.errorHandler)
	return validator.ValidateConfig(cm.config)
}

// IsPackageBlacklisted 检查包是否在黑名单中
func (cm *ConfigManagerImpl) IsPackageBlacklisted(pkgPath string) bool {
	return cm.config.IsPackageBlacklisted(pkgPath)
}

// GetPackagePrefix 获取包前缀
func (cm *ConfigManagerImpl) GetPackagePrefix(pkgPath string) string {
	return cm.config.GetPackagePrefix(pkgPath)
}

// GetPackageMapping 获取包映射
func (cm *ConfigManagerImpl) GetPackageMapping(sourcePkg string) (string, bool) {
	return cm.config.GetPackageMapping(sourcePkg)
}

// GetGlobalPrefix 获取全局前缀
func (cm *ConfigManagerImpl) GetGlobalPrefix() string {
	return cm.config.GlobalPrefix
}

// GetConfig 获取配置
func (cm *ConfigManagerImpl) GetConfig() *core.GeneratorConfig {
	return cm.config
}

// 加载YAML配置
func (cm *ConfigManagerImpl) loadYAMLConfig(path string) (*core.GeneratorConfig, error) {
	if path == "" {
		return nil, fmt.Errorf("YAML配置文件路径不能为空")
	}

	// 检查文件是否存在
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return nil, fmt.Errorf("YAML配置文件不存在: %s", path)
	}

	// 读取文件内容
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("读取YAML配置文件失败: %v", err)
	}

	// 解析YAML配置
	var config core.GeneratorConfig
	if err := yaml.Unmarshal(content, &config); err != nil {
		return nil, fmt.Errorf("YAML解析失败: %v", err)
	}

	// 设置默认值
	config = *cm.setDefaultValues(&config)

	return &config, nil
}

// 设置默认值
func (cm *ConfigManagerImpl) setDefaultValues(config *core.GeneratorConfig) *core.GeneratorConfig {
	// 如果配置为空，创建默认配置
	if config == nil {
		config = core.NewDefaultConfig()
		return config
	}

	// 设置基本配置默认值
	if config.MaxDepth <= 0 {
		config.MaxDepth = 10
	}
	if config.OutputRoot == "" {
		config.OutputRoot = "origami"
	}

	// 设置高级配置默认值
	if config.Advanced.Cache.Enabled {
		if config.Advanced.Cache.MaxSize <= 0 {
			config.Advanced.Cache.MaxSize = 1000
		}
		if config.Advanced.Cache.TTL <= 0 {
			config.Advanced.Cache.TTL = 300 // 5 minutes in seconds
		}
	}

	// 设置黑名单默认值
	if config.Blacklist.Packages == nil {
		config.Blacklist.Packages = []string{}
	}
	if config.Blacklist.Types == nil {
		config.Blacklist.Types = []string{}
	}
	if config.Blacklist.Methods == nil {
		config.Blacklist.Methods = []string{}
	}

	// 设置包前缀默认值
	if config.PackagePrefixes == nil {
		config.PackagePrefixes = make(map[string]string)
	}

	// 设置包映射默认值
	if config.PackageMappings == nil {
		config.PackageMappings = make(map[string]string)
	}

	return config
}
