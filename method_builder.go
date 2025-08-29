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
		// 命名的基础类型别名也需要导入其所属包（例如 events.WindowEventType）
		if base.PkgPath() != "" && base.Name() != "" {
			switch base.Kind() {
			case reflect.Int, reflect.Int64, reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Float32, reflect.Float64, reflect.Bool, reflect.String:
				usedPkgs[base.PkgPath()] = true
			}
		}
		// 跳过 *struct（直接解包 source，不做断言）与基础类型、slice
		if isPtrToStruct(t) || base.Kind() == reflect.String || base.Kind() == reflect.Int || base.Kind() == reflect.Int64 || base.Kind() == reflect.Bool || base.Kind() == reflect.Slice {
			continue
		}
		// map 键值类型依赖收集
		if base.Kind() == reflect.Map {
			kt := base.Key()
			vt := base.Elem()
			for kt.Kind() == reflect.Ptr {
				kt = kt.Elem()
			}
			for vt.Kind() == reflect.Ptr {
				vt = vt.Elem()
			}
			if kt.PkgPath() != "" && kt.Name() != "" {
				usedPkgs[kt.PkgPath()] = true
			}
			if vt.PkgPath() != "" && vt.Name() != "" {
				usedPkgs[vt.PkgPath()] = true
			}
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
			fmt.Fprintf(b, "\t\"github.com/php-any/origami/std/%s\"\n", retPkgBase)
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
		fmt.Fprintf(b, "\ta%d, ok := ctx.GetIndexValue(%d)\n\tif !ok { return nil, data.NewErrorThrow(nil, errors.New(\"%s.%s 缺少参数, index: %d\")) }\n\n", i, i, typeName, m.Name, i)
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

		// *struct 参数：支持 ClassValue 与 AnyValue 双路径
		if isPtrToStruct(t) {
			var ptrTypeStr string
			var elemTypeStr string
			// 基于元素类型（去一层 *）计算类型串，确保跨包（如 redis）前缀正确
			el := t.Elem()
			if el == nil {
				return "", false
			}
			if el.PkgPath() != "" && el.Name() != "" {
				if el.PkgPath() == srcPkgPath {
					elemTypeStr = importAlias + "." + el.Name()
					ptrTypeStr = "*" + elemTypeStr
				} else {
					elemTypeStr = pkgBaseName(el.PkgPath()) + "." + el.Name()
					ptrTypeStr = "*" + elemTypeStr
				}
			} else {
				// 回退：保持原有字符串并替换当前包短名为导入别名
				elemTypeStr = strings.ReplaceAll(el.String(), pkgBaseName(srcPkgPath)+".", importAlias+".")
				ptrTypeStr = "*" + elemTypeStr
			}
			fmt.Fprintf(b, "\tvar arg%[1]d %s\n", i, ptrTypeStr)
			fmt.Fprintf(b, "\tswitch v := a%[1]d.(type) {\n", i)
			// ClassValue: 支持源为 *T 或 T
			fmt.Fprintf(b, "\tcase *data.ClassValue:\n\t\tif p, ok := v.Class.(interface{ GetSource() any }); ok { \n\t\t\tif src := p.GetSource(); src != nil {\n\t\t\t\tswitch s := src.(type) {\n\t\t\t\tcase %s:\n\t\t\t\t\targ%d = s\n\t\t\t\tcase %s:\n\t\t\t\t\targ%d = &s\n\t\t\t\t}\n\t\t\t}\n\t\t} else { return nil, data.NewErrorThrow(nil, errors.New(\"%s.%s 参数类型不支持, index: %d\")) }\n", ptrTypeStr, i, elemTypeStr, i, typeName, m.Name, i)
			// 支持任意实现 GetSource 的类型
			fmt.Fprintf(b, "\tcase data.GetSource:\n\t\tif src := v.GetSource(); src != nil { switch s := src.(type) { case %[2]s: arg%[1]d = s; case %[3]s: arg%[1]d = &s } } else { return nil, data.NewErrorThrow(nil, errors.New(\"%[4]s.%[5]s 参数类型不支持, index: %[1]d\")) }\n", i, ptrTypeStr, elemTypeStr, typeName, m.Name)
			// AnyValue: 同时支持 *T 与 T
			fmt.Fprintf(b, "\tcase *data.AnyValue:\n\t\tswitch vv := v.Value.(type) { case %[2]s: arg%[1]d = vv; case %[3]s: arg%[1]d = &vv; default: return nil, data.NewErrorThrow(nil, errors.New(\"%[4]s.%[5]s 参数类型不支持, index: %[1]d\")) }\n", i, ptrTypeStr, elemTypeStr, typeName, m.Name)
			fmt.Fprintf(b, "\tdefault:\n\t\treturn nil, data.NewErrorThrow(nil, errors.New(\"%s.%s 参数类型不支持, index: %d\"))\n\t}\n", typeName, m.Name, i)
			continue
		}
		// 接口参数：支持 ClassValue 与 AnyValue 双路径，GetSource() 返回 any
		if t.Kind() == reflect.Interface {
			// 特判内置 error 接口（PkgPath 为空，Name 为 "error"）
			if t.PkgPath() == "" && t.Name() == "error" {
				fmt.Fprintf(b, "\tvar arg%[1]d error\n", i)
				fmt.Fprintf(b, "\tswitch v := a%[1]d.(type) {\n", i)
				fmt.Fprintf(b, "\tcase *data.AnyValue:\n\t\tif vv, ok := v.Value.(error); ok { arg%[1]d = vv } else { return nil, data.NewErrorThrow(nil, errors.New(\"%[2]s.%[3]s 参数类型需要 error, index: %[1]d\")) }\n", i, typeName, m.Name)
				fmt.Fprintf(b, "\tcase data.GetSource:\n\t\tif src := v.GetSource(); src != nil { if e, ok := src.(error); ok { arg%[1]d = e } else { return nil, data.NewErrorThrow(nil, errors.New(\"%[2]s.%[3]s 参数类型需要 error, index: %[1]d\")) } }\n", i, typeName, m.Name)
				fmt.Fprintf(b, "\tdefault:\n\t\treturn nil, data.NewErrorThrow(nil, errors.New(\"%s.%s 参数类型不支持, index: %d\"))\n\t}\n", typeName, m.Name, i)
				continue
			}

			// 构造接口类型字符串：同包用导入别名，跨包用对方短名；无包路径时用接口名
			ifaceName := t.Name()
			var fullIface string
			if ifaceName != "" {
				if t.PkgPath() == "" {
					fullIface = ifaceName
				} else if pkgBaseName(t.PkgPath()) == pkgBaseName(srcPkgPath) {
					fullIface = importAlias + "." + ifaceName
				} else {
					fullIface = pkgBaseName(t.PkgPath()) + "." + ifaceName
				}
			} else {
				fullIface = t.String()
				if strings.Contains(fullIface, pkgBaseName(srcPkgPath)+".") {
					fullIface = strings.ReplaceAll(fullIface, pkgBaseName(srcPkgPath)+".", importAlias+".")
				}
			}
			if ifaceName != "" {
				fmt.Fprintf(b, "\tvar arg%[1]d %s\n", i, fullIface)
				fmt.Fprintf(b, "\tswitch v := a%[1]d.(type) {\n", i)
				fmt.Fprintf(b, "\tcase *data.ClassValue:\n\t\tif p, ok := v.Class.(interface{ GetSource() any }); ok { \n\t\t\t// 检查 GetSource 返回的类型，如果是指针则解引用\n\t\t\tif src := p.GetSource(); src != nil {\n\t\t\t\tif ptr, ok := src.(*%s); ok {\n\t\t\t\t\targ%d = *ptr\n\t\t\t\t} else {\n\t\t\t\t\targ%d = src.(%s)\n\t\t\t\t}\n\t\t\t}\n\t\t} else { return nil, data.NewErrorThrow(nil, errors.New(\"%s.%s 参数类型不支持, index: %d\")) }\n", fullIface, i, i, fullIface, typeName, m.Name, i)
				fmt.Fprintf(b, "\tcase data.GetSource:\n\t\tif src := v.GetSource(); src != nil { if ptr, ok := src.(*%[2]s); ok { arg%[1]d = *ptr } else { arg%[1]d = src.(%[2]s) } } else { return nil, data.NewErrorThrow(nil, errors.New(\"%[3]s.%[4]s 参数类型不支持, index: %[1]d\")) }\n", i, fullIface, typeName, m.Name)
				fmt.Fprintf(b, "\tcase *data.AnyValue:\n\t\targ%[1]d = v.Value.(%s)\n", i, fullIface)
				fmt.Fprintf(b, "\tdefault:\n\t\treturn nil, data.NewErrorThrow(nil, errors.New(\"%s.%s 参数类型不支持, index: %d\"))\n\t}\n", typeName, m.Name, i)
			} else {
				fmt.Fprintf(b, "\targ%[1]d := a%[1]d.(*data.AnyValue).Value\n", i)
			}
			continue
		}
		// 其它常见类型
		// 特判：具名基础类型别名（同包或跨包），例如 events.WindowEventType 基于 uint
		if base.Name() != "" && base.PkgPath() != "" {
			var typeStr string
			if base.PkgPath() == srcPkgPath {
				typeStr = importAlias + "." + base.Name()
			} else {
				typeStr = pkgBaseName(base.PkgPath()) + "." + base.Name()
			}
			switch base.Kind() {
			case reflect.Int:
				fmt.Fprintf(b, "\tvar arg%[1]d %s\n", i, typeStr)
				fmt.Fprintf(b, "\targ%[1]dInt, err := a%[1]d.(*data.IntValue).AsInt()\n\tif err != nil { return nil, data.NewErrorThrow(nil, err) }\n\targ%[1]d = %s(arg%[1]dInt)\n", i, typeStr)
				continue
			case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
				fmt.Fprintf(b, "\tvar arg%[1]d %s\n", i, typeStr)
				fmt.Fprintf(b, "\targ%[1]dInt, err := a%[1]d.(*data.IntValue).AsInt()\n\tif err != nil { return nil, data.NewErrorThrow(nil, err) }\n\targ%[1]d = %s(arg%[1]dInt)\n", i, typeStr)
				continue
			case reflect.Float32, reflect.Float64:
				fmt.Fprintf(b, "\tvar arg%[1]d %s\n", i, typeStr)
				fmt.Fprintf(b, "\targ%[1]dF, err := a%[1]d.(*data.IntValue).AsInt()\n\tif err != nil { return nil, data.NewErrorThrow(nil, err) }\n\targ%[1]d = %s(float64(arg%[1]dF))\n", i, typeStr)
				continue
			}
		}

		switch base.Kind() {
		case reflect.Map:
			// 生成 map[K]V 的断言转换
			kt := base.Key()
			vt := base.Elem()
			formatType := func(t reflect.Type) string {
				for t.Kind() == reflect.Ptr {
					t = t.Elem()
				}
				if t.Name() != "" {
					if t.PkgPath() == "" {
						return t.Name()
					}
					if pkgBaseName(t.PkgPath()) == pkgBaseName(srcPkgPath) {
						return importAlias + "." + t.Name()
					}
					return pkgBaseName(t.PkgPath()) + "." + t.Name()
				}
				if t.Kind() == reflect.Interface {
					return "any"
				}
				return t.String()
			}
			mapTypeStr := "map[" + formatType(kt) + "]" + formatType(vt)
			fmt.Fprintf(b, "\tvar arg%[1]d %s\n", i, mapTypeStr)
			fmt.Fprintf(b, "\tswitch v := a%[1]d.(type) {\n", i)
			fmt.Fprintf(b, "\tcase *data.AnyValue:\n\t\targ%[1]d = v.Value.(%s)\n", i, mapTypeStr)
			fmt.Fprintf(b, "\tdefault:\n\t\treturn nil, data.NewErrorThrow(nil, errors.New(\"%s.%s 参数类型不支持, index: %d\"))\n\t}\n", typeName, m.Name, i)
			continue
		case reflect.String:
			fmt.Fprintf(b, "\targ%d := a%d.(*data.StringValue).AsString()\n", i, i)
		case reflect.Int:
			fmt.Fprintf(b, "\targ%d, err := a%d.(*data.IntValue).AsInt()\n\tif err != nil { return nil, data.NewErrorThrow(nil, err) }\n", i, i)
		case reflect.Uint, reflect.Uint8, reflect.Int8, reflect.Uint16, reflect.Uint32, reflect.Int32, reflect.Uint64:
			fmt.Fprintf(b, "\targ%[1]dInt, err := a%[1]d.(*data.IntValue).AsInt()\n\tif err != nil { return nil, data.NewErrorThrow(nil, err) }\n\targ%[1]d := %s(arg%[1]dInt)\n", i, base.Kind().String())
			// 注意：Go 语法中不能直接用 Kind() 作为类型名，此分支仅用于快速断言基础 uint，具名 alias 会在上面的具名基础类型分支处理
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
			// 计算元素类型字符串：使用元素自身包路径与包短名组合，避免误用 srcPkgPath（如 redis 包）
			elemRaw := base.Elem()
			elem := elemRaw
			ptrPrefix := ""
			for elem.Kind() == reflect.Ptr {
				ptrPrefix += "*"
				elem = elem.Elem()
			}
			var elemTypeStr string
			if elem.PkgPath() != "" && elem.Name() != "" {
				if elem.PkgPath() == srcPkgPath {
					// 同包使用导入别名（如 applicationsrc.T）
					elemTypeStr = ptrPrefix + importAlias + "." + elem.Name()
				} else {
					// 跨包使用对方包短名（如 redis.Z）
					elemTypeStr = ptrPrefix + pkgBaseName(elem.PkgPath()) + "." + elem.Name()
				}
			} else {
				// 回退：使用反射字符串并仅替换当前包短名为导入别名
				elemTypeStr = ptrPrefix + elemRaw.String()
				elemTypeStr = strings.ReplaceAll(elemTypeStr, pkgBaseName(srcPkgPath)+".", importAlias+".")
			}
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

			fmt.Fprintf(b, "\targ%[1]d := make(%[2]s, 0)\n", i, sliceTypeStr)
			fmt.Fprintf(b, "\tfor _, v := range a%[1]d.(*data.ArrayValue).Value {\n", i)
			// 在循环内部对每个元素使用 switch 语句处理
			fmt.Fprintf(b, "\t\tswitch elemVal := v.(type) {\n")
			fmt.Fprintf(b, "\t\tcase *data.ClassValue:\n\t\t\tif p, ok := elemVal.Class.(interface{ GetSource() any }); ok { \n\t\t\t\t// 检查 GetSource 返回的类型，如果是指针则解引用\n\t\t\t\tif src := p.GetSource(); src != nil {\n\t\t\t\t\tif ptr, ok := src.(*%s); ok {\n\t\t\t\t\t\targ%d = append(arg%d, *ptr)\n\t\t\t\t\t} else {\n\t\t\t\t\t\targ%d = append(arg%d, src.(%s))\n\t\t\t\t\t}\n\t\t\t\t}\n\t\t\t} else { return nil, data.NewErrorThrow(nil, errors.New(\"%s.%s 参数类型不支持, index: %d\")) }\n", elemTypeStr, i, i, i, i, elemTypeStr, typeName, m.Name, i)
			fmt.Fprintf(b, "\t\tcase *data.ProxyValue:\n\t\t\tif p, ok := elemVal.Class.(interface{ GetSource() any }); ok { if src := p.GetSource(); src != nil { if ptr, ok := src.(*%[2]s); ok { arg%[1]d = append(arg%[1]d, *ptr) } else { arg%[1]d = append(arg%[1]d, src.(%[2]s)) } } } else { return nil, data.NewErrorThrow(nil, errors.New(\"%[3]s.%[4]s 参数类型不支持, index: %[1]d\")) }\n", i, elemTypeStr, typeName, m.Name)
			fmt.Fprintf(b, "\t\tcase *data.AnyValue:\n\t\t\targ%[1]d = append(arg%[1]d, elemVal.Value.(%[2]s))\n", i, elemTypeStr)
			// fmt.Fprintf(b, "\t\tdefault:\n\t\t\targ%[1]d = append(arg%[1]d, elemVal.(%[2]s))\n", i, elemTypeStr)
			fmt.Fprintf(b, "\t\t}\n")
			fmt.Fprintf(b, "\t}\n")
		case reflect.Chan:
			// 通道类型参数：支持 AnyValue 直接断言与 GetSource 提取
			chanTypeStr := t.String()
			// 将同包前缀替换为导入别名
			chanTypeStr = strings.ReplaceAll(chanTypeStr, pkgBaseName(srcPkgPath)+".", importAlias+".")
			fmt.Fprintf(b, "\tvar arg%[1]d %s\n", i, chanTypeStr)
			fmt.Fprintf(b, "\tswitch v := a%[1]d.(type) {\n", i)
			fmt.Fprintf(b, "\tcase *data.ClassValue:\n\t\tif p, ok := v.Class.(interface{ GetSource() any }); ok { if src := p.GetSource(); src != nil { arg%[1]d = src.(%[2]s) } } else { return nil, data.NewErrorThrow(nil, errors.New(\"%[3]s.%[4]s 参数类型不支持, index: %[1]d\")) }\n", i, chanTypeStr, typeName, m.Name)
			fmt.Fprintf(b, "\tcase data.GetSource:\n\t\tif src := v.GetSource(); src != nil { arg%[1]d = src.(%[2]s) } else { return nil, data.NewErrorThrow(nil, errors.New(\"%[3]s.%[4]s 参数类型不支持, index: %[1]d\")) }\n", i, chanTypeStr, typeName, m.Name)
			fmt.Fprintf(b, "\tcase *data.AnyValue:\n\t\targ%[1]d = v.Value.(%[2]s)\n", i, chanTypeStr)
			fmt.Fprintf(b, "\tdefault:\n\t\treturn nil, data.NewErrorThrow(nil, errors.New(\"%s.%s 参数类型不支持, index: %d\"))\n\t}\n", typeName, m.Name, i)
		case reflect.Func:
			// 函数类型：递归格式化签名，按包短名/导入别名输出（修正 redis 包名与别名不一致问题）
			var formatType func(t reflect.Type) string
			formatType = func(t reflect.Type) string {
				switch t.Kind() {
				case reflect.Pointer:
					return "*" + formatType(t.Elem())
				case reflect.Slice:
					return "[]" + formatType(t.Elem())
				case reflect.Array:
					return "[" + strconvItoa(t.Len()) + "]" + formatType(t.Elem())
				case reflect.Map:
					return "map[" + formatType(t.Key()) + "]" + formatType(t.Elem())
				case reflect.Chan:
					return "chan " + formatType(t.Elem())
				case reflect.Func:
					parts := &strings.Builder{}
					parts.WriteString("func(")
					for i := 0; i < t.NumIn(); i++ {
						if i > 0 {
							parts.WriteString(", ")
						}
						parts.WriteString(formatType(t.In(i)))
					}
					parts.WriteString(")")
					// 返回值
					if t.NumOut() == 1 {
						parts.WriteString(" ")
						parts.WriteString(formatType(t.Out(0)))
					} else if t.NumOut() > 1 {
						parts.WriteString(" (")
						for i := 0; i < t.NumOut(); i++ {
							if i > 0 {
								parts.WriteString(", ")
							}
							parts.WriteString(formatType(t.Out(i)))
						}
						parts.WriteString(")")
					}
					return parts.String()
				default:
					// 命名类型与基础类型/接口
					if t.Name() != "" {
						if t.PkgPath() == "" {
							// 预声明或内置接口（如 error）
							return t.Name()
						}
						if pkgBaseName(t.PkgPath()) == pkgBaseName(srcPkgPath) {
							return importAlias + "." + t.Name()
						}
						return pkgBaseName(t.PkgPath()) + "." + t.Name()
					}
					// 非命名接口（空接口）
					if t.Kind() == reflect.Interface {
						return "any"
					}
					// 基础类型
					return t.String()
				}
			}
			funcTypeStr := formatType(base)
			// 生成形参列表（仅用于签名，不使用参数）
			paramList := make([]string, 0, base.NumIn())
			for pi := 0; pi < base.NumIn(); pi++ {
				pt := base.In(pi)
				ptStr := formatType(pt)
				paramList = append(paramList, fmt.Sprintf("p%d %s", pi, ptStr))
			}
			fmt.Fprintf(b, "\tvar arg%[1]d %s\n", i, funcTypeStr)
			fmt.Fprintf(b, "\tswitch fnv := a%[1]d.(type) {\n", i)
			if base.NumOut() == 1 {
				ot := base.Out(0)
				if ot.Kind() == reflect.Interface && ot.PkgPath() == "" && ot.Name() == "error" {
					// 生成带 error 返回的包装函数：建立上下文并映射参数
					fmt.Fprintf(b, "\tcase *data.FuncValue:\n\t\targ%[1]d = func(%s) error {\n\t\t\tfnCtx := ctx.CreateBaseContext()\n", i, strings.Join(paramList, ", "))
					for pi := 0; pi < base.NumIn(); pi++ {
						pt := base.In(pi)
						pname := fmt.Sprintf("p%d", pi)
						fmt.Fprintf(b, "\t\t\tfnCtx.SetVariableValue(fnv.Value.GetVariables()[%d], data.NewProxyValue(New%sClassFrom(%s), ctx))\n", pi, pt.Name(), pname)
					}
					fmt.Fprintf(b, "\t\t\t_, acl := fnv.Call(fnCtx)\n\t\t\treturn errors.New(acl.AsString())\n\t\t}\n")
				} else {
					fmt.Fprintf(b, "\tcase *data.FuncValue:\n\t\targ%[1]d = func(%s) { fnv.Call(ctx) }\n", i, strings.Join(paramList, ", "))
				}
			} else {
				fmt.Fprintf(b, "\tcase *data.FuncValue:\n\t\targ%[1]d = func(%s) { fnv.Call(ctx) }\n", i, strings.Join(paramList, ", "))
			}
			fmt.Fprintf(b, "\tdefault:\n\t\treturn nil, data.NewErrorThrow(nil, errors.New(\"%s.%s 参数类型不支持, index: %d\"))\n\t}\n", typeName, m.Name, i)
		case reflect.Float32, reflect.Float64:
			fmt.Fprintf(b, "\targ%[1]dF, err := a%[1]d.(*data.IntValue).AsInt()\n\tif err != nil { return nil, data.NewErrorThrow(nil, err) }\n\targ%[1]d := float64(arg%[1]dF)\n", i)
		default:
			// 具名 struct/具名基础类型：支持 ClassValue/AnyValue 双路径（通过 GetSource 接口）
			if base.PkgPath() != "" && base.Name() != "" {
				// 使用 base.String() 获取完整类型，然后替换包路径为导入别名
				typeStr := base.String()
				// 将包路径替换为导入别名（如 application.Service -> applicationsrc.Service）
				if base.PkgPath() == srcPkgPath {
					// 同包类型，使用导入别名
					typeStr = importAlias + "." + base.Name()
				}
				fmt.Fprintf(b, "\tvar arg%[1]d %s\n", i, typeStr)
				fmt.Fprintf(b, "\tswitch v := a%[1]d.(type) {\n", i)
				// 构建复杂的类型断言逻辑
				b.WriteString(fmt.Sprintf("\tcase *data.ClassValue:\n\t\tif p, ok := v.Class.(interface{ GetSource() any }); ok { \n\t\t\t// 检查 GetSource 返回的类型，如果是指针则解引用\n\t\t\tif src := p.GetSource(); src != nil {\n\t\t\t\tif ptr, ok := src.(*%s); ok {\n\t\t\t\t\targ%d = *ptr\n\t\t\t\t} else {\n\t\t\t\t\targ%d = src.(%s)\n\t\t\t\t}\n\t\t\t}\n\t\t} else { return nil, data.NewErrorThrow(nil, errors.New(\"%s.%s 参数类型不支持, index: %d\")) }\n", typeStr, i, i, typeStr, typeName, m.Name, i))
				b.WriteString(fmt.Sprintf("\tcase *data.ProxyValue:\n\t\tif p, ok := v.Class.(interface{ GetSource() any }); ok { if src := p.GetSource(); src != nil { if ptr, ok := src.(*%s); ok { arg%d = *ptr } else { arg%d = src.(%s) } } } else { return nil, data.NewErrorThrow(nil, errors.New(\"%s.%s 参数类型不支持, index: %d\")) }\n", typeStr, i, i, typeStr, typeName, m.Name, i))
				fmt.Fprintf(b, "\tcase *data.AnyValue:\n\t\targ%[1]d = v.Value.(%s)\n", i, typeStr)
				fmt.Fprintf(b, "\tdefault:\n\t\treturn nil, data.NewErrorThrow(nil, errors.New(\"%s.%s 参数类型不支持, index: %d\"))\n\t}\n", typeName, m.Name, i)
			} else {
				fmt.Fprintf(b, "\targ%[1]d := a%[1]d.(*data.AnyValue).Value\n", i)
			}
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
		} else if retT.Kind() == reflect.Struct && retT.PkgPath() != "" && retT.Name() != "" {
			// 返回 struct 值类型，取地址传入代理构造
			retName := retT.Name()
			fmt.Fprintf(b, "\treturn data.NewClassValue(%sNew%sClassFrom(&ret0), ctx), nil\n}\n\n", retCtorPrefix, retName)
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
