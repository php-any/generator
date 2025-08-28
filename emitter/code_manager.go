package emitter

import (
	"fmt"
	"sort"
	"strings"

	"github.com/php-any/generator/core"
)

// CodeManager 代码管理器
type CodeManager struct {
	imports map[string]map[string]string // 包名 -> (别名 -> 路径)
	config  *core.GeneratorConfig
}

// NewCodeManager 创建新的代码管理器
func NewCodeManager(config *core.GeneratorConfig) *CodeManager {
	return &CodeManager{
		imports: make(map[string]map[string]string),
		config:  config,
	}
}

// AddImport 添加import
func (cm *CodeManager) AddImport(pkgName, importPath string) {
	if cm.imports[pkgName] == nil {
		cm.imports[pkgName] = make(map[string]string)
	}

	// 生成别名
	alias := cm.generateAlias(importPath)
	cm.imports[pkgName][alias] = importPath
}

// AddImportWithAlias 添加带别名的import
func (cm *CodeManager) AddImportWithAlias(pkgName, importPath, alias string) {
	if cm.imports[pkgName] == nil {
		cm.imports[pkgName] = make(map[string]string)
	}
	cm.imports[pkgName][alias] = importPath
}

// GetImports 获取包的imports
func (cm *CodeManager) GetImports(pkgName string) map[string]string {
	if imports, exists := cm.imports[pkgName]; exists {
		return imports
	}
	return make(map[string]string)
}

// GenerateImportsBlock 生成import块
func (cm *CodeManager) GenerateImportsBlock(pkgName string) string {
	imports := cm.GetImports(pkgName)
	if len(imports) == 0 {
		return ""
	}

	var importLines []string
	for alias, path := range imports {
		if alias == cm.getBaseName(path) {
			// 如果别名和包名相同，不需要别名
			importLines = append(importLines, fmt.Sprintf("\t\"%s\"", path))
		} else {
			// 需要别名
			importLines = append(importLines, fmt.Sprintf("\t%s \"%s\"", alias, path))
		}
	}

	// 排序imports
	sort.Strings(importLines)

	return fmt.Sprintf("import (\n%s\n)", strings.Join(importLines, "\n"))
}

// GeneratePackageDeclaration 生成包声明
func (cm *CodeManager) GeneratePackageDeclaration(pkgName string) string {
	return fmt.Sprintf("package %s", pkgName)
}

// GenerateFileHeader 生成文件头部（包声明 + imports）
func (cm *CodeManager) GenerateFileHeader(pkgName string) string {
	packageDecl := cm.GeneratePackageDeclaration(pkgName)
	importsBlock := cm.GenerateImportsBlock(pkgName)

	if importsBlock == "" {
		return packageDecl
	}

	return fmt.Sprintf("%s\n\n%s", packageDecl, importsBlock)
}

// GenerateFunctionCode 生成函数代码
func (cm *CodeManager) GenerateFunctionCode(pkgName, functionName string, params []core.ParameterInfo, returns []core.ReturnTemplateData) string {
	// 添加必要的imports
	cm.addFunctionImports(pkgName, params, returns)

	// 生成函数代码
	code := cm.GenerateFileHeader(pkgName)
	code += "\n\n"
	code += cm.generateFunctionBody(functionName, params, returns)

	return code
}

// GenerateClassCode 生成类代码
func (cm *CodeManager) GenerateClassCode(pkgName, className string, fields []core.FieldInfo, methods []core.MethodInfo) string {
	// 添加必要的imports
	cm.addClassImports(pkgName, fields, methods)

	// 生成类代码
	code := cm.GenerateFileHeader(pkgName)
	code += "\n\n"
	code += cm.generateClassBody(className, fields, methods)

	return code
}

// GenerateMethodCode 生成方法代码
func (cm *CodeManager) GenerateMethodCode(pkgName, className, methodName string, params []core.ParameterInfo, returns []core.ReturnTemplateData) string {
	// 添加必要的imports
	cm.addMethodImports(pkgName, params, returns)

	// 生成方法代码
	code := cm.GenerateFileHeader(pkgName)
	code += "\n\n"
	code += cm.generateMethodBody(className, methodName, params, returns)

	return code
}

// 生成别名
func (cm *CodeManager) generateAlias(importPath string) string {
	// 从路径中提取包名
	parts := strings.Split(importPath, "/")
	baseName := parts[len(parts)-1]

	// 处理特殊字符
	baseName = strings.ReplaceAll(baseName, "-", "")
	baseName = strings.ReplaceAll(baseName, "_", "")

	return baseName
}

// 获取包的基础名称
func (cm *CodeManager) getBaseName(importPath string) string {
	parts := strings.Split(importPath, "/")
	return parts[len(parts)-1]
}

// 添加函数相关的imports
func (cm *CodeManager) addFunctionImports(pkgName string, params []core.ParameterInfo, returns []core.ReturnTemplateData) {
	// 根据参数和返回值类型添加imports
	for _, param := range params {
		cm.addTypeImports(pkgName, param.Type.String())
	}

	for _, ret := range returns {
		cm.addTypeImports(pkgName, ret.Type)
	}
}

