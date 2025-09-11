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
	paramTypes, paramNames, isVariadic, variadicElem := analyzeMethodParams(m, sourceIsPtr)
	returnTypes := analyzeMethodReturns(m)

	// 收集导入
	collectMethodImportsToCache(srcPkgPath, pkgName, paramTypes, returnTypes, fileCache, config)

	// 生成方法结构体
	writeMethodStruct(b, typeName, m.Name, importAlias, structType, fileCache, srcPkgPath)

	// 生成方法实现
	writeMethodImplementation(b, typeName, m.Name, paramTypes, paramNames, returnTypes, importAlias, fileCache, isVariadic, variadicElem)

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
func writeMethodImplementation(b *strings.Builder, typeName, methodName string, paramTypes []reflect.Type, paramNames []string, returnTypes []reflect.Type, importAlias string, fileCache *FileCache, isVariadic bool, variadicElem reflect.Type) {
	fmt.Fprintf(b, "func (h *%s%sMethod) Call(ctx data.Context) (data.GetValue, data.Control) {\n", typeName, methodName)

	// 标记使用的导入
	fileCache.MarkImportUsed("github.com/php-any/origami/data")
	// 仅当存在固定参数需要转换时引入 fmt 和 utils
	fixedCount := len(paramNames)
	if isVariadic {
		fixedCount = fixedCount - 1
	}
	if fixedCount > 0 {
		fileCache.MarkImportUsed("fmt")
		fileCache.MarkImportUsed("github.com/php-any/generator/utils")
	}

	// 参数类型转换（可变参数仅转换固定部分）
	if len(paramNames) > 0 {
		endIdx := len(paramNames)
		if isVariadic {
			endIdx = endIdx - 1
		}
		// 从 importAlias 反推出原包短名（去掉 src 后缀）
		origPkgName := ""
		if strings.HasSuffix(importAlias, "src") {
			origPkgName = strings.TrimSuffix(importAlias, "src")
		}
		writeParameterConversion(b, paramTypes, paramNames, endIdx, fileCache, origPkgName, importAlias)
		b.WriteString("\n")
	}

	// 处理可变参数
	origPkgName := ""
	if strings.HasSuffix(importAlias, "src") {
		origPkgName = strings.TrimSuffix(importAlias, "src")
	}
	writeVariadicParameterHandling(b, isVariadic, variadicElem, paramNames, fileCache, origPkgName, importAlias)

	// 方法调用
	sourceCall := "h.source"
	if len(returnTypes) == 0 {
		fmt.Fprintf(b, "\t%s.%s(", sourceCall, methodName)
		for i, pName := range paramNames {
			if i > 0 {
				b.WriteString(", ")
			}
			if isVariadic && i == len(paramNames)-1 {
				fmt.Fprintf(b, "%s...", pName)
			} else {
				fmt.Fprintf(b, "%s", pName)
			}
		}
		fmt.Fprintf(b, ")\n\treturn nil, nil\n")
	} else if len(returnTypes) == 1 {
		fmt.Fprintf(b, "\tret0 := %s.%s(", sourceCall, methodName)
		for i, pName := range paramNames {
			if i > 0 {
				b.WriteString(", ")
			}
			if isVariadic && i == len(paramNames)-1 {
				fmt.Fprintf(b, "%s...", pName)
			} else {
				fmt.Fprintf(b, "%s", pName)
			}
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
			if isVariadic && i == len(paramNames)-1 {
				fmt.Fprintf(b, "%s...", pName)
			} else {
				fmt.Fprintf(b, "%s", pName)
			}
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
func analyzeMethodParams(m reflect.Method, sourceIsPtr bool) ([]reflect.Type, []string, bool, reflect.Type) {
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

	isVariadic := mt.IsVariadic()
	var variadicElem reflect.Type
	if isVariadic && numIn-startIndex > 0 {
		last := mt.In(numIn - 1)
		if last.Kind() == reflect.Slice {
			variadicElem = last.Elem()
		}
	}

	return paramTypes, paramNames, isVariadic, variadicElem
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
