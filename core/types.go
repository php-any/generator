package core

import (
	"reflect"
	"sort"
	"strings"
)

// TypeInfo 表示一个类型的完整信息
type TypeInfo struct {
	Type        reflect.Type
	PackagePath string
	PackageName string
	TypeName    string
	IsPointer   bool
	IsInterface bool
	IsStruct    bool
	IsFunction  bool
	Fields      []FieldInfo
	Methods     []MethodInfo
	Imports     []ImportInfo
	CacheKey    string
	// 新增：配置相关信息
	Config *TypeConfig
}

// TypeConfig 类型相关的配置信息
type TypeConfig struct {
	// 是否被黑名单阻止
	IsBlacklisted bool
	// 包前缀
	PackagePrefix string
	// 映射的目标包
	MappedPackage string
	// 自定义配置
	Custom map[string]interface{}
}

// FieldInfo 字段信息
type FieldInfo struct {
	Name       string
	Type       *TypeInfo
	IsExported bool
	Tag        reflect.StructTag
}

// MethodInfo 方法信息
type MethodInfo struct {
	Name       string
	Parameters []ParameterInfo
	Returns    []TypeInfo
	IsVariadic bool
	IsExported bool
	Receiver   *TypeInfo
}

// ParameterInfo 参数信息
type ParameterInfo struct {
	Name  string
	Type  *TypeInfo
	Index int
}

// ImportInfo 导入信息
type ImportInfo struct {
	Path     string
	Alias    string
	Used     bool
	Priority int
}

// FunctionInfo 函数信息
type FunctionInfo struct {
	Name       string
	Package    string
	Parameters []ParameterInfo
	Returns    []TypeInfo
	IsVariadic bool
}

// GeneratorConfig 生成器配置（移动到这里避免循环导入）
type GeneratorConfig struct {
	// 全局配置
	GlobalPrefix string
	OutputRoot   string
	MaxDepth     int
	Parallel     bool
	Verbose      bool

	// 黑名单配置
	Blacklist BlacklistConfig

	// 包前缀配置
	PackagePrefixes map[string]string

	// 依赖包映射配置
	PackageMappings map[string]string

	// 高级配置
	Advanced AdvancedConfig
}

// BlacklistConfig 黑名单配置
type BlacklistConfig struct {
	// 包路径黑名单
	Packages []string

	// 类型名称黑名单
	Types []string

	// 方法名称黑名单
	Methods []string

	// 使用正则表达式匹配
	UseRegex bool

	// 正则表达式模式
	Patterns []string
}

// AdvancedConfig 高级配置
type AdvancedConfig struct {
	// 是否生成调试信息
	Debug bool

	// 是否保留原始注释
	KeepComments bool

	// 是否生成测试文件
	GenerateTests bool

	// 自定义模板路径
	TemplatePath string

	// 缓存配置
	Cache CacheConfig
}

// CacheConfig 缓存配置
type CacheConfig struct {
	// 是否启用缓存
	Enabled bool

	// 缓存目录
	Directory string

	// 缓存过期时间（秒）
	TTL int64

	// 最大缓存大小（MB）
	MaxSize int64
}

// 创建新的TypeInfo实例
func NewTypeInfo(t reflect.Type) *TypeInfo {
	if t == nil {
		return nil
	}

	// 处理函数类型的名称
	typeName := t.Name()
	if t.Kind() == reflect.Func && typeName == "" {
		typeName = "func"
	}

	info := &TypeInfo{
		Type:        t,
		PackagePath: t.PkgPath(),
		PackageName: pkgBaseName(t.PkgPath()),
		TypeName:    typeName,
		IsPointer:   t.Kind() == reflect.Ptr,
		IsInterface: t.Kind() == reflect.Interface,
		IsStruct:    t.Kind() == reflect.Struct,
		IsFunction:  t.Kind() == reflect.Func,
		Imports:     make([]ImportInfo, 0),
		CacheKey:    generateCacheKey(t),
		Config: &TypeConfig{
			Custom: make(map[string]interface{}),
		},
	}

	// 生成缓存键
	return info
}

