package analyzer

import (
	"fmt"
	"reflect"
	"sync"

	"github.com/php-any/generator/core"
)

// TypeAnalyzerImpl 类型分析器实现
type TypeAnalyzerImpl struct {
	context    *core.GeneratorContext
	typeCache  map[string]*core.TypeInfo
	cacheMutex sync.RWMutex
}

// 创建新的类型分析器
func NewTypeAnalyzer(ctx *core.GeneratorContext) *TypeAnalyzerImpl {
	return &TypeAnalyzerImpl{
		context:   ctx,
		typeCache: make(map[string]*core.TypeInfo),
	}
}

// AnalyzeType 分析类型
func (ta *TypeAnalyzerImpl) AnalyzeType(t reflect.Type) (*core.TypeInfo, error) {
	return ta.analyzeTypeWithCache(t, make(map[string]bool))
}

// analyzeTypeWithCache 带缓存分析类型，防止循环引用
func (ta *TypeAnalyzerImpl) analyzeTypeWithCache(t reflect.Type, analyzing map[string]bool) (*core.TypeInfo, error) {
	if t == nil {
		return nil, core.NewGeneratorError(core.ErrCodeTypeAnalysis, "cannot analyze nil type", nil)
	}

	typeKey := t.String()

	// 检查缓存
	if cached, exists := ta.GetCachedType(typeKey); exists {
		ta.context.Metrics.IncrementCacheHits()
		return cached, nil
	}

	// 检查是否正在分析中（防止循环引用）
	if analyzing[typeKey] {
		// 返回一个简化的类型信息，避免无限递归
		info := core.NewTypeInfo(t)
		if info == nil {
			info = &core.TypeInfo{
				Type:        t,
				PackagePath: t.PkgPath(),
				PackageName: "", // 简化处理
				TypeName:    t.Name(),
				IsPointer:   t.Kind() == reflect.Ptr,
				IsInterface: t.Kind() == reflect.Interface,
				IsStruct:    t.Kind() == reflect.Struct,
				IsFunction:  t.Kind() == reflect.Func,
				Imports:     make([]core.ImportInfo, 0),
				CacheKey:    t.String(),
				Config: &core.TypeConfig{
					Custom: make(map[string]interface{}),
				},
			}
		}
		return info, nil
	}

	// 标记为正在分析
	analyzing[typeKey] = true
	defer delete(analyzing, typeKey)

	// 创建新的类型信息
	info := core.NewTypeInfo(t)
	if info == nil {
		return nil, core.NewGeneratorError(core.ErrCodeTypeAnalysis, "failed to create type info", nil)
	}

	// 先缓存类型信息（在详细分析之前先缓存，防止循环引用）
	ta.CacheType(typeKey, info)

	// 分析字段和方法
	if err := ta.analyzeFieldsWithCache(info, analyzing); err != nil {
		return nil, err
	}

	if err := ta.analyzeMethodsWithCache(info, analyzing); err != nil {
		return nil, err
	}

	// 应用配置
	if err := ta.ApplyConfig(ta.context.GetConfigManager().GetConfig()); err != nil {
		return nil, err
	}

	// 检查是否被允许
	if !ta.IsTypeAllowed(info) {
		return nil, core.NewGeneratorError(core.ErrCodeTypeBlacklisted, "type is blacklisted", nil)
	}

	ta.context.Metrics.IncrementTypesAnalyzed()

	return info, nil
}

// AnalyzeFunction 分析函数
func (ta *TypeAnalyzerImpl) AnalyzeFunction(fn any) (*core.FunctionInfo, error) {
	if fn == nil {
		return nil, core.NewGeneratorError(core.ErrCodeTypeAnalysis, "cannot analyze nil function", nil)
	}

	val := reflect.ValueOf(fn)
	if val.Kind() != reflect.Func {
		return nil, core.NewGeneratorError(core.ErrCodeTypeAnalysis, "input is not a function", nil)
	}

	t := val.Type()
	info := &core.FunctionInfo{
		Package:    extractPackageFromFunction(val),
		IsVariadic: t.IsVariadic(),
	}

	// 分析参数
	for i := 0; i < t.NumIn(); i++ {
		paramType := t.In(i)
		typeInfo, err := ta.AnalyzeType(paramType)
		if err != nil {
			return nil, err
		}

		param := core.ParameterInfo{
			Name:  fmt.Sprintf("param%d", i),
			Type:  typeInfo,
			Index: i,
		}
		info.Parameters = append(info.Parameters, param)
	}

	// 分析返回值
	for i := 0; i < t.NumOut(); i++ {
		returnType := t.Out(i)
		typeInfo, err := ta.AnalyzeType(returnType)
		if err != nil {
			return nil, err
		}

		info.Returns = append(info.Returns, *typeInfo)
	}

	return info, nil
}

