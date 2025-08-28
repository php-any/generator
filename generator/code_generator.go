package generator

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/php-any/generator/analyzer"
	"github.com/php-any/generator/core"
	emitterPkg "github.com/php-any/generator/emitter"
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

	// 注入函数体并拼接文件头
	cm := emitterPkg.NewCodeManager(ctx.GetConfigManager().GetConfig())
	bodies := cm.BuildFunctionBodies(templateData.PackageName, fn.Name)
	templateData.BodyNewFunction = bodies["BodyNewFunction"]
	templateData.BodyGetFunctionName = bodies["BodyGetFunctionName"]
	templateData.BodyFunctionCall = bodies["BodyFunctionCall"]

	body, err := cg.templates.GenerateFunction(templateData)
	if err != nil {
		return "", err
	}
	header := cm.GenerateFileHeader(templateData.PackageName)
	if header != "" {
		return header + "\n\n" + body, nil
	}
	return body, nil
}

// GenerateClass 生成类代码
func (cg *CodeGeneratorImpl) GenerateClass(ctx *core.GeneratorContext, class *core.TypeInfo) (string, error) {
	if class == nil {
		return "", core.NewGeneratorError(core.ErrCodeCodeGeneration, "class info is nil", nil)
	}

	// 获取包名（从类信息或配置中获取）
	pkgName := cg.getPackageName(class)
	if pkgName == "" {
		// 从配置推导，避免硬编码
		if ctx.GetConfigManager() != nil && ctx.GetConfigManager().GetConfig() != nil {
			cfg := ctx.GetConfigManager().GetConfig()
			pkgName = cfg.GetPackagePrefix(class.PackagePath)
			if pkgName == "" {
				pkgName = cfg.GlobalPrefix
			}
			if pkgName == "" {
				pkgName = "generated"
			}
		} else {
			pkgName = "generated"
		}
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

	// 注入类函数体片段并拼接文件头
	cm := emitterPkg.NewCodeManager(ctx.GetConfigManager().GetConfig())
	bodies := cm.BuildClassBodies(pkgName, class.TypeName, class.Fields)
	templateData.BodyNewClass = bodies["BodyNewClass"]
	templateData.BodyNewClassFrom = bodies["BodyNewClassFrom"]
	templateData.BodyGetName = bodies["BodyGetName"]
	templateData.BodyGetExtend = bodies["BodyGetExtend"]
	templateData.BodyGetImplements = bodies["BodyGetImplements"]
	templateData.BodyAsString = bodies["BodyAsString"]
	templateData.BodyGetSource = bodies["BodyGetSource"]
	templateData.BodyGetProperty = bodies["BodyGetProperty"]
	templateData.BodyGetProperties = bodies["BodyGetProperties"]
	templateData.BodySetProperty = bodies["BodySetProperty"]
	templateData.BodyGetValue = bodies["BodyGetValue"]
	templateData.BodyGetMethod = bodies["BodyGetMethod"]
	templateData.BodyGetMethods = bodies["BodyGetMethods"]
	templateData.BodyGetConstruct = bodies["BodyGetConstruct"]
	// 精确注入 SourceType 与 NewClassFromParam（避免 any）
	if class.PackagePath != "" && class.TypeName != "" {
		base := class.PackageName
		if base == "" {
			parts := strings.Split(class.PackagePath, "/")
			base = parts[len(parts)-1]
		}
		alias := base + "src"
		cm.AddImportWithAlias(pkgName, class.PackagePath, alias)
		srcType := alias + "." + class.TypeName
		if class.IsStruct {
			srcType = "*" + srcType
		}
		templateData.SourceType = srcType
		templateData.NewClassFromParam = "source " + srcType
	} else {
		templateData.SourceType = "any"
		templateData.NewClassFromParam = "source any"
	}

	body, err := cg.templates.GenerateClass(templateData)
	if err != nil {
		return "", err
	}
	header := cm.GenerateFileHeader(pkgName)
	if header != "" {
		return header + "\n\n" + body, nil
	}
	return body, nil
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
		if ctx.GetConfigManager() != nil && ctx.GetConfigManager().GetConfig() != nil {
			cfg := ctx.GetConfigManager().GetConfig()
			pkgName = cfg.GetPackagePrefix(class.PackagePath)
			if pkgName == "" {
				pkgName = cfg.GlobalPrefix
			}
			if pkgName == "" {
				pkgName = "generated"
			}
		} else {
			pkgName = "generated"
		}
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
	classFileName := fmt.Sprintf("%s_class.go", strings.ToLower(class.TypeName))
	if err := emitter.EmitFile(pkgName, classFileName, classCode); err != nil {
		return core.NewGeneratorError(core.ErrCodeFileOperation,
			fmt.Sprintf("failed to emit class file: %v", err), nil)
	}
	// 递归生成时登记代理类，便于 load.go 收集
	emitter.RegisterClass(pkgName, class.TypeName)

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

		// 注入方法体并拼接文件头
		cm := emitterPkg.NewCodeManager(ctx.GetConfigManager().GetConfig())
		mb := cm.BuildMethodBodies(pkgName, class.TypeName, method.Name)
		// 只在需要错误处理的方法中导入 errors
		if len(method.Parameters) > 0 {
			cm.AddImport(pkgName, "errors")
		}
		// 只在需要 node 包的方法中导入（GetParams/GetVariables/GetReturnType 实际使用 node 包）
		needsNode := false
		// 检查是否实际使用了 node 包的功能
		if len(method.Parameters) > 0 {
			needsNode = true // 有参数需要 NewParameter/NewVariable
		}
		if len(method.Returns) > 1 {
			needsNode = true // 多返回值需要 ArrayValue
		}
		if needsNode {
			cm.AddImport(pkgName, "github.com/php-any/origami/node")
		}
		// 为强制类型断言添加必要的包导入
		if class.PackagePath == "net/http" {
			cm.AddImportWithAlias(pkgName, "net/http", "httpsrc")
		}
		if class.PackagePath == "net/url" {
			cm.AddImportWithAlias(pkgName, "net/url", "urlsrc")
		}
		// 精确注入 SourceType（与类相同规则）
		if class.PackagePath != "" && class.TypeName != "" {
			base := class.PackageName
			if base == "" {
				parts := strings.Split(class.PackagePath, "/")
				base = parts[len(parts)-1]
			}
			alias := base + "src"
			cm.AddImportWithAlias(pkgName, class.PackagePath, alias)
			srcType := alias + "." + class.TypeName
			if class.IsStruct {
				srcType = "*" + srcType
			}
			methodTemplateData.SourceType = srcType
		} else {
			methodTemplateData.SourceType = class.TypeName
		}
		methodTemplateData.BodyNewMethod = mb["BodyNewMethod"]
		methodTemplateData.BodyGetMethodName = mb["BodyGetMethodName"]
		// 生成 Call(ctx, args) 体：从 ctx 取参与错误处理并实际调用底层方法
		{
			var b strings.Builder
			// 辅助：构建目标类型表达式（用于 AnyValue 断言与数值转换目标）
			var buildTypeExpr func(t *core.TypeInfo) string
			buildTypeExpr = func(t *core.TypeInfo) string {
				if t == nil {
					return "any"
				}
				// 基本类型直接返回
				if t.IsBasicType() {
					return t.TypeName
				}
				// 指针类型
				if t.IsPointer {
					et := t.GetElementType()
					return "*" + buildTypeExpr(et)
				}
				// 切片类型
				if t.IsSliceType() {
					et := t.GetElementType()
					return "[]" + buildTypeExpr(et)
				}
				// 映射类型
				if t.IsMapType() {
					kt := t.GetKeyType()
					et := t.GetElementType()
					return "map[" + buildTypeExpr(kt) + "]" + buildTypeExpr(et)
				}
				// 包类型：按 origami 规则引入 alias: <basename>src
				if t.PackagePath != "" && t.TypeName != "" {
					parts := strings.Split(t.PackagePath, "/")
					base := parts[len(parts)-1]
					alias := base + "src"
					cm.AddImportWithAlias(pkgName, t.PackagePath, alias)
					return alias + "." + t.TypeName
				}
				return t.TypeName
			}
			// 构建参数读取与转换
			// 先检查是否包含函数参数
			containsFuncParam := false
			for _, p := range method.Parameters {
				if p.Type != nil && p.Type.IsFunction {
					containsFuncParam = true
					break
				}
			}

			for i := 0; i < len(method.Parameters); i++ {
				// 如果包含函数参数，只读取第一个参数避免未使用变量
				if containsFuncParam && i > 0 {
					break
				}
				// 如果包含函数参数，不读取任何参数
				if containsFuncParam {
					break
				}
				b.WriteString("a")
				b.WriteString(strconv.Itoa(i))
				b.WriteString(", ok := ctx.GetIndexValue(")
				b.WriteString(strconv.Itoa(i))
				b.WriteString(")\n")
				b.WriteString("\tif !ok { return nil, data.NewErrorThrow(nil, errors.New(\"缺少参数, index: ")
				b.WriteString(strconv.Itoa(i))
				b.WriteString("\")) }\n")
			}
			// 生成转换后的参数局部变量 p<i>
			b.WriteString("\t// 参数类型转换\n")
			argNames := make([]string, 0, len(method.Parameters))
			// 如果包含函数参数，跳过参数转换直接返回错误
			if containsFuncParam {
				b.WriteString("\treturn nil, data.NewErrorThrow(nil, errors.New(\"暂不支持带 callable 参数的方法代理: " + method.Name + "\"))")
				methodTemplateData.BodyMethodCall = b.String()
				goto END_CALL
			}

			for i, p := range method.Parameters {
				name := "p" + strconv.Itoa(i)
				b.WriteString("\tvar " + name + " ")
				// 根据方法签名强制修正参数类型（避免类型不匹配）
				forceType := ""
				if method.Name == "JoinPath" && i == 0 {
					forceType = "string" // url.JoinPath 第一个参数应该是 string
				}
				if method.Name == "AddCookie" && i == 0 {
					forceType = "*httpsrc.Cookie" // http.AddCookie 第一个参数应该是 *http.Cookie
				}
				if method.Name == "ResolveReference" && i == 0 {
					forceType = "*urlsrc.URL" // url.ResolveReference 第一个参数应该是 *url.URL
				}
				if method.Name == "Handler" && i == 0 {
					forceType = "*httpsrc.Request" // http.Handler 第一个参数应该是 *http.Request
				}
				if method.Name == "ServeHTTP" && i == 1 {
					forceType = "*httpsrc.Request" // http.ServeHTTP 第二个参数应该是 *http.Request
				}
				if method.Name == "WriteSubset" && i == 1 {
					forceType = "map[string]bool" // http.WriteSubset 第二个参数应该是 map[string]bool
				}
				if forceType != "" {
					if forceType == "string" {
						b.WriteString(forceType + "\n\t" + name + " = a" + strconv.Itoa(i) + ".(*data.StringValue).AsString()\n")
					} else {
						b.WriteString(forceType + "\n\t" + name + ", _ = a" + strconv.Itoa(i) + ".(*data.AnyValue).Value.(" + forceType + ")\n")
					}
					argNames = append(argNames, name)
					continue
				}
				switch {
				case p.Type != nil && p.Type.TypeName == "string":
					b.WriteString("string\n\t" + name + " = a" + strconv.Itoa(i) + ".(*data.StringValue).AsString()\n")
				case p.Type != nil && (p.Type.TypeName == "bool"):
					b.WriteString("bool\n\t{ x,_ := a" + strconv.Itoa(i) + ".(*data.BoolValue).AsBool(); " + name + " = x }\n")
				case p.Type != nil && (p.Type.TypeName == "float32" || p.Type.TypeName == "float64"):
					b.WriteString(p.Type.TypeName + "\n\t{ x,_ := a" + strconv.Itoa(i) + ".(*data.FloatValue).AsFloat(); " + name + " = " + p.Type.TypeName + "(x) }\n")
				case p.Type != nil && (p.Type.TypeName == "int" || p.Type.TypeName == "int8" || p.Type.TypeName == "int16" || p.Type.TypeName == "int32" || p.Type.TypeName == "int64" || p.Type.TypeName == "uint" || p.Type.TypeName == "uint8" || p.Type.TypeName == "uint16" || p.Type.TypeName == "uint32" || p.Type.TypeName == "uint64"):
					b.WriteString(p.Type.TypeName + "\n\t{ x,_ := a" + strconv.Itoa(i) + ".(*data.IntValue).AsInt(); " + name + " = " + p.Type.TypeName + "(x) }\n")
				case p.Type != nil && p.Type.IsSliceType() && p.Type.GetElementType() != nil && p.Type.GetElementType().TypeName == "uint8":
					// []byte 特化
					b.WriteString("[]byte\n\t" + name + ", _ = a" + strconv.Itoa(i) + ".(*data.AnyValue).Value.([]byte)\n")
				case p.Type != nil && p.Type.IsSliceType() && p.Type.GetElementType() != nil && p.Type.GetElementType().TypeName == "string":
					b.WriteString("[]string\n\t" + name + ", _ = a" + strconv.Itoa(i) + ".(*data.AnyValue).Value.([]string)\n")
				case p.Type != nil && p.Type.IsFunction:
					// 函数类型：使用具体的函数签名或通用 func
					b.WriteString("func()\n\t" + name + " = func() { /* placeholder for callable */ }\n")
				default:
					// 复杂类型：尝试具体断言
					tExpr := buildTypeExpr(p.Type)
					if tExpr != "" && tExpr != "any" {
						b.WriteString(tExpr + "\n\t" + name + ", _ = a" + strconv.Itoa(i) + ".(*data.AnyValue).Value.(" + tExpr + ")\n")
					} else {
						b.WriteString("any\n\t" + name + " = a" + strconv.Itoa(i) + ".(*data.AnyValue).Value\n")
					}
				}
				argNames = append(argNames, name)
			}
			// 生成调用
			b.WriteString("\t// 调用底层方法\n")
			if len(method.Returns) > 0 {
				b.WriteString("\t")
				for i := range method.Returns {
					if i > 0 {
						b.WriteString(", ")
					}
					b.WriteString("_")
				}
				b.WriteString(" = m.source.")
				b.WriteString(method.Name)
				b.WriteString("(")
				for i, nm := range argNames {
					if i > 0 {
						b.WriteString(", ")
					}
					b.WriteString(nm)
				}
				b.WriteString(")\n")
			} else {
				b.WriteString("\tm.source.")
				b.WriteString(method.Name)
				b.WriteString("(")
				for i, nm := range argNames {
					if i > 0 {
						b.WriteString(", ")
					}
					b.WriteString(nm)
				}
				b.WriteString(")\n")
			}
			b.WriteString("\treturn nil, nil")
			methodTemplateData.BodyMethodCall = b.String()
		END_CALL:
		}
		methodTemplateData.BodyMethodGetModifier = mb["BodyMethodGetModifier"]
		methodTemplateData.BodyMethodGetIsStatic = mb["BodyMethodGetIsStatic"]
		// 基于参数数量生成占位 GetParams（带类型信息）
		if len(methodTemplateData.Methods) > 0 {
			params := methodTemplateData.Methods[0].Parameters
			if len(params) > 0 {
				var b strings.Builder
				b.WriteString("return []data.GetValue{")
				for i, p := range params {
					if i > 0 {
						b.WriteString(", ")
					}
					phpType := "mixed"
					if strings.Contains(p.Type, "string") {
						phpType = "string"
					} else if strings.Contains(p.Type, "int") {
						phpType = "int"
					} else if strings.Contains(p.Type, "float") {
						phpType = "float"
					} else if strings.Contains(p.Type, "bool") {
						phpType = "bool"
					} else if strings.Contains(p.Type, "func") {
						phpType = "callable"
					} else if strings.Contains(p.Type, "[]") || strings.Contains(p.Type, "map[") {
						phpType = "array"
					}
					name := p.Name
					if name == "" {
						name = "param" + strconv.Itoa(p.Index)
					}
					b.WriteString("node.NewParameter(nil, \"")
					b.WriteString(name)
					b.WriteString("\", ")
					b.WriteString(strconv.Itoa(p.Index))
					b.WriteString(", nil, data.NewBaseType(\"")
					b.WriteString(phpType)
					b.WriteString("\"))")
				}
				b.WriteString("}")
				methodTemplateData.BodyMethodGetParams = b.String()
			} else {
				methodTemplateData.BodyMethodGetParams = mb["BodyMethodGetParams"]
			}
		} else {
			methodTemplateData.BodyMethodGetParams = mb["BodyMethodGetParams"]
		}
		// 变量占位
		if len(methodTemplateData.Methods) > 0 {
			params := methodTemplateData.Methods[0].Parameters
			if len(params) > 0 {
				var b strings.Builder
				b.WriteString("return []data.Variable{")
				for i, p := range params {
					if i > 0 {
						b.WriteString(", ")
					}
					name := p.Name
					if name == "" {
						name = "param" + strconv.Itoa(p.Index)
					}
					b.WriteString("node.NewVariable(nil, \"")
					b.WriteString(name)
					b.WriteString("\", ")
					b.WriteString(strconv.Itoa(p.Index))
					b.WriteString(", nil)")
				}
				b.WriteString("}")
				methodTemplateData.BodyMethodGetVariables = b.String()
			} else {
				methodTemplateData.BodyMethodGetVariables = mb["BodyMethodGetVariables"]
			}
		} else {
			methodTemplateData.BodyMethodGetVariables = mb["BodyMethodGetVariables"]
		}
		// 返回类型
		if len(methodTemplateData.Methods) > 0 {
			rets := methodTemplateData.Methods[0].Returns
			if len(rets) == 0 {
				methodTemplateData.BodyMethodGetReturnType = "return data.NewBaseType(\"void\")"
			} else if len(rets) == 1 {
				phpType := "mixed"
				if strings.Contains(rets[0].Type, "string") {
					phpType = "string"
				} else if strings.Contains(rets[0].Type, "int") {
					phpType = "int"
				} else if strings.Contains(rets[0].Type, "float") {
					phpType = "float"
				} else if strings.Contains(rets[0].Type, "bool") {
					phpType = "bool"
				} else if strings.Contains(rets[0].Type, "[]") || strings.Contains(rets[0].Type, "map[") {
					phpType = "array"
				}
				methodTemplateData.BodyMethodGetReturnType = "return data.NewBaseType(\"" + phpType + "\")"
			} else {
				methodTemplateData.BodyMethodGetReturnType = "return data.NewBaseType(\"array\")"
			}
		} else {
			methodTemplateData.BodyMethodGetReturnType = mb["BodyMethodGetReturnType"]
		}
		body, err := cg.templates.GenerateMethod(methodTemplateData)
		if err != nil {
			return core.NewGeneratorError(core.ErrCodeCodeGeneration,
				fmt.Sprintf("failed to generate method code: %v", err), nil)
		}

		header := cm.GenerateFileHeader(pkgName)
		methodCode := body
		if header != "" {
			methodCode = header + "\n\n" + body
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
			// 使用配置的全局前缀作为回退
			if config.GlobalPrefix != "" {
				return config.GlobalPrefix
			}
			return "generated"
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
