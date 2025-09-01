package scr

type Use struct {
	alias string
	path  string
}

type FileCache struct {
	Use map[string]Use
	// 记录文件中使用的导入
	Imports map[string]string
	// 记录导入的使用情况
	ImportUsage map[string]bool
}

// NewFileCache 创建新的文件缓存
func NewFileCache() *FileCache {
	return &FileCache{
		Use:         make(map[string]Use),
		Imports:     make(map[string]string),
		ImportUsage: make(map[string]bool),
	}
}

// AddImport 添加导入
func (fc *FileCache) AddImport(pkgPath, alias string) {
	if pkgPath != "" {
		fc.Imports[pkgPath] = alias
		fc.ImportUsage[pkgPath] = false // 初始化为未使用
	}
}

// MarkImportUsed 标记导入为已使用
func (fc *FileCache) MarkImportUsed(pkgPath string) {
	if pkgPath != "" {
		fc.ImportUsage[pkgPath] = true
	}
}

// GetImports 获取所有导入
func (fc *FileCache) GetImports() map[string]string {
	return fc.Imports
}

type Load struct {
	name     string
	typeName string
}

type PackageCache struct {
	Load map[string]Load
	Use  map[string]Use
}

type GroupCache struct {
	Config *Config
	// 当前递归层次（内部使用）
	CurrentDepth int
	// 已生成的类型缓存，防止重复生成和死循环
	generatedTypes map[string]bool
}

// NewGroupCache 创建新的 GroupCache 实例
func NewGroupCache(config *Config) *GroupCache {
	return &GroupCache{
		Config:         config,
		CurrentDepth:   0,
		generatedTypes: make(map[string]bool),
	}
}

// IsTypeGenerated 检查类型是否已生成
func (gc *GroupCache) IsTypeGenerated(typeKey string) bool {
	return gc.generatedTypes[typeKey]
}

// MarkTypeGenerated 标记类型为已生成
func (gc *GroupCache) MarkTypeGenerated(typeKey string) {
	gc.generatedTypes[typeKey] = true
}

// GlobalCache 全局缓存管理器
type GlobalCache struct {
	// 包级别的缓存
	packageCaches map[string]*PackageCache
}

// NewGlobalCache 创建新的全局缓存
func NewGlobalCache() *GlobalCache {
	return &GlobalCache{
		packageCaches: make(map[string]*PackageCache),
	}
}

// GetPackageCache 获取包缓存
func (gc *GlobalCache) GetPackageCache(pkgName string) *PackageCache {
	if cache, exists := gc.packageCaches[pkgName]; exists {
		return cache
	}

	cache := &PackageCache{
		Load: make(map[string]Load),
		Use:  make(map[string]Use),
	}
	gc.packageCaches[pkgName] = cache
	return cache
}

// RegisterClass 注册类到包缓存
func (gc *GlobalCache) RegisterClass(pkgName, typeName string) {
	cache := gc.GetPackageCache(pkgName)
	cache.Load[typeName] = Load{
		name:     typeName,
		typeName: "class",
	}
}

// RegisterFunction 注册函数到包缓存
func (gc *GlobalCache) RegisterFunction(pkgName, funcName string) {
	cache := gc.GetPackageCache(pkgName)
	cache.Load[funcName] = Load{
		name:     funcName,
		typeName: "func",
	}
}

// ListRegistered 列出包中已注册的类型和函数
func (gc *GlobalCache) ListRegistered(pkgName string) (classes, functions []string) {
	cache := gc.GetPackageCache(pkgName)

	for name, load := range cache.Load {
		if load.typeName == "class" {
			classes = append(classes, name)
		} else if load.typeName == "func" {
			functions = append(functions, name)
		}
	}

	return classes, functions
}

// 全局缓存实例
var globalCache = NewGlobalCache()