// generateCacheKey 生成类型缓存键
func generateCacheKey(t reflect.Type) string {
	if t == nil {
		return ""
	}
	return t.String()
}

// pkgBaseName 从包路径中提取基础包名
func pkgBaseName(pkgPath string) string {
	if pkgPath == "" {
		return ""
	}
	parts := strings.Split(pkgPath, "/")
	return parts[len(parts)-1]
}

// AddImport 添加导入信息
func (t *TypeInfo) AddImport(path, alias string) {
	// 检查是否已存在
	for i, imp := range t.Imports {
		if imp.Path == path {
			t.Imports[i].Used = true
			return
		}
	}

	t.Imports = append(t.Imports, ImportInfo{
		Path:  path,
		Alias: alias,
		Used:  true,
	})
}

// SortImports 对导入进行排序
func (t *TypeInfo) SortImports() {
	sort.Slice(t.Imports, func(i, j int) bool {
		return t.Imports[i].Path < t.Imports[j].Path
	})
}

// GetUsedImports 获取使用的导入
func (t *TypeInfo) GetUsedImports() []ImportInfo {
	var used []ImportInfo
	for _, imp := range t.Imports {
		if imp.Used {
			used = append(used, imp)
		}
	}
	return used
}

// IsBasicType 检查是否为基本类型
func (t *TypeInfo) IsBasicType() bool {
	if t.Type == nil {
		return false
	}
	kind := t.Type.Kind()
	return kind == reflect.Bool ||
		kind == reflect.Int || kind == reflect.Int8 || kind == reflect.Int16 || kind == reflect.Int32 || kind == reflect.Int64 ||
		kind == reflect.Uint || kind == reflect.Uint8 || kind == reflect.Uint16 || kind == reflect.Uint32 || kind == reflect.Uint64 ||
		kind == reflect.Float32 || kind == reflect.Float64 ||
		kind == reflect.String ||
		t.Type.PkgPath() == ""
}

// IsSliceType 检查是否为切片类型
func (t *TypeInfo) IsSliceType() bool {
	return t.Type != nil && t.Type.Kind() == reflect.Slice
}

// IsMapType 检查是否为映射类型
func (t *TypeInfo) IsMapType() bool {
	return t.Type != nil && t.Type.Kind() == reflect.Map
}

// IsChanType 检查是否为通道类型
func (t *TypeInfo) IsChanType() bool {
	return t.Type != nil && t.Type.Kind() == reflect.Chan
}

// GetElementType 获取元素类型（用于切片、数组、映射等）
func (t *TypeInfo) GetElementType() *TypeInfo {
	if t.Type == nil {
		return nil
	}
	switch t.Type.Kind() {
	case reflect.Slice, reflect.Array:
		return NewTypeInfo(t.Type.Elem())
	case reflect.Map:
		return NewTypeInfo(t.Type.Elem())
	case reflect.Ptr:
		return NewTypeInfo(t.Type.Elem())
	default:
		return nil
	}
}

// GetKeyType 获取键类型（用于映射）
func (t *TypeInfo) GetKeyType() *TypeInfo {
	if t.Type != nil && t.Type.Kind() == reflect.Map {
		return NewTypeInfo(t.Type.Key())
	}
	return nil
}

// String 返回类型的字符串表示
func (t *TypeInfo) String() string {
	if t.Type == nil {
		return "<nil>"
	}
	return t.Type.String()
}

// Equal 比较两个TypeInfo是否相等
func (t *TypeInfo) Equal(other *TypeInfo) bool {
	if t == nil || other == nil {
		return t == other
	}
	return t.CacheKey == other.CacheKey
}

