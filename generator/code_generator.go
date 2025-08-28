package generator

import (
	"fmt"
	"strings"

	"github.com/php-any/generator/analyzer"
	"github.com/php-any/generator/core"
)

// CodeGeneratorImpl 代码生成器实现
type CodeGeneratorImpl struct {
	context   *core.GeneratorContext
	converter core.TypeConverter
	templates core.TemplateGenerator
}

// 创建新的代码生成器
func NewCodeGenerator(ctx *core.GeneratorContext, converter core.TypeConverter, templates core.TemplateGenerator) *CodeGeneratorImpl {
	return &CodeGeneratorImpl{
		context:   ctx,
		converter: converter,
		templates: templates,
	}
}

// Generate 生成代码
func (cg *CodeGeneratorImpl) Generate(ctx *core.GeneratorContext, info interface{}) (string, error) {
	if ctx == nil {
		return "", core.NewGeneratorError(core.ErrCodeCodeGeneration, "context is nil", nil)
	}

	if info == nil {
		return "", core.NewGeneratorError(core.ErrCodeCodeGeneration, "info is nil", nil)
	}

	// 根据信息类型选择生成策略
	switch v := info.(type) {
	case *core.FunctionInfo:
		return cg.GenerateFunction(ctx, v)
	case *core.TypeInfo:
		return cg.GenerateClass(ctx, v)
	case *core.MethodInfo:
		return cg.GenerateMethod(ctx, v)
	default:
		return "", core.NewGeneratorError(core.ErrCodeCodeGeneration,
			fmt.Sprintf("unsupported info type: %T", info), nil)
	}
}

// GenerateFunction 生成函数代码
func (cg *CodeGeneratorImpl) GenerateFunction(ctx *core.GeneratorContext, fn *core.FunctionInfo) (string, error) {
	if fn == nil {
		return "", core.NewGeneratorError(core.ErrCodeCodeGeneration, "function info is nil", nil)
	}

	// 准备模板数据
	templateData := &core.TemplateData{
		FunctionName: fn.Name,
		Context:      ctx,
		Config:       ctx.GetConfigManager().GetConfig(),
	}

	// 转换参数
	for i, param := range fn.Parameters {
		paramTypeStr, err := cg.converter.ConvertParameter(param)
		if err != nil {
			return "", core.NewGeneratorError(core.ErrCodeCodeGeneration,
				fmt.Sprintf("failed to convert parameter %s: %v", param.Name, err), nil)
		}

		templateData.Parameters = append(templateData.Parameters, core.ParameterTemplateData{
			Name:  param.Name,
			Type:  paramTypeStr,
			Index: i,
		})
	}

	// 转换返回值
	for i, ret := range fn.Returns {
		retTypeStr, err := cg.converter.ConvertReturn(ret)
		if err != nil {
			return "", core.NewGeneratorError(core.ErrCodeCodeGeneration,
				fmt.Sprintf("failed to convert return value %d: %v", i, err), nil)
		}

		templateData.Returns = append(templateData.Returns, core.ReturnTemplateData{
			Type:  retTypeStr,
			Index: i,
		})
	}

	// 生成代码
	return cg.templates.GenerateFunction(templateData)
}

