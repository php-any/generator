package scr

import (
	"fmt"
	"reflect"
	"strings"
)

// buildMethodFileBody 构建方法文件内容
func buildMethodFileBody(srcPkgPath, pkgName, typeName string, m reflect.Method, sourceIsPtr bool, fileCache *FileCache, structType reflect.Type, config *Config) (string, bool) {
	b := &strings.Builder{}
	importAlias := pkgName + "src"

	// 分析方法参数和返回值
	paramTypes, paramNames := analyzeMethodParams(m, sourceIsPtr)
	returnTypes := analyzeMethodReturns(m)

	// 收集导入
	collectMethodImportsToCache(srcPkgPath, pkgName, paramTypes, returnTypes, fileCache, config)

	// 生成方法结构体
	writeMethodStruct(b, typeName, m.Name, importAlias, structType, fileCache, srcPkgPath)

	// 生成方法实现
	writeMethodImplementation(b, typeName, m.Name, paramTypes, paramNames, returnTypes, importAlias, fileCache)

	// 在文件开头写入导入（在代码生成完成后，但需要插入到文件开头）
	content := b.String()
	b.Reset()
	writeImportsFromCache(b, fileCache)
	b.WriteString(content)

	return b.String(), true
}

// writeMethodStruct 写入方法结构体
func writeMethodStruct(b *strings.Builder, typeName, methodName, importAlias string, structType reflect.Type, fileCache *FileCache, srcPkgPath string) {
	// 标记使用的导入
	fileCache.MarkImportUsed("github.com/php-any/origami/data")
	if srcPkgPath != "" {
		fileCache.MarkImportUsed(srcPkgPath)
	}

	fmt.Fprintf(b, "type %s%sMethod struct {\n", typeName, methodName)
	if structType.Kind() == reflect.Interface {
		fmt.Fprintf(b, "\tsource %s.%s\n", importAlias, typeName)
	} else {
		fmt.Fprintf(b, "\tsource *%s.%s\n", importAlias, typeName)
	}
	b.WriteString("}\n\n")
}