// 创建默认配置
func NewDefaultConfig() *GeneratorConfig {
	config := &GeneratorConfig{
		GlobalPrefix:    "origami",
		OutputRoot:      "generated",
		MaxDepth:        3,
		Parallel:        false,
		Verbose:         false,
		Blacklist:       BlacklistConfig{},
		PackagePrefixes: make(map[string]string),
		PackageMappings: make(map[string]string),
		Advanced: AdvancedConfig{
			Debug:         false,
			KeepComments:  true,
			GenerateTests: false,
			TemplatePath:  "./templates",
			Cache: CacheConfig{
				Enabled:   true,
				Directory: "./cache",
				TTL:       3600,
				MaxSize:   100,
			},
		},
	}

	// 设置默认包映射规则
	config.setDefaultPackageMappings()

	return config
}

// 设置默认包映射规则
func (c *GeneratorConfig) setDefaultPackageMappings() {
	defaultMappings := map[string]string{
		"applicationsrc": "github.com/php-any/origami/application",
		"contextsrc":     "github.com/php-any/origami/context",
		"eventssrc":      "github.com/php-any/origami/events",
		"httpsrc":        "github.com/php-any/origami/http",
		"windowsrc":      "github.com/php-any/origami/window",
		"menusrc":        "github.com/php-any/origami/menu",
		"dialogsrc":      "github.com/php-any/origami/dialog",
		"clipboardsrc":   "github.com/php-any/origami/clipboard",
		"keyboardsrc":    "github.com/php-any/origami/keyboard",
		"mousesrc":       "github.com/php-any/origami/mouse",
		"devicesrc":      "github.com/php-any/origami/device",
		"storagesrc":     "github.com/php-any/origami/storage",
		"networkingsrc":  "github.com/php-any/origami/networking",
		"securitysrc":    "github.com/php-any/origami/security",
		"uilibsrc":       "github.com/php-any/origami/uilib",
	}

	for source, target := range defaultMappings {
		c.PackageMappings[source] = target
	}
}

// 合并配置
func (c *GeneratorConfig) Merge(other *GeneratorConfig) {
	if other == nil {
		return
	}

	// 合并基本配置
	if other.GlobalPrefix != "" {
		c.GlobalPrefix = other.GlobalPrefix
	}
	if other.OutputRoot != "" {
		c.OutputRoot = other.OutputRoot
	}
	if other.MaxDepth > 0 {
		c.MaxDepth = other.MaxDepth
	}
	c.Parallel = other.Parallel
	c.Verbose = other.Verbose

	// 合并黑名单配置
	c.Blacklist.Merge(&other.Blacklist)

	// 合并包前缀配置
	for pkg, prefix := range other.PackagePrefixes {
		c.PackagePrefixes[pkg] = prefix
	}

	// 合并包映射配置
	for source, target := range other.PackageMappings {
		c.PackageMappings[source] = target
	}

	// 合并高级配置
	c.Advanced.Merge(&other.Advanced)
}

// 合并黑名单配置
func (bc *BlacklistConfig) Merge(other *BlacklistConfig) {
	if other == nil {
		return
	}

	// 合并包黑名单
	bc.Packages = append(bc.Packages, other.Packages...)
	bc.Packages = removeDuplicates(bc.Packages)

	// 合并类型黑名单
	bc.Types = append(bc.Types, other.Types...)
	bc.Types = removeDuplicates(bc.Types)

	// 合并方法黑名单
	bc.Methods = append(bc.Methods, other.Methods...)
	bc.Methods = removeDuplicates(bc.Methods)

	// 合并正则表达式配置
	bc.UseRegex = bc.UseRegex || other.UseRegex
	bc.Patterns = append(bc.Patterns, other.Patterns...)
	bc.Patterns = removeDuplicates(bc.Patterns)
}

// 合并高级配置
func (ac *AdvancedConfig) Merge(other *AdvancedConfig) {
	if other == nil {
		return
	}

	ac.Debug = ac.Debug || other.Debug
	ac.KeepComments = ac.KeepComments || other.KeepComments
	ac.GenerateTests = ac.GenerateTests || other.GenerateTests

	if other.TemplatePath != "" {
		ac.TemplatePath = other.TemplatePath
	}

	ac.Cache.Merge(&other.Cache)
}