// GenerateClass 生成类代码
func (cg *CodeGeneratorImpl) GenerateClass(ctx *core.GeneratorContext, class *core.TypeInfo) (string, error) {
	if class == nil {
		return "", core.NewGeneratorError(core.ErrCodeCodeGeneration, "class info is nil", nil)
	}

	// 获取包名（从类信息或配置中获取）
	pkgName := cg.getPackageName(class)
	if pkgName == "" {
		pkgName = "example" // 默认包名
	}

	// 准备模板数据
	templateData := &core.TemplateData{
		PackageName: pkgName,
		ClassName:   class.TypeName,
		Context:     ctx,
		Config:      ctx.GetConfigManager().GetConfig(),
	}

	// 转换字段
	for _, field := range class.Fields {
		fieldTypeStr, err := cg.converter.ConvertField(field)
		if err != nil {
			return "", core.NewGeneratorError(core.ErrCodeCodeGeneration,
				fmt.Sprintf("failed to convert field %s: %v", field.Name, err), nil)
		}

		templateData.Fields = append(templateData.Fields, core.FieldTemplateData{
			Name:       field.Name,
			Type:       fieldTypeStr,
			IsExported: field.IsExported,
		})
	}

	// 转换方法
	for _, method := range class.Methods {
		methodTemplateData := core.MethodTemplateData{
			Name:       method.Name,
			ClassName:  class.TypeName,
			IsVariadic: method.IsVariadic,
			IsExported: method.IsExported,
		}

		// 转换方法参数
		for i, param := range method.Parameters {
			paramTypeStr, err := cg.converter.ConvertParameter(param)
			if err != nil {
				return "", core.NewGeneratorError(core.ErrCodeCodeGeneration,
					fmt.Sprintf("failed to convert method parameter %s: %v", param.Name, err), nil)
			}

			methodTemplateData.Parameters = append(methodTemplateData.Parameters, core.ParameterTemplateData{
				Name:  param.Name,
				Type:  paramTypeStr,
				Index: i,
			})
		}

		// 转换方法返回值
		for i, ret := range method.Returns {
			retTypeStr, err := cg.converter.ConvertReturn(ret)
			if err != nil {
				return "", core.NewGeneratorError(core.ErrCodeCodeGeneration,
					fmt.Sprintf("failed to convert method return value %d: %v", i, err), nil)
			}

			methodTemplateData.Returns = append(methodTemplateData.Returns, core.ReturnTemplateData{
				Type:  retTypeStr,
				Index: i,
			})
		}

		templateData.Methods = append(templateData.Methods, methodTemplateData)
	}

	// 生成代码
	return cg.templates.GenerateClass(templateData)
}

// GenerateClassWithMethods 生成类代码和所有方法文件
func (cg *CodeGeneratorImpl) GenerateClassWithMethods(ctx *core.GeneratorContext, class *core.TypeInfo, emitter core.FileEmitter) error {
	return cg.GenerateClassWithMethodsRecursive(ctx, class, emitter, 100) // 默认递归深度为100
}

// GenerateClassWithMethodsRecursive 递归生成类代码和所有方法文件，包括依赖类型
func (cg *CodeGeneratorImpl) GenerateClassWithMethodsRecursive(ctx *core.GeneratorContext, class *core.TypeInfo, emitter core.FileEmitter, maxDepth int) error {
	if class == nil {
		return core.NewGeneratorError(core.ErrCodeCodeGeneration, "class info is nil", nil)
	}

	// 创建类型分析器来收集依赖
	typeAnalyzer := analyzer.NewTypeAnalyzer(ctx)

	// 收集所有依赖类型
	dependencies, err := typeAnalyzer.CollectDependencies(class, maxDepth)
	if err != nil {
		return core.NewGeneratorError(core.ErrCodeCodeGeneration,
			fmt.Sprintf("failed to collect dependencies: %v", err), nil)
	}

	fmt.Printf("收集到 %d 个依赖类型\n", len(dependencies))

	// 按包名分组依赖类型，去重
	packageGroups := make(map[string][]*core.TypeInfo)
	seenTypes := make(map[string]bool)

	for _, dep := range dependencies {
		pkgName := cg.getPackageName(dep)
		if pkgName == "" {
			continue
		}

		// 去重：使用类型名和包名的组合作为键
		typeKey := fmt.Sprintf("%s.%s", pkgName, dep.TypeName)
		if seenTypes[typeKey] {
			continue
		}

		seenTypes[typeKey] = true
		packageGroups[pkgName] = append(packageGroups[pkgName], dep)
	}

	// 为每个包生成代码
	for pkgName, types := range packageGroups {
		fmt.Printf("为包 %s 生成 %d 个类型的代码\n", pkgName, len(types))

		for _, depType := range types {
			if err := cg.generateSingleClassWithMethods(ctx, depType, emitter); err != nil {
				return core.NewGeneratorError(core.ErrCodeCodeGeneration,
					fmt.Sprintf("failed to generate code for type %s: %v", depType.TypeName, err), nil)
			}
		}
	}

	// 生成主类型的代码
	return cg.generateSingleClassWithMethods(ctx, class, emitter)
}

