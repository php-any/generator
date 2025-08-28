package analyzer

import (
	"reflect"
	"strings"

	"github.com/php-any/generator/core"
)

// PackageAnalyzerImpl 包分析器实现
type PackageAnalyzerImpl struct {
	context *core.GeneratorContext
}

// 创建新的包分析器
func NewPackageAnalyzer(ctx *core.GeneratorContext) *PackageAnalyzerImpl {
	return &PackageAnalyzerImpl{
		context: ctx,
	}
}

// AnalyzeImports 分析导入
func (pa *PackageAnalyzerImpl) AnalyzeImports(types []*core.TypeInfo) ([]core.ImportInfo, error) {
	imports := make(map[string]core.ImportInfo)

	for _, typeInfo := range types {
		if typeInfo == nil {
			continue
		}

		// 收集当前类型的导入
		typeImports, err := pa.CollectDependencies(typeInfo)
		if err != nil {
			return nil, err
		}

		// 合并导入
		for _, imp := range typeImports {
			imports[imp.Path] = imp
		}
	}

	// 转换为切片
	result := make([]core.ImportInfo, 0, len(imports))
	for _, imp := range imports {
		result = append(result, imp)
	}

	return result, nil
}

// ResolveAlias 解析别名
func (pa *PackageAnalyzerImpl) ResolveAlias(pkgPath string) string {
	if pkgPath == "" {
		return "origami"
	}

	// 获取包前缀
	prefix := pa.GetPackagePrefix(pkgPath)
	if prefix == "" {
		prefix = pa.GetPackagePrefix(pkgPath)
	}

	// 如果前缀为空，使用默认前缀
	if prefix == "" {
		prefix = "origami"
	}

	return prefix + "src"
}

// CollectDependencies 收集依赖
func (pa *PackageAnalyzerImpl) CollectDependencies(typeInfo *core.TypeInfo) ([]core.ImportInfo, error) {
	if typeInfo == nil {
		return nil, nil
	}

	imports := make(map[string]core.ImportInfo)

	// 递归收集类型的依赖
	if err := pa.collectTypeDependencies(typeInfo, imports); err != nil {
		return nil, err
	}

	// 收集字段的依赖
	for _, field := range typeInfo.Fields {
		if field.Type != nil {
			if err := pa.collectTypeDependencies(field.Type, imports); err != nil {
				return nil, err
			}
		}
	}

	// 收集方法的依赖
	for _, method := range typeInfo.Methods {
		for _, param := range method.Parameters {
			if param.Type != nil {
				if err := pa.collectTypeDependencies(param.Type, imports); err != nil {
					return nil, err
				}
			}
		}

		for _, ret := range method.Returns {
			if err := pa.collectReturnDependencies(&ret, imports); err != nil {
				return nil, err
			}
		}
	}

	// 转换为切片
	result := make([]core.ImportInfo, 0, len(imports))
	for _, imp := range imports {
		result = append(result, imp)
	}

	return result, nil
}

// GetPackagePrefix 获取包前缀
func (pa *PackageAnalyzerImpl) GetPackagePrefix(pkgPath string) string {
	if pa.context.GetConfigManager() == nil {
		return ""
	}

	return pa.context.GetConfigManager().GetPackagePrefix(pkgPath)
}

// GetMappedPackage 获取映射的包
func (pa *PackageAnalyzerImpl) GetMappedPackage(sourcePkg string) string {
	if pa.context.GetConfigManager() == nil {
		return sourcePkg
	}

	mappedPkg, exists := pa.context.GetConfigManager().GetPackageMapping(sourcePkg)
	if !exists {
		return sourcePkg
	}

	return mappedPkg
}

// 收集类型依赖
func (pa *PackageAnalyzerImpl) collectTypeDependencies(typeInfo *core.TypeInfo, imports map[string]core.ImportInfo) error {
	if typeInfo == nil || typeInfo.Type == nil {
		return nil
	}

	t := typeInfo.Type

	// 处理基本类型
	switch t.Kind() {
	case reflect.Ptr, reflect.Slice, reflect.Array, reflect.Map:
		// 递归处理元素类型
		if err := pa.collectElementTypeDependencies(t, imports); err != nil {
			return err
		}
	case reflect.Struct, reflect.Interface:
		// 处理具名类型
		if err := pa.collectNamedTypeDependencies(t, imports); err != nil {
			return err
		}
	case reflect.Func:
		// 处理函数类型
		if err := pa.collectFunctionTypeDependencies(t, imports); err != nil {
			return err
		}
	}

	return nil
}