// AnalyzeMethod 分析方法
func (ta *TypeAnalyzerImpl) AnalyzeMethod(m reflect.Method) (*core.MethodInfo, error) {
	return ta.analyzeMethodWithCache(m, make(map[string]bool))
}

// analyzeMethodWithCache 带缓存分析方法
func (ta *TypeAnalyzerImpl) analyzeMethodWithCache(m reflect.Method, analyzing map[string]bool) (*core.MethodInfo, error) {
	info := &core.MethodInfo{
		Name:       m.Name,
		IsExported: m.PkgPath == "",
		IsVariadic: m.Type.IsVariadic(),
	}

	// 分析参数（跳过接收者）
	for i := 1; i < m.Type.NumIn(); i++ {
		paramType := m.Type.In(i)
		typeInfo, err := ta.analyzeTypeWithCache(paramType, analyzing)
		if err != nil {
			return nil, err
		}

		param := core.ParameterInfo{
			Name:  fmt.Sprintf("param%d", i-1),
			Type:  typeInfo,
			Index: i - 1,
		}
		info.Parameters = append(info.Parameters, param)
	}

	// 分析返回值
	for i := 0; i < m.Type.NumOut(); i++ {
		returnType := m.Type.Out(i)
		typeInfo, err := ta.analyzeTypeWithCache(returnType, analyzing)
		if err != nil {
			return nil, err
		}

		info.Returns = append(info.Returns, *typeInfo)
	}

	return info, nil
}

// GetCachedType 获取缓存的类型
func (ta *TypeAnalyzerImpl) GetCachedType(key string) (*core.TypeInfo, bool) {
	ta.cacheMutex.RLock()
	defer ta.cacheMutex.RUnlock()
	info, exists := ta.typeCache[key]
	return info, exists
}

// CacheType 缓存类型
func (ta *TypeAnalyzerImpl) CacheType(key string, info *core.TypeInfo) {
	ta.cacheMutex.Lock()
	defer ta.cacheMutex.Unlock()
	ta.typeCache[key] = info
}

// ApplyConfig 应用配置
func (ta *TypeAnalyzerImpl) ApplyConfig(config interface{}) error {
	if config == nil {
		return nil
	}

	_, ok := config.(*core.GeneratorConfig)
	if !ok {
		return core.NewGeneratorError(core.ErrCodeConfigInvalid, "invalid config type", nil)
	}

	// 这里可以应用配置到类型信息上
	// 例如设置包前缀、检查黑名单等
	// 具体实现根据需求而定

	return nil
}

// IsTypeAllowed 检查类型是否被允许
func (ta *TypeAnalyzerImpl) IsTypeAllowed(typeInfo *core.TypeInfo) bool {
	if typeInfo == nil {
		return false
	}

	config := ta.context.GetConfigManager()
	if config == nil {
		return true // 没有配置管理器时默认允许
	}

	// 检查包是否在黑名单中
	if config.IsPackageBlacklisted(typeInfo.PackagePath) {
		return false
	}

	// 检查类型是否在黑名单中
	if genConfig := config.GetConfig(); genConfig != nil {
		if genConfig.IsTypeBlacklisted(typeInfo.TypeName) {
			return false
		}
	}

	return true
}

// CollectDependencies 收集类型的所有依赖
func (ta *TypeAnalyzerImpl) CollectDependencies(typeInfo *core.TypeInfo, maxDepth int) ([]*core.TypeInfo, error) {
	return ta.collectDependenciesWithCache(typeInfo, maxDepth, 0, make(map[string]bool))
}