// writeMethodImplementation 写入方法实现
func writeMethodImplementation(b *strings.Builder, typeName, methodName string, paramTypes []reflect.Type, paramNames []string, returnTypes []reflect.Type, importAlias string, fileCache *FileCache) {
	fmt.Fprintf(b, "func (h *%s%sMethod) Call(ctx data.Context) (data.GetValue, data.Control) {\n", typeName, methodName)

	// 标记使用的导入
	fileCache.MarkImportUsed("github.com/php-any/origami/data")
	if len(paramNames) > 0 {
		fileCache.MarkImportUsed("fmt")
		fileCache.MarkImportUsed("github.com/php-any/generator/utils")
	}

	// 参数类型转换
	if len(paramNames) > 0 {
		for i, pName := range paramNames {
			typeStr := getTypeString(paramTypes[i], NewFileCache())
			// 替换包名为别名
			paramType := paramTypes[i]
			if paramType.Kind() == reflect.Ptr && paramType.Elem() != nil {
				paramType = paramType.Elem()
			}
			if paramType.PkgPath() != "" {
				originalPkgName := pkgBaseName(paramType.PkgPath())
				if strings.Contains(typeStr, originalPkgName+".") {
					typeStr = strings.ReplaceAll(typeStr, originalPkgName+".", importAlias+".")
				}
			}
			fmt.Fprintf(b, "\t%s, err := utils.ConvertFromIndex[%s](ctx, %d)\n", pName, typeStr, i)
			fmt.Fprintf(b, "\tif err != nil { return nil, data.NewErrorThrow(nil, fmt.Errorf(\"参数转换失败: %%v\", err)) }\n")
		}
		b.WriteString("\n")
	}

	// 方法调用
	sourceCall := "h.source"
	if len(returnTypes) == 0 {
		fmt.Fprintf(b, "\t%s.%s(", sourceCall, methodName)
		for i, pName := range paramNames {
			if i > 0 {
				b.WriteString(", ")
			}
			fmt.Fprintf(b, "%s", pName)
		}
		fmt.Fprintf(b, ")\n\treturn nil, nil\n")
	} else if len(returnTypes) == 1 {
		fmt.Fprintf(b, "\tret0 := %s.%s(", sourceCall, methodName)
		for i, pName := range paramNames {
			if i > 0 {
				b.WriteString(", ")
			}
			fmt.Fprintf(b, "%s", pName)
		}
		fmt.Fprintf(b, ")\n")
		if returnTypes[0].Kind() == reflect.Ptr && returnTypes[0].Elem().Kind() == reflect.Struct {
			fmt.Fprintf(b, "\treturn data.NewClassValue(New%sClassFrom(ret0), ctx), nil\n", returnTypes[0].Elem().Name())
		} else {
			fmt.Fprintf(b, "\treturn data.NewAnyValue(ret0), nil\n")
		}
	} else {
		fmt.Fprintf(b, "\tret0, ret1 := %s.%s(", sourceCall, methodName)
		for i, pName := range paramNames {
			if i > 0 {
				b.WriteString(", ")
			}
			fmt.Fprintf(b, "%s", pName)
		}
		fmt.Fprintf(b, ")\n\treturn data.NewArrayValue([]data.Value{data.NewAnyValue(ret0), data.NewAnyValue(ret1)}), nil\n")
	}

	b.WriteString("}\n\n")

	// 写入方法接口实现（名称小驼峰）
	fmt.Fprintf(b, "func (h *%s%sMethod) GetName() string { return \"%s\" }\n", typeName, methodName, lowerFirst(methodName))
	fmt.Fprintf(b, "func (h *%s%sMethod) GetModifier() data.Modifier { return data.ModifierPublic }\n", typeName, methodName)
	fmt.Fprintf(b, "func (h *%s%sMethod) GetIsStatic() bool { return true }\n", typeName, methodName)

	// 只在有参数时才生成参数相关方法
	if len(paramTypes) > 0 {
		// 标记使用的导入
		fileCache.MarkImportUsed("github.com/php-any/origami/node")

		// 参数清单
		fmt.Fprintf(b, "func (h *%s%sMethod) GetParams() []data.GetValue { return []data.GetValue{\n", typeName, methodName)
		for i := range paramTypes {
			pName := paramNames[i]
			fmt.Fprintf(b, "\t\tnode.NewParameter(nil, \"%s\", %d, nil, nil),\n", pName, i)
		}
		fmt.Fprintf(b, "\t}\n}\n")
		// 变量清单
		fmt.Fprintf(b, "func (h *%s%sMethod) GetVariables() []data.Variable { return []data.Variable{\n", typeName, methodName)
		for i := range paramTypes {
			pName := paramNames[i]
			fmt.Fprintf(b, "\t\tnode.NewVariable(nil, \"%s\", %d, nil),\n", pName, i)
		}
		fmt.Fprintf(b, "\t}\n}\n")
	} else {
		// 无参数时返回空切片
		fmt.Fprintf(b, "func (h *%s%sMethod) GetParams() []data.GetValue { return []data.GetValue{} }\n", typeName, methodName)
		fmt.Fprintf(b, "func (h *%s%sMethod) GetVariables() []data.Variable { return []data.Variable{} }\n", typeName, methodName)
	}

	// 返回类型
	fmt.Fprintf(b, "func (h *%s%sMethod) GetReturnType() data.Types { return data.NewBaseType(\"void\") }\n", typeName, methodName)
}

// analyzeMethodParams 分析方法参数
func analyzeMethodParams(m reflect.Method, sourceIsPtr bool) ([]reflect.Type, []string) {
	mt := m.Type
	numIn := mt.NumIn()

	// 计算实际参数数量（排除接收者）
	startIndex := 0
	if sourceIsPtr {
		startIndex = 1 // 跳过接收者
	}

	paramTypes := make([]reflect.Type, 0, numIn-startIndex)
	paramNames := make([]string, 0, numIn-startIndex)

	for i := startIndex; i < numIn; i++ {
		paramType := mt.In(i)
		paramTypes = append(paramTypes, paramType)
		paramNames = append(paramNames, fmt.Sprintf("param%d", i-startIndex))
	}

	return paramTypes, paramNames
}

// analyzeMethodReturns 分析方法返回值
func analyzeMethodReturns(m reflect.Method) []reflect.Type {
	mt := m.Type
	numOut := mt.NumOut()

	returnTypes := make([]reflect.Type, 0, numOut)
	for i := 0; i < numOut; i++ {
		returnType := mt.Out(i)
		returnTypes = append(returnTypes, returnType)
	}

	return returnTypes
}

// collectMethodTypeImports 收集方法类型的导入
func collectMethodTypeImports(mt reflect.Type, srcPkgPath string, fileCache *FileCache, config *Config) {
	// 收集参数类型（只收集直接类型，不递归收集字段）
	for i := 0; i < mt.NumIn(); i++ {
		collectDirectTypeImports(mt.In(i), srcPkgPath, fileCache, config)
	}

	// 收集返回值类型（只收集直接类型，不递归收集字段）
	for i := 0; i < mt.NumOut(); i++ {
		collectDirectTypeImports(mt.Out(i), srcPkgPath, fileCache, config)
	}
}
