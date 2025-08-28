package core

import (
	"fmt"
	"strings"
	"sync"
)

// GeneratorError 生成器错误
type GeneratorError struct {
	Code    string
	Message string
	Details map[string]interface{}
	Cause   error
}

// 错误代码定义
const (
	ErrCodeConfigInvalid      = "CONFIG_INVALID"
	ErrCodeTypeAnalysis       = "TYPE_ANALYSIS"
	ErrCodeTypeConversion     = "TYPE_CONVERSION"
	ErrCodeCodeGeneration     = "CODE_GENERATION"
	ErrCodeFileOperation      = "FILE_OPERATION"
	ErrCodePackageBlacklisted = "PACKAGE_BLACKLISTED"
	ErrCodeTypeBlacklisted    = "TYPE_BLACKLISTED"
	ErrCodeValidationFailed   = "VALIDATION_FAILED"
)

// 实现error接口
func (e *GeneratorError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("[%s] %s: %v", e.Code, e.Message, e.Cause)
	}
	return fmt.Sprintf("[%s] %s", e.Code, e.Message)
}

// Unwrap 返回原始错误
func (e *GeneratorError) Unwrap() error {
	return e.Cause
}

// 创建新的生成器错误
func NewGeneratorError(code, message string, cause error) *GeneratorError {
	return &GeneratorError{
		Code:    code,
		Message: message,
		Details: make(map[string]interface{}),
		Cause:   cause,
	}
}

// 添加错误详情
func (e *GeneratorError) AddDetail(key string, value interface{}) *GeneratorError {
	e.Details[key] = value
	return e
}

// ValidationError 验证错误
type ValidationError struct {
	Field   string
	Message string
	Value   interface{}
}

// 实现error接口
func (e *ValidationError) Error() string {
	return fmt.Sprintf("validation failed for field '%s': %s", e.Field, e.Message)
}

// DefaultErrorHandler 默认错误处理器
type DefaultErrorHandler struct {
	errors   []error
	warnings []string
	mutex    sync.RWMutex
}

// 创建默认错误处理器
func NewDefaultErrorHandler() *DefaultErrorHandler {
	return &DefaultErrorHandler{
		errors:   make([]error, 0),
		warnings: make([]string, 0),
	}
}

// HandleError 处理错误
func (h *DefaultErrorHandler) HandleError(err error) {
	h.mutex.Lock()
	defer h.mutex.Unlock()
	h.errors = append(h.errors, err)
}

// HandleWarning 处理警告
func (h *DefaultErrorHandler) HandleWarning(msg string) {
	h.mutex.Lock()
	defer h.mutex.Unlock()
	h.warnings = append(h.warnings, msg)
}

// HasErrors 检查是否有错误
func (h *DefaultErrorHandler) HasErrors() bool {
	h.mutex.RLock()
	defer h.mutex.RUnlock()
	return len(h.errors) > 0
}

// GetErrors 获取所有错误
func (h *DefaultErrorHandler) GetErrors() []error {
	h.mutex.RLock()
	defer h.mutex.RUnlock()
	result := make([]error, len(h.errors))
	copy(result, h.errors)
	return result
}

// GetWarnings 获取所有警告
func (h *DefaultErrorHandler) GetWarnings() []string {
	h.mutex.RLock()
	defer h.mutex.RUnlock()
	result := make([]string, len(h.warnings))
	copy(result, h.warnings)
	return result
}

// Clear 清除所有错误和警告
func (h *DefaultErrorHandler) Clear() {
	h.mutex.Lock()
	defer h.mutex.Unlock()
	h.errors = make([]error, 0)
	h.warnings = make([]string, 0)
}

// GetErrorSummary 获取错误摘要
func (h *DefaultErrorHandler) GetErrorSummary() string {
	h.mutex.RLock()
	defer h.mutex.RUnlock()

	if len(h.errors) == 0 && len(h.warnings) == 0 {
		return "No errors or warnings"
	}

	var summary strings.Builder

	if len(h.errors) > 0 {
		summary.WriteString(fmt.Sprintf("Errors (%d):\n", len(h.errors)))
		for i, err := range h.errors {
			summary.WriteString(fmt.Sprintf("  %d. %s\n", i+1, err.Error()))
		}
	}

	if len(h.warnings) > 0 {
		if summary.Len() > 0 {
			summary.WriteString("\n")
		}
		summary.WriteString(fmt.Sprintf("Warnings (%d):\n", len(h.warnings)))
		for i, warning := range h.warnings {
			summary.WriteString(fmt.Sprintf("  %d. %s\n", i+1, warning))
		}
	}

	return summary.String()
}