// collectDependenciesWithCache 带缓存的依赖收集
func (ta *TypeAnalyzerImpl) collectDependenciesWithCache(typeInfo *core.TypeInfo, maxDepth, currentDepth int, collected map[string]bool) ([]*core.TypeInfo, error) {
	if typeInfo == nil || currentDepth > maxDepth {
		return nil, nil
	}

	var dependencies []*core.TypeInfo

	// 检查是否已经收集过
	if collected[typeInfo.CacheKey] {
		return dependencies, nil
	}

	// 标记为已收集
	collected[typeInfo.CacheKey] = true

	// 调试信息
	if typeInfo.TypeName == "" {
		fmt.Printf("分析依赖: [空类型名] %s (包: %s, 种类: %s) (深度: %d)\n",
			typeInfo.Type.String(), typeInfo.PackagePath, typeInfo.Type.Kind().String(), currentDepth)
	} else {
		fmt.Printf("分析依赖: %s (包: %s) (深度: %d)\n", typeInfo.TypeName, typeInfo.PackagePath, currentDepth)
	}

	// 如果是结构体，需要收集字段类型
	if typeInfo.IsStruct {
		for _, field := range typeInfo.Fields {
			if field.Type != nil && !ta.isBasicType(field.Type) {
				fieldDeps, err := ta.collectDependenciesWithCache(field.Type, maxDepth, currentDepth+1, collected)
				if err != nil {
					return nil, err
				}
				dependencies = append(dependencies, fieldDeps...)
			}
		}
	}

	// 收集方法参数和返回值的类型
	for _, method := range typeInfo.Methods {
		// 收集参数类型
		for _, param := range method.Parameters {
			if param.Type != nil && !ta.isBasicType(param.Type) {
				paramDeps, err := ta.collectDependenciesWithCache(param.Type, maxDepth, currentDepth+1, collected)
				if err != nil {
					return nil, err
				}
				dependencies = append(dependencies, paramDeps...)
			}
		}

		// 收集返回值类型
		for _, ret := range method.Returns {
			if !ta.isBasicType(&ret) {
				retDeps, err := ta.collectDependenciesWithCache(&ret, maxDepth, currentDepth+1, collected)
				if err != nil {
					return nil, err
				}
				dependencies = append(dependencies, retDeps...)
			}
		}
	}

	// 如果当前类型不是基本类型且有包路径，将其添加到依赖中
	if !ta.isBasicType(typeInfo) && typeInfo.PackagePath != "" {
		dependencies = append(dependencies, typeInfo)
	} else if typeInfo.Type != nil && typeInfo.Type.Kind() == reflect.Ptr {
		// 对于指针类型，检查指向的类型
		elemType := typeInfo.Type.Elem()
		elemTypeInfo, err := ta.analyzeTypeWithCache(elemType, make(map[string]bool))
		if err != nil {
			return nil, err
		}

		if !ta.isBasicType(elemTypeInfo) && elemTypeInfo.PackagePath != "" {
			// 将指针指向的类型添加到依赖中
			dependencies = append(dependencies, elemTypeInfo)

			// 递归分析指针指向的类型
			elemDeps, err := ta.collectDependenciesWithCache(elemTypeInfo, maxDepth, currentDepth+1, collected)
			if err != nil {
				return nil, err
			}
			dependencies = append(dependencies, elemDeps...)
		}
		if elemTypeInfo.TypeName == "Request" && elemTypeInfo.PackagePath == "net/http" {
			fmt.Printf("发现 Request 类型，在指针处理中\n")
		}
	} else if typeInfo.Type != nil && typeInfo.Type.Kind() == reflect.Slice {
		// 对于切片类型，检查元素类型
		elemType := typeInfo.Type.Elem()
		elemTypeInfo := &core.TypeInfo{
			Type:        elemType,
			PackagePath: elemType.PkgPath(),
			TypeName:    elemType.Name(),
			IsPointer:   elemType.Kind() == reflect.Ptr,
			IsInterface: elemType.Kind() == reflect.Interface,
			IsStruct:    elemType.Kind() == reflect.Struct,
			IsFunction:  elemType.Kind() == reflect.Func,
			CacheKey:    elemType.String(),
		}
		if !ta.isBasicType(elemTypeInfo) && elemTypeInfo.PackagePath != "" {
			// 将切片元素类型添加到依赖中
			dependencies = append(dependencies, elemTypeInfo)
		}
	}

	return dependencies, nil
}

