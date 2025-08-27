package generator

import (
	"fmt"
	"reflect"
	"sort"
	"strings"
)

// 生成类包装代码（仅依赖已生成的方法）
func buildClassFileBody(srcPkgPath, pkgName, typeName string, methods map[string]reflect.Method, structType reflect.Type, namePrefix string) string {
	b := &strings.Builder{}
	importAlias := pkgName + "src"
	// import 延后输出，先收集依赖

	// 收集导出字段（属性）
	var fields []reflect.StructField
	if structType != nil {
		for i := 0; i < structType.NumField(); i++ {
			field := structType.Field(i)
			if field.PkgPath == "" { // 仅导出字段
				fields = append(fields, field)
			}
		}
	}

	// 统一收集方法名，供后续使用
	names := make([]string, 0, len(methods))
	for n := range methods {
		names = append(names, n)
	}
	sort.Strings(names)

	// 额外导入（跨包字段依赖）递归扫描
	extraImports := make(map[string]string)
	needRuntime := false
	addPkg := func(p string) {
		if p != "" && p != srcPkgPath {
			extraImports[p] = pkgBaseName(p)
		}
	}
	var scanType func(t reflect.Type)
	scanType = func(t reflect.Type) {
		if t == nil {
			return
		}
		switch t.Kind() {
		case reflect.Ptr:
			scanType(t.Elem())
		case reflect.Slice, reflect.Array:
			scanType(t.Elem())
		case reflect.Map:
			scanType(t.Key())
			scanType(t.Elem())
		case reflect.Struct:
			if t.PkgPath() != "" && t.Name() != "" {
				addPkg(t.PkgPath())
			}
		case reflect.Interface:
			if t.PkgPath() != "" && t.Name() != "" {
				addPkg(t.PkgPath())
			}
		default:
			if t.PkgPath() != "" && t.Name() != "" {
				addPkg(t.PkgPath())
			}
		}
	}
	for _, f := range fields {
		ft := f.Type
		scanType(ft)
		// 判断是否需要 runtime（同包接口/指针结构体/值结构体属性）
		if (ft.PkgPath() == "" || ft.PkgPath() == srcPkgPath) || (ft.Kind() == reflect.Ptr && ft.Elem() != nil && (ft.Elem().PkgPath() == "" || ft.Elem().PkgPath() == srcPkgPath)) {
			switch {
			case ft.Kind() == reflect.Interface && ft.PkgPath() != "" && ft.Name() != "":
				needRuntime = true
			case ft.Kind() == reflect.Ptr && ft.Elem() != nil && ft.Elem().Kind() == reflect.Struct:
				needRuntime = true
			case ft.Kind() == reflect.Struct && ft.PkgPath() != "" && ft.Name() != "":
				needRuntime = true
			}
		}
	}

	// 输出 import 块（包含源包、origami 与额外依赖）
	b.WriteString("import (\n")
	// 源包别名（用于 From 构造函数形参类型）
	fmt.Fprintf(b, "\t%s %q\n", importAlias, srcPkgPath)
	b.WriteString("\t\"github.com/php-any/origami/data\"\n")
	b.WriteString("\t\"github.com/php-any/origami/node\"\n")
	if needRuntime {
		b.WriteString("\t\"github.com/php-any/origami/runtime\"\n")
	}
	if len(extraImports) > 0 {
		keys := make([]string, 0, len(extraImports))
		for p := range extraImports {
			keys = append(keys, p)
		}
		sort.Strings(keys)
		for _, p := range keys {
			alias := extraImports[p]
			fmt.Fprintf(b, "\t%s %q\n", alias, p)
		}
	}
	b.WriteString(")\n\n")

	// New<Class>Class() - 仅对 struct 生成；接口类型不生成无参构造
	if structType != nil {
		fmt.Fprintf(b, "func New%[1]sClass() data.ClassStmt {\n", typeName)
		fmt.Fprintf(b, "\treturn &%[1]sClass{\n\t\tsource: nil,\n", typeName)
		// 默认构造函数方法（空逻辑）
		fmt.Fprintf(b, "\t\tconstruct: &%sConstructMethod{source: nil},\n", typeName)
		for _, n := range names {
			// 字段名用小写；类型名使用导出方法名，确保形如 StmtCloseMethod
			fmt.Fprintf(b, "\t\t%s: &%s%sMethod{source: nil},\n", lowerFirst(n), typeName, n)
		}
		b.WriteString("\t}\n}\n\n")
	}

	// New<Class>ClassFrom(source alias.Type or *alias.Type)
	star := ""
	if structType != nil {
		star = "*"
	}
	fmt.Fprintf(b, "func New%[1]sClassFrom(source %s%s.%[1]s) data.ClassStmt {\n", typeName, star, importAlias)
	fmt.Fprintf(b, "\treturn &%[1]sClass{\n\t\tsource: source,\n", typeName)
	fmt.Fprintf(b, "\t\tconstruct: &%sConstructMethod{source: source},\n", typeName)
	for _, n := range names {
		fmt.Fprintf(b, "\t\t%s: &%s%sMethod{source: source},\n", lowerFirst(n), typeName, n)
	}
	b.WriteString("\t}\n}\n\n")

	// struct（保存类级别 source，并保留方法代理字段）
	fmt.Fprintf(b, "type %sClass struct {\n\tnode.Node\n\tsource %s%s.%s\n", typeName, star, importAlias, typeName)
	// 构造函数方法（空实现）
	fmt.Fprintf(b, "\tconstruct data.Method\n")
	for _, n := range names {
		fmt.Fprintf(b, "\t%[1]s data.Method\n", lowerFirst(n))
	}
	b.WriteString("}\n\n")

	// interface impls
	if structType != nil {
		// 对 struct 类型：直接复用 New<Class>ClassFrom
		b.WriteString("func (s *")
		b.WriteString(typeName)
		b.WriteString("Class) GetValue(ctx data.Context) (data.GetValue, data.Control) {\n")
		fmt.Fprintf(b, "\treturn data.NewProxyValue(New%[1]sClassFrom(&%[2]s.%[1]s{}), ctx.CreateBaseContext()), nil\n}\n", typeName, importAlias)
	} else {
		fmt.Fprintf(b, "func (s *%[1]sClass) GetValue(_ data.Context) (data.GetValue, data.Control) { clone := *s; return &clone, nil }\n", typeName)
	}
	fmt.Fprintf(b, "func (s *%[1]sClass) GetName() string { return \"%[2]s\\\\%[1]s\" }\n", typeName, namePrefix)
	fmt.Fprintf(b, "func (s *%[1]sClass) GetExtend() *string { return nil }\n", typeName)
	fmt.Fprintf(b, "func (s *%[1]sClass) GetImplements() []string { return nil }\n", typeName)
	fmt.Fprintf(b, "func (s *%[1]sClass) AsString() string { return \"%[2]s{}\" }\n", typeName, typeName)

	// 暴露底层 source，便于参数转换（接口/结构体统一）
	fmt.Fprintf(b, "func (s *%[1]sClass) GetSource() any { return s.source }\n", typeName)

	// GetProperty 实现
	if len(fields) > 0 {
		fmt.Fprintf(b, "func (s *%[1]sClass) GetProperty(name string) (data.Property, bool) {\n\tswitch name {\n", typeName)
		for _, field := range fields {
			fmt.Fprintf(b, "\tcase \"%[1]s\":\n", field.Name)
			// 动态创建属性
			ft := field.Type
			// 跨包字段一律按 AnyValue 处理，避免跨包代理
			isCrossPkg := false
			if ft.Kind() == reflect.Ptr && ft.Elem() != nil {
				isCrossPkg = ft.Elem().PkgPath() != "" && ft.Elem().PkgPath() != srcPkgPath
			} else {
				isCrossPkg = ft.PkgPath() != "" && ft.PkgPath() != srcPkgPath
			}
			if isCrossPkg {
				fmt.Fprintf(b, "\t\treturn node.NewProperty(nil, \"%[1]s\", \"public\", true, data.NewAnyValue(s.source.%[1]s)), true\n", field.Name)
			} else {
				switch {
				case ft.Kind() == reflect.Interface && ft.PkgPath() != "" && ft.Name() != "":
					fmt.Fprintf(b, "\t\treturn node.NewProperty(nil, \"%[1]s\", \"public\", true, data.NewClassValue(New%[2]sClassFrom(s.source.%[1]s), runtime.NewContextToDo())), true\n", field.Name, ft.Name())
				case ft.Kind() == reflect.Ptr && ft.Elem() != nil && ft.Elem().Kind() == reflect.Struct:
					fmt.Fprintf(b, "\t\treturn node.NewProperty(nil, \"%[1]s\", \"public\", true, data.NewClassValue(New%[2]sClassFrom(s.source.%[1]s), runtime.NewContextToDo())), true\n", field.Name, ft.Elem().Name())
				case ft.Kind() == reflect.Struct && ft.PkgPath() != "" && ft.Name() != "":
					fmt.Fprintf(b, "\t\treturn node.NewProperty(nil, \"%[1]s\", \"public\", true, data.NewClassValue(New%[2]sClassFrom(&s.source.%[1]s), runtime.NewContextToDo())), true\n", field.Name, ft.Name())
				default:
					fmt.Fprintf(b, "\t\treturn node.NewProperty(nil, \"%[1]s\", \"public\", true, data.NewAnyValue(s.source.%[1]s)), true\n", field.Name)
				}
			}
		}
		b.WriteString("\t}\n\treturn nil, false\n}\n\n")

		// GetProperties 实现
		fmt.Fprintf(b, "func (s *%[1]sClass) GetProperties() map[string]data.Property {\n\treturn map[string]data.Property{\n", typeName)
		for _, field := range fields {
			// 直接创建属性信息，不调用 GetProperty 获取值
			ft := field.Type
			// 跨包字段一律按 AnyValue 处理，避免跨包代理
			isCrossPkg := false
			if ft.Kind() == reflect.Ptr && ft.Elem() != nil {
				isCrossPkg = ft.Elem().PkgPath() != "" && ft.Elem().PkgPath() != srcPkgPath
			} else {
				isCrossPkg = ft.PkgPath() != "" && ft.PkgPath() != srcPkgPath
			}
			if isCrossPkg {
				fmt.Fprintf(b, "\t\t\"%[1]s\": node.NewProperty(nil, \"%[1]s\", \"public\", true, data.NewAnyValue(nil)),\n", field.Name)
			} else {
				switch {
				case ft.Kind() == reflect.Interface && ft.PkgPath() != "" && ft.Name() != "":
					fmt.Fprintf(b, "\t\t\"%[1]s\": node.NewProperty(nil, \"%[1]s\", \"public\", true, data.NewClassValue(nil, runtime.NewContextToDo())),\n", field.Name)
				case ft.Kind() == reflect.Ptr && ft.Elem() != nil && ft.Elem().Kind() == reflect.Struct:
					fmt.Fprintf(b, "\t\t\"%[1]s\": node.NewProperty(nil, \"%[1]s\", \"public\", true, data.NewClassValue(nil, runtime.NewContextToDo())),\n", field.Name)
				case ft.Kind() == reflect.Struct && ft.PkgPath() != "" && ft.Name() != "":
					fmt.Fprintf(b, "\t\t\"%[1]s\": node.NewProperty(nil, \"%[1]s\", \"public\", true, data.NewClassValue(nil, runtime.NewContextToDo())),\n", field.Name)
				default:
					fmt.Fprintf(b, "\t\t\"%[1]s\": node.NewProperty(nil, \"%[1]s\", \"public\", true, data.NewAnyValue(nil)),\n", field.Name)
				}
			}
		}
		b.WriteString("\t}\n}\n\n")

		// SetProperty 实现（无返回值）
		fmt.Fprintf(b, "func (s *%[1]sClass) SetProperty(name string, value data.Value) {\n\tswitch name {\n", typeName)
		for _, field := range fields {
			fmt.Fprintf(b, "\tcase \"%[1]s\":\n", field.Name)
			ft := field.Type
			// 跨包字段：仅支持 AnyValue 直接断言为实际类型
			isCrossPkg := false
			if ft.Kind() == reflect.Ptr && ft.Elem() != nil {
				isCrossPkg = ft.Elem().PkgPath() != "" && ft.Elem().PkgPath() != srcPkgPath
			} else {
				isCrossPkg = ft.PkgPath() != "" && ft.PkgPath() != srcPkgPath
			}
			if isCrossPkg {
				// 生成类型字符串，并将同包前缀替换为导入别名
				typeStr := ft.String()
				typeStr = strings.ReplaceAll(typeStr, pkgBaseName(srcPkgPath)+".", importAlias+".")
				fmt.Fprintf(b, "\t\tif v, ok := value.(*data.AnyValue); ok { s.source.%[1]s = v.Value.(%s) }\n", field.Name, typeStr)
				continue
			}
			switch {
			case ft.Kind() == reflect.Interface && ft.PkgPath() != "" && ft.Name() != "":
				fullIface := ft.String()
				fullIface = strings.ReplaceAll(fullIface, pkgBaseName(srcPkgPath)+".", importAlias+".")
				fmt.Fprintf(b, "\t\tswitch v := value.(type) {\n")
				fmt.Fprintf(b, "\t\tcase data.GetSource:\n\t\t\tif src := v.GetSource(); src != nil { s.source.%[1]s = src.(%[2]s) }\n", field.Name, fullIface)
				fmt.Fprintf(b, "\t\tcase *data.AnyValue:\n\t\t\ts.source.%[1]s = v.Value.(%[2]s)\n", field.Name, fullIface)
				fmt.Fprintf(b, "\t\t}\n")
			case ft.Kind() == reflect.Ptr && ft.Elem() != nil && ft.Elem().Kind() == reflect.Struct:
				ptrType := "*" + strings.ReplaceAll(ft.Elem().String(), pkgBaseName(srcPkgPath)+".", importAlias+".")
				fmt.Fprintf(b, "\t\tswitch v := value.(type) {\n")
				fmt.Fprintf(b, "\t\tcase data.GetSource:\n\t\t\tif src := v.GetSource(); src != nil { if ptr, ok := src.(%[2]s); ok { s.source.%[1]s = ptr } }\n", field.Name, ptrType)
				fmt.Fprintf(b, "\t\tcase *data.AnyValue:\n\t\t\ts.source.%[1]s = v.Value.(%[2]s)\n", field.Name, ptrType)
				fmt.Fprintf(b, "\t\t}\n")
			case ft.Kind() == reflect.Struct && ft.PkgPath() != "" && ft.Name() != "":
				valType := strings.ReplaceAll(ft.String(), pkgBaseName(srcPkgPath)+".", importAlias+".")
				fmt.Fprintf(b, "\t\tswitch v := value.(type) {\n")
				fmt.Fprintf(b, "\t\tcase data.GetSource:\n\t\t\tif src := v.GetSource(); src != nil { if ptr, ok := src.(*%[2]s); ok { s.source.%[1]s = *ptr } else { s.source.%[1]s = src.(%[2]s) } }\n", field.Name, valType)
				fmt.Fprintf(b, "\t\tcase *data.AnyValue:\n\t\t\ts.source.%[1]s = v.Value.(%[2]s)\n", field.Name, valType)
				fmt.Fprintf(b, "\t\t}\n")
			default:
				// 基础与常见类型
				switch ft.Kind() {
				case reflect.String:
					// 若为同包命名基础类型，需做显式类型转换
					if ft.Name() != "" && (ft.PkgPath() == srcPkgPath || ft.PkgPath() == "") {
						typeStr := strings.ReplaceAll(ft.String(), pkgBaseName(srcPkgPath)+".", importAlias+".")
						fmt.Fprintf(b, "\t\tif sv, ok := value.(*data.StringValue); ok { s.source.%[1]s = %s(sv.AsString()) }\n", field.Name, typeStr)
					} else {
						fmt.Fprintf(b, "\t\tif sv, ok := value.(*data.StringValue); ok { s.source.%[1]s = sv.AsString() }\n", field.Name)
					}
				case reflect.Int:
					if ft.Name() != "" && (ft.PkgPath() == srcPkgPath || ft.PkgPath() == "") {
						typeStr := strings.ReplaceAll(ft.String(), pkgBaseName(srcPkgPath)+".", importAlias+".")
						fmt.Fprintf(b, "\t\tif iv, ok := value.(*data.IntValue); ok { if x, err := iv.AsInt(); err == nil { s.source.%[1]s = %s(x) } }\n", field.Name, typeStr)
					} else {
						fmt.Fprintf(b, "\t\tif iv, ok := value.(*data.IntValue); ok { if x, err := iv.AsInt(); err == nil { s.source.%[1]s = x } }\n", field.Name)
					}
				case reflect.Int64:
					if ft.Name() != "" && (ft.PkgPath() == srcPkgPath || ft.PkgPath() == "") {
						typeStr := strings.ReplaceAll(ft.String(), pkgBaseName(srcPkgPath)+".", importAlias+".")
						fmt.Fprintf(b, "\t\tif iv, ok := value.(*data.IntValue); ok { if x, err := iv.AsInt(); err == nil { s.source.%[1]s = %s(int64(x)) } }\n", field.Name, typeStr)
					} else {
						fmt.Fprintf(b, "\t\tif iv, ok := value.(*data.IntValue); ok { if x, err := iv.AsInt(); err == nil { s.source.%[1]s = int64(x) } }\n", field.Name)
					}
				case reflect.Bool:
					if ft.Name() != "" && (ft.PkgPath() == srcPkgPath || ft.PkgPath() == "") {
						typeStr := strings.ReplaceAll(ft.String(), pkgBaseName(srcPkgPath)+".", importAlias+".")
						fmt.Fprintf(b, "\t\tif bv, ok := value.(*data.BoolValue); ok { if x, err := bv.AsBool(); err == nil { s.source.%[1]s = %s(x) } }\n", field.Name, typeStr)
					} else {
						fmt.Fprintf(b, "\t\tif bv, ok := value.(*data.BoolValue); ok { if x, err := bv.AsBool(); err == nil { s.source.%[1]s = x } }\n", field.Name)
					}
				default:
					fmt.Fprintf(b, "\t\tif v, ok := value.(*data.AnyValue); ok { s.source.%[1]s = v.Value.(%[2]s) }\n", field.Name, strings.ReplaceAll(ft.String(), pkgBaseName(srcPkgPath)+".", importAlias+"."))
				}
			}
		}
		b.WriteString("\t}\n}\n\n")
	} else {
		// 如果没有字段，返回空的实现
		fmt.Fprintf(b, "func (s *%[1]sClass) GetProperty(_ string) (data.Property, bool) { return nil, false }\n", typeName)
		fmt.Fprintf(b, "func (s *%[1]sClass) GetProperties() map[string]data.Property { return nil }\n", typeName)
	}

	// GetMethod
	fmt.Fprintf(b, "func (s *%[1]sClass) GetMethod(name string) (data.Method, bool) {\n\tswitch name {\n", typeName)
	for _, n := range names {
		fmt.Fprintf(b, "\tcase \"%[1]s\": return s.%[1]s, true\n", lowerFirst(n))
	}
	b.WriteString("\t}\n\treturn nil, false\n}\n\n")

	// GetMethods
	fmt.Fprintf(b, "func (s *%[1]sClass) GetMethods() []data.Method { return []data.Method{\n", typeName)
	for _, n := range names {
		fmt.Fprintf(b, "\t\ts.%s,\n", lowerFirst(n))
	}
	b.WriteString("\t}\n}\n\n")
	fmt.Fprintf(b, "func (s *%[1]sClass) GetConstruct() data.Method { return s.construct }\n", typeName)

	// 生成一个空逻辑的构造方法代理，遵循 data.Method 约定
	fmt.Fprintf(b, "\ntype %sConstructMethod struct {\n\tsource %s%s.%s\n}\n\n", typeName, star, importAlias, typeName)
	fmt.Fprintf(b, "func (h *%sConstructMethod) Call(ctx data.Context) (data.GetValue, data.Control) {\n\treturn nil, nil\n}\n\n", typeName)
	fmt.Fprintf(b, "func (h *%sConstructMethod) GetName() string { return \"construct\" }\n", typeName)
	fmt.Fprintf(b, "func (h *%sConstructMethod) GetModifier() data.Modifier { return data.ModifierPublic }\n", typeName)
	fmt.Fprintf(b, "func (h *%sConstructMethod) GetIsStatic() bool { return false }\n", typeName)
	fmt.Fprintf(b, "func (h *%sConstructMethod) GetParams() []data.GetValue { return []data.GetValue{} }\n", typeName)
	fmt.Fprintf(b, "func (h *%sConstructMethod) GetVariables() []data.Variable { return []data.Variable{} }\n", typeName)
	fmt.Fprintf(b, "func (h *%sConstructMethod) GetReturnType() data.Types { return data.NewBaseType(\"void\") }\n", typeName)
	return b.String()
}
