package emitter

import (
	"fmt"
	"reflect"
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

// BuildClassBodies 生成类模板的函数体片段
func (cm *CodeManager) BuildClassBodies(pkgName, className string, fields []core.FieldInfo) map[string]string {
	// 必要导入
	cm.AddImport(pkgName, cm.mapPackagePath("data"))
	cm.AddImport(pkgName, cm.mapPackagePath("node"))

	bodies := map[string]string{}
	bodies["BodyNewClass"] = fmt.Sprintf("return &%sClass{}", className)
	bodies["BodyNewClassFrom"] = fmt.Sprintf("return &%sClass{source: source}", className)
	bodies["BodyGetName"] = fmt.Sprintf("return \"%s\\\\%s\"", pkgName, className)
	bodies["BodyGetExtend"] = "return nil"
	bodies["BodyGetImplements"] = "return nil"
	bodies["BodyAsString"] = fmt.Sprintf("return \"%s{}\"", className)
	bodies["BodyGetSource"] = "return s.source"
	// 默认字段逻辑（导出字段）
	if len(fields) > 0 {
		var sbGet strings.Builder
		sbGet.WriteString("if s.source == nil { return nil, false }\n")
		sbGet.WriteString("src := s.source\n")
		sbGet.WriteString("switch name {\n")
		var sbProps strings.Builder
		sbProps.WriteString("props := make(map[string]data.Property)\n")
		var sbSet strings.Builder
		sbSet.WriteString("if s.source == nil { return }\n")
		sbSet.WriteString("src := s.source\n")
		sbSet.WriteString("switch name {\n")
		for _, f := range fields {
			if !f.IsExported {
				continue
			}
			fname := f.Name
			// GetProperty cases
			sbGet.WriteString("case \"")
			sbGet.WriteString(fname)
			sbGet.WriteString("\":\n")
			kind := "other"
			if f.Type != nil && f.Type.Type != nil {
				switch f.Type.Type.Kind() {
				case reflect.String:
					kind = "string"
				case reflect.Bool:
					kind = "bool"
				case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
					reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
					kind = "int"
				case reflect.Float32, reflect.Float64:
					kind = "float"
				default:
					kind = "other"
				}
			}
			if kind == "string" {
				sbGet.WriteString("\treturn node.NewProperty(nil, \"")
				sbGet.WriteString(fname)
				sbGet.WriteString("\", \"public\", true, data.NewStringValue(src.")
				sbGet.WriteString(fname)
				sbGet.WriteString(")), true\n")
			} else if kind == "bool" {
				sbGet.WriteString("\treturn node.NewProperty(nil, \"")
				sbGet.WriteString(fname)
				sbGet.WriteString("\", \"public\", true, data.NewBoolValue(src.")
				sbGet.WriteString(fname)
				sbGet.WriteString(")), true\n")
			} else if kind == "int" {
				sbGet.WriteString("\treturn node.NewProperty(nil, \"")
				sbGet.WriteString(fname)
				sbGet.WriteString("\", \"public\", true, data.NewIntValue(int(src.")
				sbGet.WriteString(fname)
				sbGet.WriteString("))), true\n")
			} else {
				sbGet.WriteString("\treturn node.NewProperty(nil, \"")
				sbGet.WriteString(fname)
				sbGet.WriteString("\", \"public\", true, data.NewAnyValue(src.")
				sbGet.WriteString(fname)
				sbGet.WriteString(")), true\n")
			}

			// GetProperties entries
			sbProps.WriteString("props[\"")
			sbProps.WriteString(fname)
			sbProps.WriteString("\"] = node.NewProperty(nil, \"")
			sbProps.WriteString(fname)
			sbProps.WriteString("\", \"public\", true, data.NewAnyValue(nil))\n")

			// SetProperty cases
			sbSet.WriteString("case \"")
			sbSet.WriteString(fname)
			sbSet.WriteString("\":\n")
			if f.Type != nil && f.Type.Type != nil && f.Type.Type.Kind() == reflect.String {
				sbSet.WriteString("\tif sv, ok := value.(*data.StringValue); ok { src.")
				sbSet.WriteString(fname)
				sbSet.WriteString(" = string(sv.AsString()) }\n")
			} else if kind == "bool" {
				sbSet.WriteString("\tif bv, ok := value.(*data.BoolValue); ok { if x, err := bv.AsBool(); err == nil { src.")
				sbSet.WriteString(fname)
				sbSet.WriteString(" = x } }\n")
			} else if kind == "int" {
				// 具名整型优先按具名类型转换
				if f.Type != nil && f.Type.Type != nil && f.Type.Type.Name() != "" && f.Type.Type.PkgPath() != "" {
					targetType := fmt.Sprintf("%ssrc.%s", f.Type.PackageName, f.Type.TypeName)
					sbSet.WriteString("\tif iv, ok := value.(*data.IntValue); ok { if x, err := iv.AsInt(); err == nil { src.")
					sbSet.WriteString(fname)
					sbSet.WriteString(" = ")
					sbSet.WriteString(targetType)
					sbSet.WriteString("(x) } }\n")
				} else {
					// 基本整型按具体种类转换
					switch f.Type.Type.Kind() {
					case reflect.Int:
						sbSet.WriteString("\tif iv, ok := value.(*data.IntValue); ok { if x, err := iv.AsInt(); err == nil { src.")
						sbSet.WriteString(fname)
						sbSet.WriteString(" = int(x) } }\n")
					case reflect.Int8:
						sbSet.WriteString("\tif iv, ok := value.(*data.IntValue); ok { if x, err := iv.AsInt(); err == nil { src.")
						sbSet.WriteString(fname)
						sbSet.WriteString(" = int8(x) } }\n")
					case reflect.Int16:
						sbSet.WriteString("\tif iv, ok := value.(*data.IntValue); ok { if x, err := iv.AsInt(); err == nil { src.")
						sbSet.WriteString(fname)
						sbSet.WriteString(" = int16(x) } }\n")
					case reflect.Int32:
						sbSet.WriteString("\tif iv, ok := value.(*data.IntValue); ok { if x, err := iv.AsInt(); err == nil { src.")
						sbSet.WriteString(fname)
						sbSet.WriteString(" = int32(x) } }\n")
					case reflect.Int64:
						sbSet.WriteString("\tif iv, ok := value.(*data.IntValue); ok { if x, err := iv.AsInt(); err == nil { src.")
						sbSet.WriteString(fname)
						sbSet.WriteString(" = int64(x) } }\n")
					case reflect.Uint:
						sbSet.WriteString("\tif iv, ok := value.(*data.IntValue); ok { if x, err := iv.AsInt(); err == nil { src.")
						sbSet.WriteString(fname)
						sbSet.WriteString(" = uint(x) } }\n")
					case reflect.Uint8:
						sbSet.WriteString("\tif iv, ok := value.(*data.IntValue); ok { if x, err := iv.AsInt(); err == nil { src.")
						sbSet.WriteString(fname)
						sbSet.WriteString(" = uint8(x) } }\n")
					case reflect.Uint16:
						sbSet.WriteString("\tif iv, ok := value.(*data.IntValue); ok { if x, err := iv.AsInt(); err == nil { src.")
						sbSet.WriteString(fname)
						sbSet.WriteString(" = uint16(x) } }\n")
					case reflect.Uint32:
						sbSet.WriteString("\tif iv, ok := value.(*data.IntValue); ok { if x, err := iv.AsInt(); err == nil { src.")
						sbSet.WriteString(fname)
						sbSet.WriteString(" = uint32(x) } }\n")
					case reflect.Uint64:
						sbSet.WriteString("\tif iv, ok := value.(*data.IntValue); ok { if x, err := iv.AsInt(); err == nil { src.")
						sbSet.WriteString(fname)
						sbSet.WriteString(" = uint64(x) } }\n")
					default:
						// 兜底
						sbSet.WriteString("\tif iv, ok := value.(*data.IntValue); ok { if x, err := iv.AsInt(); err == nil { src.")
						sbSet.WriteString(fname)
						sbSet.WriteString(" = int(x) } }\n")
					}
				}
			} else {
				if f.Type != nil && f.Type.Type != nil {
					typeExpr := cm.buildGoTypeExpr(pkgName, f.Type)
					sbSet.WriteString("\tif av, ok := value.(*data.AnyValue); ok { if v, ok2 := av.Value.(")
					sbSet.WriteString(typeExpr)
					sbSet.WriteString("); ok2 { src.")
					sbSet.WriteString(fname)
					sbSet.WriteString(" = v } }\n")
				} else {
					sbSet.WriteString("\tif av, ok := value.(*data.AnyValue); ok { src.")
					sbSet.WriteString(fname)
					sbSet.WriteString(" = av.Value }\n")
				}
			}
		}
		sbGet.WriteString("}\nreturn nil, false")
		bodies["BodyGetProperty"] = sbGet.String()
		sbProps.WriteString("return props")
		bodies["BodyGetProperties"] = sbProps.String()
		sbSet.WriteString("}")
		bodies["BodySetProperty"] = sbSet.String()
	} else {
		bodies["BodyGetProperty"] = "return nil, false"
		bodies["BodyGetProperties"] = "return make(map[string]data.Property)"
		bodies["BodySetProperty"] = "return"
	}
	bodies["BodyGetValue"] = "return data.NewClassValue(s, ctx.CreateBaseContext()), nil"
	bodies["BodyGetMethod"] = "return nil, false"
	bodies["BodyGetMethods"] = "return nil"
	bodies["BodyGetConstruct"] = "return nil"

	// 不再保留硬编码的类型特例，统一由字段与配置驱动
	return bodies
}

// buildGoTypeExpr 根据 TypeInfo 构建用于断言的 Go 类型表达式，并自动添加必要 import
func (cm *CodeManager) buildGoTypeExpr(pkgName string, t *core.TypeInfo) string {
	if t == nil || t.Type == nil {
		return "any"
	}
	// 基础类型与本地包直接返回反射字符串
	// 对于外部包，使用 <pkgName>src.TypeName 形式，并根据配置映射路径
	// 指针/切片/映射/数组等复合类型递归处理
	switch t.Type.Kind() {
	case reflect.Ptr:
		elem := cm.buildGoTypeExpr(pkgName, t.GetElementType())
		return "*" + elem
	case reflect.Slice:
		elem := cm.buildGoTypeExpr(pkgName, t.GetElementType())
		return "[]" + elem
	case reflect.Array:
		// 无法得知定长，回退为切片表达式以用于断言语义
		elem := cm.buildGoTypeExpr(pkgName, t.GetElementType())
		return "[]" + elem
	case reflect.Map:
		keyExpr := cm.buildGoTypeExpr(pkgName, t.GetKeyType())
		valExpr := cm.buildGoTypeExpr(pkgName, t.GetElementType())
		return "map[" + keyExpr + "]" + valExpr
	default:
		// 具名类型
		if t.Type.PkgPath() != "" {
			base := core.NewTypeInfo(t.Type).PackageName
			// 基于配置映射：优先 base（无别名），其次 base+"src"（有别名）
			if cm.config != nil && cm.config.PackageMappings != nil {
				if _, ok := cm.config.PackageMappings[base]; ok {
					// 无别名，直接按 base 使用配置路径
					cm.AddImport(pkgName, cm.mapPackagePath(base))
					return base + "." + t.TypeName
				}
				aliasKey := base + "src"
				if _, ok := cm.config.PackageMappings[aliasKey]; ok {
					cm.AddImportWithAlias(pkgName, cm.mapPackagePath(aliasKey), aliasKey)
					return aliasKey + "." + t.TypeName
				}
			}
			// 未配置映射：根据是否标准库决定
			if cm.isStandardLibrary(base) {
				cm.AddImport(pkgName, cm.mapPackagePath(base))
				return base + "." + t.TypeName
			}
			alias := base + "src"
			cm.AddImportWithAlias(pkgName, cm.mapPackagePath(alias), alias)
			return alias + "." + t.TypeName
		}
		// 内建或无包名类型
		return t.Type.String()
	}
}

// BuildClassSignature 填充类模板签名相关占位
func (cm *CodeManager) BuildClassSignature(data *core.TemplateData) {
	// 默认 any（如需具体类型，可通过分析器提供并在此处注入，避免硬编码）
	data.NewClassFromParam = "source any"
	data.SourceType = "any"
}

// BuildFunctionBodies 生成函数模板的函数体片段
func (cm *CodeManager) BuildFunctionBodies(pkgName, functionName string) map[string]string {
	cm.AddImport(pkgName, cm.mapPackagePath("data"))
	cm.AddImport(pkgName, cm.mapPackagePath("node"))
	bodies := map[string]string{}
	bodies["BodyNewFunction"] = fmt.Sprintf("return &%sFunction{}", functionName)
	bodies["BodyGetFunctionName"] = fmt.Sprintf("return \"%s\"", functionName)
	bodies["BodyFunctionCall"] = fmt.Sprintf("return data.NewStringValue(\"%s 调用完成\"), nil", functionName)
	bodies["BodyFuncGetModifier"] = "return data.ModifierPublic"
	bodies["BodyFuncGetIsStatic"] = "return true"
	bodies["BodyFuncGetParams"] = "return []data.GetValue{}"
	bodies["BodyFuncGetVariables"] = "return []data.Variable{}"
	bodies["BodyFuncGetReturnType"] = "return data.NewBaseType(\"mixed\")"
	return bodies
}

// BuildMethodBodies 生成方法模板的函数体片段
func (cm *CodeManager) BuildMethodBodies(pkgName, className, methodName string) map[string]string {
	cm.AddImport(pkgName, cm.mapPackagePath("data"))
	bodies := map[string]string{}
	bodies["BodyNewMethod"] = "return &" + methodName + "Method{ source: source }"
	bodies["BodyGetMethodName"] = fmt.Sprintf("return \"%s\"", methodName)
	bodies["BodyMethodCall"] = fmt.Sprintf("return data.NewStringValue(\"%s 调用完成\"), nil", methodName)
	bodies["BodyMethodGetModifier"] = "return data.ModifierPublic"
	bodies["BodyMethodGetIsStatic"] = "return false"
	bodies["BodyMethodGetParams"] = "return []data.GetValue{}"
	bodies["BodyMethodGetVariables"] = "return []data.Variable{}"
	bodies["BodyMethodGetReturnType"] = "return data.NewBaseType(\"void\")"
	return bodies
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
	// 检查是否是外部包类型（形如 pkg.Type）
	if strings.Contains(typeName, ".") {
		parts := strings.Split(typeName, ".")
		if len(parts) == 2 {
			externalPkg := parts[0]
			// 优先使用配置映射 base 键（无别名）
			if cm.config != nil && cm.config.PackageMappings != nil {
				if _, ok := cm.config.PackageMappings[externalPkg]; ok {
					cm.AddImport(pkgName, cm.mapPackagePath(externalPkg))
					return
				}
				aliasKey := externalPkg + "src"
				if _, ok := cm.config.PackageMappings[aliasKey]; ok {
					cm.AddImportWithAlias(pkgName, cm.mapPackagePath(aliasKey), aliasKey)
					return
				}
			}
			// 未配置映射：根据是否标准库决定
			if cm.isStandardLibrary(externalPkg) {
				cm.AddImport(pkgName, cm.mapPackagePath(externalPkg))
				return
			}
			alias := externalPkg + "src"
			cm.AddImportWithAlias(pkgName, cm.mapPackagePath(alias), alias)
		}
	}
}

// 检查是否是标准库
func (cm *CodeManager) isStandardLibrary(pkgName string) bool {
	libs := []string{"fmt", "os", "io", "bufio", "bytes", "strings", "strconv", "time", "math", "sort", "reflect", "encoding", "crypto", "context", "errors", "log", "path", "filepath", "net", "http"}
	if cm.config != nil && len(cm.config.StandardLibs) > 0 {
		libs = cm.config.StandardLibs
	}

	for _, lib := range libs {
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

	// 如果没有找到映射，使用配置的默认前缀
	prefix := "github.com/php-any/origami"
	if cm.config != nil && cm.config.DefaultImportPrefix != "" {
		prefix = cm.config.DefaultImportPrefix
	}
	return fmt.Sprintf("%s/%s", prefix, pkgName)
}

// 生成函数体
func (cm *CodeManager) generateFunctionBody(functionName string, params []core.ParameterInfo, returns []core.ReturnTemplateData) string {
	// 占位：未使用此生成路径
	return "// function body not used by generator\n"
}

// 生成类体
func (cm *CodeManager) generateClassBody(className string, fields []core.FieldInfo, methods []core.MethodInfo) string {
	// 占位：未使用此生成路径
	return "// class body not used by generator\n"
}

// 生成方法体
func (cm *CodeManager) generateMethodBody(className, methodName string, params []core.ParameterInfo, returns []core.ReturnTemplateData) string {
	// 占位：未使用此生成路径
	return "// method body not used by generator\n"
}