// 合并缓存配置
func (cc *CacheConfig) Merge(other *CacheConfig) {
	if other == nil {
		return
	}

	cc.Enabled = cc.Enabled || other.Enabled

	if other.Directory != "" {
		cc.Directory = other.Directory
	}

	if other.TTL > 0 {
		cc.TTL = other.TTL
	}

	if other.MaxSize > 0 {
		cc.MaxSize = other.MaxSize
	}
}

// 移除重复项
func removeDuplicates(slice []string) []string {
	seen := make(map[string]bool)
	result := make([]string, 0)

	for _, item := range slice {
		if !seen[item] {
			seen[item] = true
			result = append(result, item)
		}
	}

	return result
}

// 获取包前缀
func (c *GeneratorConfig) GetPackagePrefix(pkgPath string) string {
	// 检查是否有特定包的前缀配置
	if prefix, exists := c.PackagePrefixes[pkgPath]; exists {
		return prefix
	}

	// 检查是否有通配符匹配
	for pattern, prefix := range c.PackagePrefixes {
		if strings.HasSuffix(pattern, "/*") {
			basePattern := strings.TrimSuffix(pattern, "/*")
			if strings.HasPrefix(pkgPath, basePattern) {
				return prefix
			}
		}
	}

	// 返回全局前缀
	return c.GlobalPrefix
}

// 获取包映射
func (c *GeneratorConfig) GetPackageMapping(sourcePkg string) (string, bool) {
	targetPkg, exists := c.PackageMappings[sourcePkg]
	return targetPkg, exists
}

// 检查包是否在黑名单中
func (c *GeneratorConfig) IsPackageBlacklisted(pkgPath string) bool {
	for _, blacklistedPkg := range c.Blacklist.Packages {
		if c.matchesPattern(pkgPath, blacklistedPkg) {
			return true
		}
	}
	return false
}

// 检查类型是否在黑名单中
func (c *GeneratorConfig) IsTypeBlacklisted(typeName string) bool {
	for _, blacklistedType := range c.Blacklist.Types {
		if c.matchesPattern(typeName, blacklistedType) {
			return true
		}
	}
	return false
}

// 检查方法是否在黑名单中
func (c *GeneratorConfig) IsMethodBlacklisted(methodName string) bool {
	for _, blacklistedMethod := range c.Blacklist.Methods {
		if c.matchesPattern(methodName, blacklistedMethod) {
			return true
		}
	}
	return false
}

// 匹配模式
func (c *GeneratorConfig) matchesPattern(value, pattern string) bool {
	// 简单匹配
	if value == pattern {
		return true
	}

	// 通配符匹配
	if strings.Contains(pattern, "*") {
		return c.matchesWildcard(value, pattern)
	}

	return false
}

// 通配符匹配
func (c *GeneratorConfig) matchesWildcard(value, pattern string) bool {
	// 将通配符模式转换为简单的字符串匹配
	// 这里实现简单的通配符匹配，可以根据需要扩展为正则表达式
	parts := strings.Split(pattern, "*")
	if len(parts) == 1 {
		return value == pattern
	}

	// 检查是否以第一个部分开始
	if parts[0] != "" && !strings.HasPrefix(value, parts[0]) {
		return false
	}

	// 检查是否以最后一个部分结束
	lastPart := parts[len(parts)-1]
	if lastPart != "" && !strings.HasSuffix(value, lastPart) {
		return false
	}

	// 检查中间部分
	currentIndex := 0
	for i := 1; i < len(parts)-1; i++ {
		if parts[i] == "" {
			continue
		}
		index := strings.Index(value[currentIndex:], parts[i])
		if index == -1 {
			return false
		}
		currentIndex += index + len(parts[i])
	}

	return true
}

// lowerFirst 将标识符首字母小写
func lowerFirst(s string) string {
	if s == "" {
		return s
	}
	runes := []rune(s)
	runes[0] = []rune(strings.ToLower(string(runes[0])))[0]
	return string(runes)
}

// upperFirst 将标识符首字母大写
func upperFirst(s string) string {
	if s == "" {
		return s
	}
	runes := []rune(s)
	runes[0] = []rune(strings.ToUpper(string(runes[0])))[0]
	return string(runes)
}