// generateSingleClassWithMethods 生成单个类的代码和方法文件
func (cg *CodeGeneratorImpl) generateSingleClassWithMethods(ctx *core.GeneratorContext, class *core.TypeInfo, emitter core.FileEmitter) error {
	if class == nil {
		return core.NewGeneratorError(core.ErrCodeCodeGeneration, "class info is nil", nil)
	}

	// 获取包名
	pkgName := cg.getPackageName(class)
	if pkgName == "" {
		pkgName = "example" // 默认包名
	}

	// 检查是否已经生成过（避免重复生成）
	if cg.isTypeGenerated(class) {
		return nil
	}

	// 生成类文件
	classCode, err := cg.GenerateClass(ctx, class)
	if err != nil {
		return err
	}

	// 输出类文件（使用类型名确保唯一性）
	classFileName := fmt.Sprintf("%s_%s_class.go", pkgName, class.TypeName)
	if err := emitter.EmitFile(pkgName, classFileName, classCode); err != nil {
		return core.NewGeneratorError(core.ErrCodeFileOperation,
			fmt.Sprintf("failed to emit class file: %v", err), nil)
	}

	// 生成方法文件
	fmt.Printf("为类型 %s 生成 %d 个方法文件\n", class.TypeName, len(class.Methods))
	for _, method := range class.Methods {
		// 准备方法模板数据（method.tmpl 通过 Methods[0] 读取方法名与参数/返回）
		methodTemplateData := &core.TemplateData{
			PackageName: pkgName,
			ClassName:   class.TypeName,
			Context:     ctx,
			Config:      ctx.GetConfigManager().GetConfig(),
			Methods: []core.MethodTemplateData{{
				Name:       method.Name,
				ClassName:  class.TypeName,
				Parameters: []core.ParameterTemplateData{},
				Returns:    []core.ReturnTemplateData{},
				IsVariadic: method.IsVariadic,
				IsExported: method.IsExported,
			}},
		}

		// 转换方法参数
		for i, param := range method.Parameters {
			paramTypeStr, err := cg.converter.ConvertParameter(param)
			if err != nil {
				return core.NewGeneratorError(core.ErrCodeCodeGeneration,
					fmt.Sprintf("failed to convert method parameter %s: %v", param.Name, err), nil)
			}

			methodTemplateData.Parameters = append(methodTemplateData.Parameters, core.ParameterTemplateData{ // 兼容保留
				Name:  param.Name,
				Type:  paramTypeStr,
				Index: i,
			})
			mt0 := methodTemplateData.Methods[0]
			mt0.Parameters = append(mt0.Parameters, core.ParameterTemplateData{
				Name:  param.Name,
				Type:  paramTypeStr,
				Index: i,
			})
			methodTemplateData.Methods[0] = mt0
		}

		// 转换方法返回值
		for i, ret := range method.Returns {
			retTypeStr, err := cg.converter.ConvertReturn(ret)
			if err != nil {
				return core.NewGeneratorError(core.ErrCodeCodeGeneration,
					fmt.Sprintf("failed to convert method return value %d: %v", i, err), nil)
			}

			methodTemplateData.Returns = append(methodTemplateData.Returns, core.ReturnTemplateData{ // 兼容保留
				Type:  retTypeStr,
				Index: i,
			})
			mt0 := methodTemplateData.Methods[0]
			mt0.Returns = append(mt0.Returns, core.ReturnTemplateData{
				Type:  retTypeStr,
				Index: i,
			})
			methodTemplateData.Methods[0] = mt0
		}

		// 生成方法代码
		methodCode, err := cg.templates.GenerateMethod(methodTemplateData)
		if err != nil {
			return core.NewGeneratorError(core.ErrCodeCodeGeneration,
				fmt.Sprintf("failed to generate method code: %v", err), nil)
		}

		// 输出方法文件
		methodFileName := fmt.Sprintf("%s_%s_method.go", pkgName, method.Name)
		if err := emitter.EmitFile(pkgName, methodFileName, methodCode); err != nil {
			return core.NewGeneratorError(core.ErrCodeFileOperation,
				fmt.Sprintf("failed to emit method file: %v", err), nil)
		}
	}

	// 标记为已生成
	cg.markTypeGenerated(class)

	return nil
}

