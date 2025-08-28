package config

import (
	"fmt"
	"strings"

	"github.com/php-any/generator/core"
)

// ConfigValidator 配置验证器
type ConfigValidator struct {
	errorHandler core.ErrorHandler
}

func NewConfigValidator(errorHandler core.ErrorHandler) *ConfigValidator {
	return &ConfigValidator{errorHandler: errorHandler}
}

func (cv *ConfigValidator) ValidateConfig(cfg *core.GeneratorConfig) error {
	var errors []core.ValidationError

	if cfg.MaxDepth <= 0 {
		errors = append(errors, core.ValidationError{Field: "max_depth", Message: "max_depth must be greater than 0", Value: cfg.MaxDepth})
	}
	if cfg.OutputRoot == "" {
		errors = append(errors, core.ValidationError{Field: "output_root", Message: "output_root cannot be empty", Value: cfg.OutputRoot})
	}

	errors = append(errors, cv.validateBlacklist(cfg.Blacklist)...)
	errors = append(errors, cv.validatePackageMappings(cfg.PackageMappings)...)
	errors = append(errors, cv.validateCacheConfig(cfg.Advanced.Cache)...)

	for _, e := range errors {
		cv.errorHandler.HandleError(fmt.Errorf("validation error: %s", e.Error()))
	}
	if len(errors) > 0 {
		return core.NewGeneratorError(core.ErrCodeValidationFailed, "configuration validation failed", nil)
	}
	return nil
}

func (cv *ConfigValidator) validateBlacklist(blacklist core.BlacklistConfig) []core.ValidationError {
	var errors []core.ValidationError
	for i, pkg := range blacklist.Packages {
		if strings.TrimSpace(pkg) == "" {
			errors = append(errors, core.ValidationError{Field: fmt.Sprintf("blacklist.packages[%d]", i), Message: "package path cannot be empty", Value: pkg})
		}
	}
	for i, typ := range blacklist.Types {
		if strings.TrimSpace(typ) == "" {
			errors = append(errors, core.ValidationError{Field: fmt.Sprintf("blacklist.types[%d]", i), Message: "type name cannot be empty", Value: typ})
		}
	}
	for i, method := range blacklist.Methods {
		if strings.TrimSpace(method) == "" {
			errors = append(errors, core.ValidationError{Field: fmt.Sprintf("blacklist.methods[%d]", i), Message: "method name cannot be empty", Value: method})
		}
	}
	if blacklist.UseRegex {
		for i, pattern := range blacklist.Patterns {
			if strings.TrimSpace(pattern) == "" {
				errors = append(errors, core.ValidationError{Field: fmt.Sprintf("blacklist.patterns[%d]", i), Message: "regex pattern cannot be empty", Value: pattern})
			}
		}
	}
	return errors
}

func (cv *ConfigValidator) validatePackageMappings(mappings map[string]string) []core.ValidationError {
	var errors []core.ValidationError
	for source, target := range mappings {
		if strings.TrimSpace(source) == "" {
			errors = append(errors, core.ValidationError{Field: "package_mappings.source", Message: "source package path cannot be empty", Value: source})
		}
		if strings.TrimSpace(target) == "" {
			errors = append(errors, core.ValidationError{Field: "package_mappings.target", Message: "target package path cannot be empty", Value: target})
		}
		if source == target {
			cv.errorHandler.HandleWarning(fmt.Sprintf("circular package mapping detected: %s -> %s", source, target))
		}
	}
	return errors
}

func (cv *ConfigValidator) validateCacheConfig(cache core.CacheConfig) []core.ValidationError {
	var errors []core.ValidationError
	if cache.Enabled {
		if cache.TTL <= 0 {
			errors = append(errors, core.ValidationError{Field: "cache.ttl", Message: "cache TTL must be greater than 0", Value: cache.TTL})
		}
		if cache.MaxSize <= 0 {
			errors = append(errors, core.ValidationError{Field: "cache.max_size", Message: "cache max size must be greater than 0", Value: cache.MaxSize})
		}
		if cache.Directory == "" {
			errors = append(errors, core.ValidationError{Field: "cache.directory", Message: "cache directory cannot be empty when cache is enabled", Value: cache.Directory})
		}
	}
	return errors
}
