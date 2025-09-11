package scr

import (
	"fmt"
	"reflect"
	"strings"
)

// buildFunctionFileBody 构建函数文件内容
func buildFunctionFileBody(srcPkgPath, pkgName, namePrefix, funcName string, t reflect.Type, fileCache *FileCache, config *Config) string {
	b := &strings.Builder{}
	importAlias := pkgName + "src"

	// 分析函数参数和返回值
	paramTypes, paramNames, isVariadic, variadicElem := analyzeFunctionParams(t)
	returnTypes := analyzeFunctionReturns(t)

	// 收集导入
	collectFunctionImportsToCache(srcPkgPath, pkgName, paramTypes, returnTypes, fileCache, config)

	// 生成函数结构体
	writeFunctionStruct(b, funcName, fileCache, srcPkgPath)

	// 生成函数实现
	origPkgName := ""
	if srcPkgPath != "" {
		origPkgName = pkgBaseName(srcPkgPath)
	}
	writeFunctionImplementation(b, namePrefix, funcName, paramTypes, paramNames, returnTypes, importAlias, fileCache, isVariadic, variadicElem, origPkgName)

	// 在文件开头写入导入（在代码生成完成后，但需要插入到文件开头）
	content := b.String()
	b.Reset()
	writeImportsFromCache(b, fileCache)
	b.WriteString(content)

	return b.String()
}

// writeFunctionStruct 写入函数结构体
func writeFunctionStruct(b *strings.Builder, funcName string, fileCache *FileCache, srcPkgPath string) {
	// 标记使用的导入
	fileCache.MarkImportUsed("github.com/php-any/origami/data")
	if srcPkgPath != "" {
		fileCache.MarkImportUsed(srcPkgPath)
	}

	fmt.Fprintf(b, "type %sFunction struct{}\n\n", funcName)

	// 添加构造函数
	fmt.Fprintf(b, "func New%sFunction() data.FuncStmt {\n", funcName)
	fmt.Fprintf(b, "\treturn &%sFunction{}\n", funcName)
	fmt.Fprintf(b, "}\n\n")
}

// writeFunctionImplementation 写入函数实现
func writeFunctionImplementation(b *strings.Builder, namePrefix, funcName string, paramTypes []reflect.Type, paramNames []string, returnTypes []reflect.Type, importAlias string, fileCache *FileCache, isVariadic bool, variadicElem reflect.Type, origPkgName string) {
	fmt.Fprintf(b, "func (h *%sFunction) Call(ctx data.Context) (data.GetValue, data.Control) {\n", funcName)

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
		writeParameterConversion(b, paramTypes, paramNames, endIdx, fileCache, origPkgName, importAlias)
		b.WriteString("\n")
	}

	// 处理可变参数
	writeVariadicParameterHandling(b, isVariadic, variadicElem, paramNames, fileCache, origPkgName, importAlias)

	// 函数调用
	if len(returnTypes) == 0 {
		fmt.Fprintf(b, "\t%s.%s(", importAlias, funcName)
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
		fmt.Fprintf(b, "\tret0 := %s.%s(", importAlias, funcName)
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
		fmt.Fprintf(b, "\tret0, ret1 := %s.%s(", importAlias, funcName)
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

	// 写入函数接口实现
	fmt.Fprintf(b, "func (h *%sFunction) GetName() string { return \"%s\\\\%s\" }\n", funcName, namePrefix, funcName)
	fmt.Fprintf(b, "func (h *%sFunction) GetIsStatic() bool { return false }\n", funcName)

	// 参数清单
	if len(paramNames) > 0 {
		// 标记使用的导入
		fileCache.MarkImportUsed("github.com/php-any/origami/node")
	}
	fmt.Fprintf(b, "func (h *%sFunction) GetParams() []data.GetValue { return []data.GetValue{\n", funcName)
	for i, paramName := range paramNames {
		datExpr := getDataTypeExpr(paramTypes[i])
		fmt.Fprintf(b, "\t\tnode.NewParameter(nil, \"%s\", %d, nil, %s),\n", paramName, i, datExpr)
	}
	fmt.Fprintf(b, "\t}\n}\n")

	// 变量清单
	fmt.Fprintf(b, "func (h *%sFunction) GetVariables() []data.Variable { return []data.Variable{\n", funcName)
	for i, paramName := range paramNames {
		datExpr := getDataTypeExpr(paramTypes[i])
		fmt.Fprintf(b, "\t\tnode.NewVariable(nil, \"%s\", %d, %s),\n", paramName, i, datExpr)
	}
	fmt.Fprintf(b, "\t}\n}\n")

	// 返回类型
	if len(returnTypes) == 0 {
		fmt.Fprintf(b, "func (h *%sFunction) GetReturnType() data.Types { return data.NewBaseType(\"void\") }\n", funcName)
	} else if len(returnTypes) == 1 {
		retTypeExpr := getDataTypeExpr(returnTypes[0])
		fmt.Fprintf(b, "func (h *%sFunction) GetReturnType() data.Types { return %s }\n", funcName, retTypeExpr)
	} else {
		// 多返回值用数组类型
		left := namePrefix
		right := funcName + "Result"
		retTypeExpr := fmt.Sprintf("data.NewBaseType(\"%s\\\\%s\")", left, right)
		fmt.Fprintf(b, "func (h *%sFunction) GetReturnType() data.Types { return %s }\n", funcName, retTypeExpr)
	}
}

// analyzeFunctionParams 分析函数参数
func analyzeFunctionParams(t reflect.Type) ([]reflect.Type, []string, bool, reflect.Type) {
	numIn := t.NumIn()

	paramTypes := make([]reflect.Type, 0, numIn)
	paramNames := make([]string, 0, numIn)

	for i := 0; i < numIn; i++ {
		paramType := t.In(i)
		paramTypes = append(paramTypes, paramType)
		paramNames = append(paramNames, fmt.Sprintf("param%d", i))
	}

	isVariadic := t.IsVariadic()
	var variadicElem reflect.Type
	if isVariadic && numIn > 0 {
		last := t.In(numIn - 1)
		if last.Kind() == reflect.Slice {
			variadicElem = last.Elem()
		}
	}

	return paramTypes, paramNames, isVariadic, variadicElem
}

// analyzeFunctionReturns 分析函数返回值
func analyzeFunctionReturns(t reflect.Type) []reflect.Type {
	numOut := t.NumOut()

	returnTypes := make([]reflect.Type, 0, numOut)
	for i := 0; i < numOut; i++ {
		returnType := t.Out(i)
		returnTypes = append(returnTypes, returnType)
	}

	return returnTypes
}

// getDataTypeExpr 将 Go 类型映射为 data.* 类型表达式字符串
func getDataTypeExpr(t reflect.Type) string {
	// 简化处理，统一使用 nil
	return "nil"
}