// isTypeGenerated 检查类型是否已生成
func (cg *CodeGeneratorImpl) isTypeGenerated(typeInfo *core.TypeInfo) bool {
	return cg.context.IsTypeGenerated(typeInfo.CacheKey)
}

// markTypeGenerated 标记类型为已生成
func (cg *CodeGeneratorImpl) markTypeGenerated(typeInfo *core.TypeInfo) {
	cg.context.MarkTypeGenerated(typeInfo.CacheKey)
}

// getPackageName 获取包名
func (cg *CodeGeneratorImpl) getPackageName(class *core.TypeInfo) string {
	if class == nil {
		return ""
	}

	// 从类信息中获取包名
	if class.PackagePath != "" {
		parts := strings.Split(class.PackagePath, "/")
		return parts[len(parts)-1]
	}

	// 从配置中获取包名映射
	if cg.context.GetConfigManager() != nil {
		config := cg.context.GetConfigManager().GetConfig()
		if config != nil {
			// 这里可以根据配置规则确定包名
			return "example"
		}
	}

	return ""
}

// GenerateMethod 生成方法代码
func (cg *CodeGeneratorImpl) GenerateMethod(ctx *core.GeneratorContext, method *core.MethodInfo) (string, error) {
	if method == nil {
		return "", core.NewGeneratorError(core.ErrCodeCodeGeneration, "method info is nil", nil)
	}

	// 准备模板数据
	templateData := &core.TemplateData{
		FunctionName: method.Name,
		Context:      ctx,
		Config:       ctx.GetConfigManager().GetConfig(),
	}

	// 转换参数
	for i, param := range method.Parameters {
		paramTypeStr, err := cg.converter.ConvertParameter(param)
		if err != nil {
			return "", core.NewGeneratorError(core.ErrCodeCodeGeneration,
				fmt.Sprintf("failed to convert parameter %s: %v", param.Name, err), nil)
		}

		templateData.Parameters = append(templateData.Parameters, core.ParameterTemplateData{
			Name:  param.Name,
			Type:  paramTypeStr,
			Index: i,
		})
	}

	// 转换返回值
	for i, ret := range method.Returns {
		retTypeStr, err := cg.converter.ConvertReturn(ret)
		if err != nil {
			return "", core.NewGeneratorError(core.ErrCodeCodeGeneration,
				fmt.Sprintf("failed to convert return value %d: %v", i, err), nil)
		}

		templateData.Returns = append(templateData.Returns, core.ReturnTemplateData{
			Type:  retTypeStr,
			Index: i,
		})
	}

	// 生成代码
	return cg.templates.GenerateMethod(templateData)
}

// ApplyPackageConfig 应用包配置
func (cg *CodeGeneratorImpl) ApplyPackageConfig(ctx *core.GeneratorContext, pkgPath string) error {
	if ctx == nil || pkgPath == "" {
		return nil
	}

	// 这里可以应用包级别的配置
	// 例如设置包前缀、检查黑名单等
	// 具体实现根据需求而定

	return nil
}