// 添加类相关的imports
func (cm *CodeManager) addClassImports(pkgName string, fields []core.FieldInfo, methods []core.MethodInfo) {
	// 根据字段类型添加imports
	for _, field := range fields {
		cm.addTypeImports(pkgName, field.Type.String())
	}

	// 根据方法参数和返回值类型添加imports
	for _, method := range methods {
		for _, param := range method.Parameters {
			cm.addTypeImports(pkgName, param.Type.String())
		}
		for _, ret := range method.Returns {
			cm.addTypeImports(pkgName, ret.Type.String())
		}
	}
}

// 添加方法相关的imports
func (cm *CodeManager) addMethodImports(pkgName string, params []core.ParameterInfo, returns []core.ReturnTemplateData) {
	// 根据参数和返回值类型添加imports
	for _, param := range params {
		cm.addTypeImports(pkgName, param.Type.String())
	}

	for _, ret := range returns {
		cm.addTypeImports(pkgName, ret.Type)
	}
}

// 根据类型添加imports
func (cm *CodeManager) addTypeImports(pkgName, typeName string) {
	// 检查是否是外部包类型
	if strings.Contains(typeName, ".") {
		parts := strings.Split(typeName, ".")
		if len(parts) == 2 {
			externalPkg := parts[0]
			// 检查是否需要添加import
			if !cm.isStandardLibrary(externalPkg) && !cm.isLocalPackage(externalPkg, pkgName) {
				// 这里可以根据配置映射包路径
				importPath := cm.mapPackagePath(externalPkg)
				cm.AddImport(pkgName, importPath)
			}
		}
	}
}

// 检查是否是标准库
func (cm *CodeManager) isStandardLibrary(pkgName string) bool {
	standardLibs := []string{
		"fmt", "os", "io", "bufio", "bytes", "strings", "strconv",
		"time", "math", "sort", "reflect", "encoding", "crypto",
		"context", "errors", "log", "path", "filepath", "net", "http",
	}

	for _, lib := range standardLibs {
		if pkgName == lib {
			return true
		}
	}
	return false
}

// 检查是否是本地包
func (cm *CodeManager) isLocalPackage(pkgName, currentPkg string) bool {
	return pkgName == currentPkg
}

// 映射包路径
func (cm *CodeManager) mapPackagePath(pkgName string) string {
	if cm.config != nil && cm.config.PackageMappings != nil {
		if mappedPath, exists := cm.config.PackageMappings[pkgName]; exists {
			return mappedPath
		}
	}

	// 如果没有找到映射，返回默认的包路径
	return fmt.Sprintf("github.com/php-any/origami/%s", pkgName)
}

// 生成函数体
func (cm *CodeManager) generateFunctionBody(functionName string, params []core.ParameterInfo, returns []core.ReturnTemplateData) string {
	// 这里可以根据需要生成更复杂的函数体
	return fmt.Sprintf(`// %sFunction 函数代理
type %sFunction struct {
	node.Node
}

// New%sFunction 创建新的函数代理
func New%sFunction() *%sFunction {
	return &%sFunction{}
}

// GetName 获取函数名
func (f *%sFunction) GetName() string {
	return "%s"
}

// Call 调用函数
func (f *%sFunction) Call(ctx data.Context, args []data.Value) (data.GetValue, data.Control) {
	// TODO: 实现函数调用逻辑
	return data.NewStringValue("function result"), nil
}`,
		functionName, functionName, functionName, functionName, functionName, functionName, functionName, functionName, functionName)
}

// 生成类体
func (cm *CodeManager) generateClassBody(className string, fields []core.FieldInfo, methods []core.MethodInfo) string {
	// 这里可以根据需要生成更复杂的类体
	return fmt.Sprintf(`// %sClass 类代理
type %sClass struct {
	node.Node
}

// New%sClass 创建新的类代理
func New%sClass() *%sClass {
	return &%sClass{}
}

// GetName 获取类名
func (c *%sClass) GetName() string {
	return "%s"
}`,
		className, className, className, className, className, className, className)
}

// 生成方法体
func (cm *CodeManager) generateMethodBody(className, methodName string, params []core.ParameterInfo, returns []core.ReturnTemplateData) string {
	// 这里可以根据需要生成更复杂的方法体
	return fmt.Sprintf(`// %sMethod 方法代理
type %sMethod struct {
	node.Node
}

// New%sMethod 创建新的方法代理
func New%sMethod() *%sMethod {
	return &%sMethod{}
}

// GetName 获取方法名
func (m *%sMethod) GetName() string {
	return "%s"
}

// Call 调用方法
func (m *%sMethod) Call(ctx data.Context, args []data.Value) (data.GetValue, data.Control) {
	// TODO: 实现方法调用逻辑
	return data.NewStringValue("method result"), nil
}`,
		methodName, methodName, methodName, methodName, methodName, methodName, methodName, methodName)
}