// 收集元素类型依赖
func (pa *PackageAnalyzerImpl) collectElementTypeDependencies(t reflect.Type, imports map[string]core.ImportInfo) error {
	// 处理指针类型
	if t.Kind() == reflect.Ptr {
		elemType := t.Elem()
		if elemType.PkgPath() != "" {
			pa.addImport(elemType.PkgPath(), imports)
		}
		return pa.collectTypeDependencies(&core.TypeInfo{Type: elemType}, imports)
	}

	// 处理切片和数组
	if t.Kind() == reflect.Slice || t.Kind() == reflect.Array {
		elemType := t.Elem()
		if elemType.PkgPath() != "" {
			pa.addImport(elemType.PkgPath(), imports)
		}
		return pa.collectTypeDependencies(&core.TypeInfo{Type: elemType}, imports)
	}

	// 处理映射
	if t.Kind() == reflect.Map {
		keyType := t.Key()
		elemType := t.Elem()

		if keyType.PkgPath() != "" {
			pa.addImport(keyType.PkgPath(), imports)
		}
		if elemType.PkgPath() != "" {
			pa.addImport(elemType.PkgPath(), imports)
		}

		if err := pa.collectTypeDependencies(&core.TypeInfo{Type: keyType}, imports); err != nil {
			return err
		}
		return pa.collectTypeDependencies(&core.TypeInfo{Type: elemType}, imports)
	}

	return nil
}

// 收集具名类型依赖
func (pa *PackageAnalyzerImpl) collectNamedTypeDependencies(t reflect.Type, imports map[string]core.ImportInfo) error {
	if t.PkgPath() == "" {
		return nil
	}

	// 检查是否需要映射
	mappedPkg := pa.GetMappedPackage(t.PkgPath())
	if mappedPkg != t.PkgPath() {
		pa.addImport(mappedPkg, imports)
	} else {
		pa.addImport(t.PkgPath(), imports)
	}

	return nil
}

// 收集函数类型依赖
func (pa *PackageAnalyzerImpl) collectFunctionTypeDependencies(t reflect.Type, imports map[string]core.ImportInfo) error {
	// 收集参数类型依赖
	for i := 0; i < t.NumIn(); i++ {
		paramType := t.In(i)
		if paramType.PkgPath() != "" {
			pa.addImport(paramType.PkgPath(), imports)
		}
		if err := pa.collectTypeDependencies(&core.TypeInfo{Type: paramType}, imports); err != nil {
			return err
		}
	}

	// 收集返回值类型依赖
	for i := 0; i < t.NumOut(); i++ {
		returnType := t.Out(i)
		if returnType.PkgPath() != "" {
			pa.addImport(returnType.PkgPath(), imports)
		}
		if err := pa.collectTypeDependencies(&core.TypeInfo{Type: returnType}, imports); err != nil {
			return err
		}
	}

	return nil
}

// 收集返回值依赖
func (pa *PackageAnalyzerImpl) collectReturnDependencies(ret *core.TypeInfo, imports map[string]core.ImportInfo) error {
	if ret == nil {
		return nil
	}

	// 这里可以添加特殊的返回值依赖收集逻辑
	// 例如处理error类型、接口类型等

	return nil
}

// 添加导入
func (pa *PackageAnalyzerImpl) addImport(pkgPath string, imports map[string]core.ImportInfo) {
	if pkgPath == "" {
		return
	}

	// 跳过标准库
	if !strings.Contains(pkgPath, ".") {
		return
	}

	alias := pa.ResolveAlias(pkgPath)
	imports[pkgPath] = core.ImportInfo{
		Path:  pkgPath,
		Alias: alias,
		Used:  true,
	}
}