// isBasicType 检查是否为基本类型
func (ta *TypeAnalyzerImpl) isBasicType(typeInfo *core.TypeInfo) bool {
	if typeInfo == nil || typeInfo.Type == nil {
		return false
	}

	kind := typeInfo.Type.Kind()

	// 基本类型
	if kind == reflect.Bool ||
		kind == reflect.Int || kind == reflect.Int8 || kind == reflect.Int16 || kind == reflect.Int32 || kind == reflect.Int64 ||
		kind == reflect.Uint || kind == reflect.Uint8 || kind == reflect.Uint16 || kind == reflect.Uint32 || kind == reflect.Uint64 ||
		kind == reflect.Float32 || kind == reflect.Float64 ||
		kind == reflect.String {
		return true
	}

	// 内置接口类型
	if typeInfo.TypeName == "error" {
		return true
	}

	// 标准库类型（不需要生成代理）- 但保留 http 包
	if typeInfo.PackagePath == "time" ||
		typeInfo.PackagePath == "fmt" ||
		typeInfo.PackagePath == "io" ||
		typeInfo.PackagePath == "os" ||
		typeInfo.PackagePath == "context" {
		return true
	}

	// http 包中的类型需要生成代理
	if typeInfo.PackagePath == "net/http" {
		return false
	}

	return false
}

// 分析字段
func (ta *TypeAnalyzerImpl) analyzeFields(info *core.TypeInfo) error {
	return ta.analyzeFieldsWithCache(info, make(map[string]bool))
}

// analyzeFieldsWithCache 带缓存分析字段
func (ta *TypeAnalyzerImpl) analyzeFieldsWithCache(info *core.TypeInfo, analyzing map[string]bool) error {
	if info.Type == nil || info.Type.Kind() != reflect.Struct {
		return nil
	}

	for i := 0; i < info.Type.NumField(); i++ {
		field := info.Type.Field(i)
		if field.PkgPath != "" {
			// 跳过非导出字段
			continue
		}

		fieldType, err := ta.analyzeTypeWithCache(field.Type, analyzing)
		if err != nil {
			return err
		}

		fieldInfo := core.FieldInfo{
			Name:       field.Name,
			Type:       fieldType,
			IsExported: field.PkgPath == "",
			Tag:        field.Tag,
		}

		info.Fields = append(info.Fields, fieldInfo)
	}

	return nil
}

// 分析方法
func (ta *TypeAnalyzerImpl) analyzeMethods(info *core.TypeInfo) error {
	return ta.analyzeMethodsWithCache(info, make(map[string]bool))
}

// analyzeMethodsWithCache 带缓存分析方法
func (ta *TypeAnalyzerImpl) analyzeMethodsWithCache(info *core.TypeInfo, analyzing map[string]bool) error {
	if info.Type == nil {
		return nil
	}

	// 分析值接收者的方法
	if info.Type.Kind() == reflect.Struct {
		fmt.Printf("分析结构体 %s 的方法 (值接收者): %d 个方法\n", info.TypeName, info.Type.NumMethod())
		if info.TypeName == "Request" {
			fmt.Printf("Request 结构体详细信息: %s\n", info.Type.String())
		}
		for i := 0; i < info.Type.NumMethod(); i++ {
			method := info.Type.Method(i)
			methodInfo, err := ta.analyzeMethodWithCache(method, analyzing)
			if err != nil {
				return err
			}

			// 检查方法是否被允许
			if genConfig := ta.context.GetConfigManager().GetConfig(); genConfig != nil {
				if genConfig.IsMethodBlacklisted(methodInfo.Name) {
					continue // 跳过黑名单中的方法
				}
			}

			info.Methods = append(info.Methods, *methodInfo)
		}
	}

	// 分析指针接收者的方法
	ptrType := reflect.PtrTo(info.Type)
	fmt.Printf("分析结构体 %s 的方法 (指针接收者): %d 个方法\n", info.TypeName, ptrType.NumMethod())
	for i := 0; i < ptrType.NumMethod(); i++ {
		method := ptrType.Method(i)
		methodInfo, err := ta.analyzeMethodWithCache(method, analyzing)
		if err != nil {
			return err
		}

		// 检查方法是否被允许
		if genConfig := ta.context.GetConfigManager().GetConfig(); genConfig != nil {
			if genConfig.IsMethodBlacklisted(methodInfo.Name) {
				continue // 跳过黑名单中的方法
			}
		}

		info.Methods = append(info.Methods, *methodInfo)
	}

	return nil
}

// 从函数中提取包信息
func extractPackageFromFunction(val reflect.Value) string {
	if !val.IsValid() {
		return ""
	}

	// 这里可以尝试从函数名中提取包信息
	// 暂时返回空字符串，后续可以完善
	return ""
}
