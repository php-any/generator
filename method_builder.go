package generator

import (
	"fmt"
	"os"
	"reflect"
	"strings"
)

// 生成实例方法包装代码
// sourceIsPtr 决定 source 字段是否为指针类型（结构体为 *，接口为非 *）
func buildMethodFileBody(srcPkgPath, pkgName, typeName string, m reflect.Method, sourceIsPtr bool) (string, bool) {
	// 支持更多参数类型组合
	mt := m.Type
	// 参数个数计算
	numIn := mt.NumIn()
	if sourceIsPtr {
		// 结构体方法，需要去掉接收者参数
		numIn = numIn - 1
		if numIn < 0 {
			return "", false
		}
	}
	// 接口方法，不需要减去接收者，直接使用 mt.NumIn()

	// 参数类型收集
	paramTypes := make([]reflect.Type, 0, numIn)
	paramNames := make([]string, 0, numIn)
	for i := 0; i < numIn; i++ {
		// 接口方法参数从 In(0) 开始，结构体方法从 In(1) 开始（跳过接收者）
		var t reflect.Type
		if sourceIsPtr {
			t = mt.In(i + 1) // 结构体方法，跳过接收者
		} else {
			t = mt.In(i) // 接口方法，不跳过接收者
		}
		paramTypes = append(paramTypes, t)
		paramNames = append(paramNames, "param"+strconvItoa(i))
	}
	// 优先通过源码提取真实参数名
	if m.Func.IsValid() {
		if names, ok := tryExtractParamNames(m.Func.Pointer(), numIn); ok {
			paramNames = names
		}
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

	// 检测单一非错误返回值是否为跨包 *struct 或接口，以便导入 origami/<pkg> 并在调用处加前缀
	retPkgBase := ""
	retCtorPrefix := ""
	if nonErrCount == 1 {
		retT := mt.Out(0)
		if hasErr {
			retT = mt.Out(0)
		}
		if retT.Kind() == reflect.Ptr && retT.Elem().Kind() == reflect.Struct {
			// 返回 *struct 类型
			pkgBase := pkgBaseName(retT.Elem().PkgPath())
			if pkgBase != "" && pkgBase != pkgName {
				retPkgBase = pkgBase
				retCtorPrefix = pkgBase + "."
			}
		} else if retT.Kind() == reflect.Interface && retT.PkgPath() != "" && retT.Name() != "" {
			// 返回具名接口类型
			pkgBase := pkgBaseName(retT.PkgPath())
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
	star := ""
	if sourceIsPtr {
		star = "*"
	}
	fmt.Fprintf(b, "type %s%sMethod struct {\n\tsource %s%s.%s\n}\n\n", typeName, m.Name, star, importAlias, typeName)

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

		// 检查是否为可变参数
		if m.Type.IsVariadic() && i == len(paramTypes)-1 {
			// 这是可变参数，生成的代码可能需要手动修改
			fmt.Fprintf(os.Stderr, "⚠️  检测到可变参数：%s.%s 方法的第 %d 个参数, 如果需要修改参数值, 可能需要使用 NewParametersReference 替代 NewParameter 改为引用传参 %s\n", typeName, m.Name, i, paramNames[i])
			fmt.Fprintf(os.Stderr, "   生成的文件：origami/%s/%s_%s_method.go\n\n", pkgName, strings.ToLower(typeName), strings.ToLower(m.Name))
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
			// 将切片按元素真实类型展开，而不是一律使用 []any
			// 计算元素类型字符串，并将源包短名替换为导入别名（如 driversrc.Value）
			elem := base.Elem()
			elemTypeStr := elem.String()
			// 默认按源包替换为 importAlias 前缀
			elemTypeStr = strings.ReplaceAll(elemTypeStr, pkgBaseName(srcPkgPath)+".", importAlias+".")
			// 特判：database/sql/driver 中的 Value 由于是别名，反射可能显示为 interface{}
			if elem.Kind() == reflect.Interface && elem.PkgPath() == "" {
				if srcPkgPath == "database/sql/driver" {
					// 显式使用 driversrc.Value
					elemTypeStr = importAlias + ".Value"
				} else {
					// 其它包的空接口一律使用 any
					elemTypeStr = "any"
				}
			}
			sliceTypeStr := "[]" + elemTypeStr

			fmt.Fprintf(b, "\targ%d := make(%s, 0)\n", i, sliceTypeStr)
			fmt.Fprintf(b, "\tfor _, v := range a%d.(*data.ArrayValue).Value {\n", i)
			// 将 any 转为具体元素类型再追加；若元素类型为 any，则无需断言
			if elemTypeStr == "any" || elemTypeStr == "interface{}" {
				fmt.Fprintf(b, "\t\targ%d = append(arg%d, v)\n", i, i)
			} else {
				fmt.Fprintf(b, "\t\targ%d = append(arg%d, v.(%s))\n", i, i, elemTypeStr)
			}
			fmt.Fprintf(b, "\t}\n")
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
			// 变参最后一个参数需要使用 ... 展开
			if m.Type.IsVariadic() && i == len(paramTypes)-1 {
				fmt.Fprintf(b, "arg%d...", i)
			} else {
				fmt.Fprintf(b, "arg%d", i)
			}
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
			// 变参最后一个参数需要使用 ... 展开
			if m.Type.IsVariadic() && i == len(paramTypes)-1 {
				fmt.Fprintf(b, "arg%d...", i)
			} else {
				fmt.Fprintf(b, "arg%d", i)
			}
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
			// 返回 *struct 类型，需要生成代理类
			retName := retT.Elem().Name()
			fmt.Fprintf(b, "\treturn data.NewClassValue(%sNew%sClassFrom(ret0), ctx), nil\n}\n\n", retCtorPrefix, retName)
		} else if retT.Kind() == reflect.Interface && retT.PkgPath() != "" && retT.Name() != "" {
			// 返回具名接口类型，需要生成代理类
			retName := retT.Name()
			fmt.Fprintf(b, "\treturn data.NewClassValue(%sNew%sClassFrom(ret0), ctx), nil\n}\n\n", retCtorPrefix, retName)
		} else {
			switch retT.Kind() {
			case reflect.Bool:
				b.WriteString("\treturn data.NewBoolValue(ret0), nil\n}\n\n")
			case reflect.Int:
				b.WriteString("\treturn data.NewIntValue(ret0), nil\n}\n\n")
			case reflect.Int64:
				b.WriteString("\treturn data.NewIntValue(int(ret0)), nil\n}\n\n")
			case reflect.String:
				b.WriteString("\treturn data.NewStringValue(ret0), nil\n}\n\n")
			default:
				b.WriteString("\treturn data.NewAnyValue(ret0), nil\n}\n\n")
			}
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
	// 注意：对于可变参数（variadic parameters），需要使用 NewParameters 来引用参数
	b.WriteString("func (h *")
	b.WriteString(typeName)
	b.WriteString(m.Name)
	b.WriteString("Method) GetParams() []data.GetValue { return []data.GetValue{\n")
	for i := 0; i < numIn; i++ {
		// 检查是否为可变参数
		if m.Type.IsVariadic() && i == len(paramTypes)-1 {
			// 这是可变参数，使用 NewParameters
			fmt.Fprintf(b, "\t\tnode.NewParameters(nil, %q, %d, nil, nil),\n", paramNames[i], i)
		} else {
			// 普通参数，使用 NewParameter
			fmt.Fprintf(b, "\t\tnode.NewParameter(nil, %q, %d, nil, nil),\n", paramNames[i], i)
		}
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
