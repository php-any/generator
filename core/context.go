package core

import (
	"sync"
)

// GeneratorContext 生成器上下文
type GeneratorContext struct {
	Options        *GenOptions
	GeneratedTypes map[string]bool
	TypeCache      map[string]*TypeInfo
	ErrorHandler   ErrorHandler
	Metrics        *Metrics
	ConfigManager  ConfigManager
	mutex          sync.RWMutex
}

// GenOptions 生成选项
type GenOptions struct {
	OutputRoot   string
	NamePrefix   string
	MaxDepth     int
	Parallel     bool
	Verbose      bool
	TemplatePath string
	ConfigFile   string
}

// ErrorHandler 错误处理器接口
type ErrorHandler interface {
	HandleError(err error)
	HandleWarning(msg string)
	HasErrors() bool
	GetErrors() []error
	GetWarnings() []string
}

// ConfigManager 配置管理器接口
type ConfigManager interface {
	LoadConfig(path string) error
	ValidateConfig() error
	IsPackageBlacklisted(pkgPath string) bool
	GetPackagePrefix(pkgPath string) string
	GetPackageMapping(sourcePkg string) (string, bool)
	GetGlobalPrefix() string
	GetConfig() *GeneratorConfig
}

// Metrics 指标收集器
type Metrics struct {
	TypesAnalyzed  int64
	FilesGenerated int64
	ErrorsCount    int64
	WarningsCount  int64
	GenerationTime int64 // 毫秒
	CacheHits      int64
	CacheMisses    int64
	mutex          sync.RWMutex
}

// 创建新的生成器上下文
func NewGeneratorContext(options *GenOptions) *GeneratorContext {
	if options == nil {
		options = &GenOptions{}
	}

	return &GeneratorContext{
		Options:        options,
		GeneratedTypes: make(map[string]bool),
		TypeCache:      make(map[string]*TypeInfo),
		ErrorHandler:   NewDefaultErrorHandler(),
		Metrics:        &Metrics{},
		ConfigManager:  nil, // 将在配置管理器中设置
	}
}

// IsTypeGenerated 检查类型是否已生成
func (ctx *GeneratorContext) IsTypeGenerated(typeKey string) bool {
	ctx.mutex.RLock()
	defer ctx.mutex.RUnlock()
	return ctx.GeneratedTypes[typeKey]
}

// MarkTypeGenerated 标记类型为已生成
func (ctx *GeneratorContext) MarkTypeGenerated(typeKey string) {
	ctx.mutex.Lock()
	defer ctx.mutex.Unlock()
	ctx.GeneratedTypes[typeKey] = true
}

// GetCachedType 获取缓存的类型信息
func (ctx *GeneratorContext) GetCachedType(key string) (*TypeInfo, bool) {
	ctx.mutex.RLock()
	defer ctx.mutex.RUnlock()
	info, exists := ctx.TypeCache[key]
	return info, exists
}

// CacheType 缓存类型信息
func (ctx *GeneratorContext) CacheType(key string, info *TypeInfo) {
	ctx.mutex.Lock()
	defer ctx.mutex.Unlock()
	ctx.TypeCache[key] = info
	ctx.Metrics.IncrementCacheMisses()
}

// GetTypeCacheSize 获取类型缓存大小
func (ctx *GeneratorContext) GetTypeCacheSize() int {
	ctx.mutex.RLock()
	defer ctx.mutex.RUnlock()
	return len(ctx.TypeCache)
}

// ClearCache 清除缓存
func (ctx *GeneratorContext) ClearCache() {
	ctx.mutex.Lock()
	defer ctx.mutex.Unlock()
	ctx.TypeCache = make(map[string]*TypeInfo)
}

// SetConfigManager 设置配置管理器
func (ctx *GeneratorContext) SetConfigManager(cm ConfigManager) {
	ctx.mutex.Lock()
	defer ctx.mutex.Unlock()
	ctx.ConfigManager = cm
}

// GetConfigManager 获取配置管理器
func (ctx *GeneratorContext) GetConfigManager() ConfigManager {
	ctx.mutex.RLock()
	defer ctx.mutex.RUnlock()
	return ctx.ConfigManager
}

// 指标相关方法
func (m *Metrics) IncrementTypesAnalyzed() {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	m.TypesAnalyzed++
}

func (m *Metrics) IncrementFilesGenerated() {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	m.FilesGenerated++
}

func (m *Metrics) IncrementErrors() {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	m.ErrorsCount++
}

func (m *Metrics) IncrementWarnings() {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	m.WarningsCount++
}

func (m *Metrics) IncrementCacheHits() {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	m.CacheHits++
}

func (m *Metrics) IncrementCacheMisses() {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	m.CacheMisses++
}

func (m *Metrics) SetGenerationTime(timeMs int64) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	m.GenerationTime = timeMs
}

// GetMetrics 获取指标快照
func (m *Metrics) GetMetrics() *MetricsSnapshot {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	return &MetricsSnapshot{
		TypesAnalyzed:  m.TypesAnalyzed,
		FilesGenerated: m.FilesGenerated,
		ErrorsCount:    m.ErrorsCount,
		WarningsCount:  m.WarningsCount,
		GenerationTime: m.GenerationTime,
		CacheHits:      m.CacheHits,
		CacheMisses:    m.CacheMisses,
		CacheHitRate:   float64(m.CacheHits) / float64(m.CacheHits+m.CacheMisses),
	}
}

// MetricsSnapshot 指标快照
type MetricsSnapshot struct {
	TypesAnalyzed  int64
	FilesGenerated int64
	ErrorsCount    int64
	WarningsCount  int64
	GenerationTime int64
	CacheHits      int64
	CacheMisses    int64
	CacheHitRate   float64
}
