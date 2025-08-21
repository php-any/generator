package generator

import (
	"fmt"
	"reflect"
	"strings"
)

// 生成实例方法包装代码
func buildMethodFileBody(srcPkgPath, pkgName, typeName string, m reflect.Method) (string, bool) {
	// 支持更多参数类型组合
	mt := m.Type
	// 去掉接收者参数
	numIn := mt.NumIn() - 1
	if numIn < 0 {
		return "", false
	}

	// 参数类型收集
	paramTypes := make([]reflect.Type, 0, numIn)
	paramNames := make([]string, 0, numIn)
	for i := 0; i < numIn; i++ {
		t := mt.In(i + 1)
		paramTypes = append(paramTypes, t)
		paramNames = append(paramNames, "param"+strconvItoa(i))
	}
	// 优先通过源码提取真实参数名
	if names, ok := tryExtractParamNames(m.Func.Pointer(), numIn); ok {
		paramNames = names
	}

	// 返回值分析
	numOut := mt.NumOut()
	errorType := reflect.TypeOf((*error)(nil)).Elem()
	hasErr := false
	nonErrCount := numOut
	if numOut > 0 && mt.Out(numOut-1).Implements(errorType) {
		hasErr = true
		nonErrCount = numOut - 1
	}

	importAlias := pkgName + "src"

	// 检测需要的包导入：仅为会在代码里做类型断言的参数类型收集依赖包
	usedPkgs := make(map[string]bool)
	for _, t := range paramTypes {
		base := t
		if base.Kind() == reflect.Ptr {
			base = base.Elem()
		}
		// 跳过 *struct（直接解包 source，不做断言）与基础类型、slice
		if isPtrToStruct(t) || base.Kind() == reflect.String || base.Kind() == reflect.Int || base.Kind() == reflect.Int64 || base.Kind() == reflect.Bool || base.Kind() == reflect.Slice {
			continue
		}
		// 仅在 default 或 interface 分支会生成断言表达式，才收集包
		collectPkgPaths(t, usedPkgs)
	}
	// 源包通过 importAlias 导入，移除重复
	delete(usedPkgs, srcPkgPath)

	// 检测单一非错误返回值是否为跨包 *struct，以便导入 origami/<pkg> 并在调用处加前缀
	retPkgBase := ""
	retCtorPrefix := ""
	if nonErrCount == 1 {
		retT := mt.Out(0)
		if hasErr {
			retT = mt.Out(0)
		}
		if retT.Kind() == reflect.Ptr && retT.Elem().Kind() == reflect.Struct {
			pkgBase := pkgBaseName(retT.Elem().PkgPath())
			if pkgBase != "" && pkgBase != pkgName {
				retPkgBase = pkgBase
				retCtorPrefix = pkgBase + "."
			}
		}
	}

	b := &strings.Builder{}
	b.WriteString("import (\n")
	// 可选导入：errors 仅在存在参数校验时才使用
	if numIn > 0 {
		b.WriteString("\t\"errors\"\n")
	}
	// 导入在参数中实际使用的包
	for pkgPath := range usedPkgs {
		fmt.Fprintf(b, "\t\"%s\"\n", pkgPath)
	}
	// 导入因返回值跨包需要的 origami/<pkg>
	if retPkgBase != "" {
		modPath := getModulePath()
		if modPath != "" {
			fmt.Fprintf(b, "\t\"%s/origami/%s\"\n", modPath, retPkgBase)
		}
	}
	// time.Duration 入参需要导入 time 包
	for _, t := range paramTypes {
		base := t
		if base.Kind() == reflect.Ptr {
			base = base.Elem()
		}
		if base.PkgPath() == "time" && base.Name() == "Duration" {
			fmt.Fprintf(b, "\t\"time\"\n")
			break
		}
	}
	fmt.Fprintf(b, "\t%s %q\n", importAlias, srcPkgPath)
	b.WriteString("\t\"github.com/php-any/origami/data\"\n")
	if numIn > 0 {
		b.WriteString("\t\"github.com/php-any/origami/node\"\n")
	}
	b.WriteString(")\n\n")

	// type
	fmt.Fprintf(b, "type %s%sMethod struct {\n\tsource *%s.%s\n}\n\n", typeName, m.Name, importAlias, typeName)

	// Call
	fmt.Fprintf(b, "func (h *%s%sMethod) Call(ctx data.Context) (data.GetValue, data.Control) {\n\n", typeName, m.Name)
	for i := 0; i < numIn; i++ {
		fmt.Fprintf(b, "\ta%d, ok := ctx.GetIndexValue(%d)\n\tif !ok { return nil, data.NewErrorThrow(nil, errors.New(\"缺少参数, index: %d\")) }\n\n", i, i, i)
	}
	// 参数预处理：按类型严格处理
	for i, t := range paramTypes {
		base := t
		if base.Kind() == reflect.Ptr {
			base = base.Elem()
		}

		// *struct 参数：从代理类取出具体 source
		if isPtrToStruct(t) {
			clsName := t.Elem().Name()
			fmt.Fprintf(b, "\targ%[1]dClass := a%[1]d.(*data.ClassValue).Class.(*%sClass)\n", i, clsName)
			fmt.Fprintf(b, "\targ%[1]d := arg%[1]dClass.source\n", i)
			continue
		}
		// interface{} 参数：从 AnyValue 中提取并转换类型（将源包短名替换为导入别名）
		if t.Kind() == reflect.Interface {
			typeStr := t.String()
			typeStr = strings.ReplaceAll(typeStr, pkgBaseName(srcPkgPath)+".", importAlias+".")
			fmt.Fprintf(b, "\targ%[1]d := a%[1]d.(*data.AnyValue).Value.(%s)\n", i, typeStr)
			continue
		}
		// 其它常见类型
		switch base.Kind() {
		case reflect.String:
			fmt.Fprintf(b, "\targ%d := a%d.(*data.StringValue).AsString()\n", i, i)
		case reflect.Int:
			fmt.Fprintf(b, "\targ%d, err := a%d.(*data.IntValue).AsInt()\n\tif err != nil { return nil, data.NewErrorThrow(nil, err) }\n", i, i)
		case reflect.Int64:
			// time.Duration 特判
			if base.PkgPath() == "time" && base.Name() == "Duration" {
				fmt.Fprintf(b, "\targ%[1]dInt, err := a%[1]d.(*data.IntValue).AsInt()\n\tif err != nil { return nil, data.NewErrorThrow(nil, err) }\n\targ%[1]d := time.Duration(arg%[1]dInt)\n", i)
			} else {
				fmt.Fprintf(b, "\targ%[1]dInt, err := a%[1]d.(*data.IntValue).AsInt()\n\tif err != nil { return nil, data.NewErrorThrow(nil, err) }\n\targ%[1]d := int64(arg%[1]dInt)\n", i)
			}
		case reflect.Bool:
			fmt.Fprintf(b, "\targ%d, err := a%d.(*data.BoolValue).AsBool()\n\tif err != nil { return nil, data.NewErrorThrow(nil, err) }\n", i, i)
		case reflect.Slice:
			fmt.Fprintf(b, "\targ%d := *a%d.(*data.ArrayValue)\n", i, i)
		default:
			typeStr := t.String()
			typeStr = strings.ReplaceAll(typeStr, pkgBaseName(srcPkgPath)+".", importAlias+".")
			fmt.Fprintf(b, "\targ%[1]d := a%[1]d.(*data.AnyValue).Value.(%s)\n", i, typeStr)
		}
	}
	b.WriteString("\n")

	// 调用并接收返回值
	if hasErr && nonErrCount == 0 {
		// 仅有一个 error 返回值：使用 if err := call(...); err != nil 模式，避免 err 变量重复声明
		b.WriteString("\tif err := ")
		b.WriteString("h.source.")
		b.WriteString(m.Name)
		b.WriteString("(")
		for i := range paramTypes {
			if i > 0 {
				b.WriteString(", ")
			}
			fmt.Fprintf(b, "arg%d", i)
		}
		b.WriteString("); err != nil {\n\t\treturn nil, data.NewErrorThrow(nil, err)\n\t}\n")
	} else {
		if numOut > 0 {
			for j := 0; j < nonErrCount; j++ {
				if j == 0 {
					b.WriteString("\t")
				} else {
					b.WriteString(", ")
				}
				fmt.Fprintf(b, "ret%d", j)
			}
			if hasErr {
				if nonErrCount > 0 {
					b.WriteString(", ")
				} else {
					b.WriteString("\t")
				}
				b.WriteString("err")
			}
			b.WriteString(" := ")
		} else {
			b.WriteString("\t")
		}

		b.WriteString("h.source.")
		b.WriteString(m.Name)
		b.WriteString("(")
		for i := range paramTypes {
			if i > 0 {
				b.WriteString(", ")
			}
			fmt.Fprintf(b, "arg%d", i)
		}
		b.WriteString(")\n")

		// 错误处理（有非错误返回值时）
		if hasErr {
			b.WriteString("\tif err != nil {\n\t\treturn nil, data.NewErrorThrow(nil, err)\n\t}\n")
		}
	}

	// 成功返回值封装
	if nonErrCount == 0 {
		b.WriteString("\treturn nil, nil\n}\n\n")
	} else if nonErrCount == 1 {
		// 判断单返回是否为 *struct，若是则返回代理类 ClassValue
		retT := mt.Out(0)
		if hasErr {
			retT = mt.Out(0)
		}
		if retT.Kind() == reflect.Ptr && retT.Elem().Kind() == reflect.Struct {
			retName := retT.Elem().Name()
			fmt.Fprintf(b, "\treturn data.NewClassValue(%sNew%sClassFrom(ret0), ctx), nil\n}\n\n", retCtorPrefix, retName)
		} else {
			b.WriteString("\treturn data.NewAnyValue(ret0), nil\n}\n\n")
		}
	} else {
		b.WriteString("\treturn data.NewAnyValue([]any{")
		for j := 0; j < nonErrCount; j++ {
			if j > 0 {
				b.WriteString(", ")
			}
			fmt.Fprintf(b, "ret%d", j)
		}
		b.WriteString("}), nil\n}\n\n")
	}

	// GetName, GetModifier, GetIsStatic
	fmt.Fprintf(b, "func (h *%s%sMethod) GetName() string { return %q }\n", typeName, m.Name, lowerFirst(m.Name))
	fmt.Fprintf(b, "func (h *%s%sMethod) GetModifier() data.Modifier { return data.ModifierPublic }\n", typeName, m.Name)
	fmt.Fprintf(b, "func (h *%s%sMethod) GetIsStatic() bool { return true }\n", typeName, m.Name)

	// GetParams
	b.WriteString("func (h *")
	b.WriteString(typeName)
	b.WriteString(m.Name)
	b.WriteString("Method) GetParams() []data.GetValue { return []data.GetValue{\n")
	for i := 0; i < numIn; i++ {
		fmt.Fprintf(b, "\t\tnode.NewParameter(nil, %q, %d, nil, nil),\n", paramNames[i], i)
	}
	b.WriteString("\t}\n}\n\n")

	// GetVariables
	b.WriteString("func (h *")
	b.WriteString(typeName)
	b.WriteString(m.Name)
	b.WriteString("Method) GetVariables() []data.Variable { return []data.Variable{\n")
	for i := 0; i < numIn; i++ {
		fmt.Fprintf(b, "\t\tnode.NewVariable(nil, %q, %d, nil),\n", paramNames[i], i)
	}
	b.WriteString("\t}\n}\n\n")

	// GetReturnType：暂固定为 void（与 demo/log 一致）
	fmt.Fprintf(b, "func (h *%s%sMethod) GetReturnType() data.Types { return data.NewBaseType(\"void\") }\n", typeName, m.Name)

	return b.String(), true
}
