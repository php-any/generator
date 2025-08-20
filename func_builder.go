package generator

import (
	"errors"
	"fmt"
	"path/filepath"
	"reflect"
	"runtime"
	"strconv"
	"strings"
)

// 生成顶级函数包装（如 sql.Open）
func generateTopFunction(fn any, opt GenOptions) error {
	val := reflect.ValueOf(fn)
	typ := val.Type()
	if typ.Kind() != reflect.Func {
		return fmt.Errorf("期待函数，实际: %s", typ.Kind())
	}
	// 优先从返回值/入参中寻找带包路径的命名类型（处理指针）
	var pkgPath string
	for i := 0; i < typ.NumOut() && pkgPath == ""; i++ {
		t := typ.Out(i)
		if t.Kind() == reflect.Pointer {
			t = t.Elem()
		}
		pkgPath = t.PkgPath()
	}
	for i := 0; i < typ.NumIn() && pkgPath == ""; i++ {
		t := typ.In(i)
		if t.Kind() == reflect.Pointer {
			t = t.Elem()
		}
		pkgPath = t.PkgPath()
	}
	// 兜底：从运行时函数名中提取 import 路径
	fullName := ""
	if pkgPath == "" {
		if f := runtime.FuncForPC(val.Pointer()); f != nil {
			fullName = f.Name() // 形如: "database/sql.Open"
			if dot := strings.LastIndex(fullName, "."); dot > 0 {
				candidate := fullName[:dot]
				if idx := strings.Index(candidate, "("); idx >= 0 {
					if slash := strings.LastIndex(candidate, "/"); slash > 0 {
						candidate = candidate[:slash]
					}
				}
				pkgPath = candidate
			}
		}
	}
	if pkgPath == "" {
		return errors.New("无法确定函数所属包")
	}

	pkgName := pkgBaseName(pkgPath)
	if fullName == "" {
		if f := runtime.FuncForPC(val.Pointer()); f != nil {
			fullName = f.Name()
		}
	}
	simpleName := fullName
	if idx := strings.LastIndex(simpleName, "."); idx >= 0 {
		simpleName = simpleName[idx+1:]
	}

	outDir := filepath.Join(opt.OutputRoot, pkgName)
	file := filepath.Join(outDir, strings.ToLower(simpleName)+"_func.go")
	body := buildTopFunctionFileBodyWithNamesAndPC(pkgPath, pkgName, simpleName, fullName, typ, val.Pointer())
	return EmitFile(file, pkgName, body)
}

func buildTopFunctionFileBodyWithNamesAndPC(srcPkgPath, pkgName, funcName, fullName string, fnType reflect.Type, pc uintptr) string {

	numIn := fnType.NumIn()
	numOut := fnType.NumOut()

	// 参数名优先从源码提取
	paramNames := make([]string, 0, numIn)
	if names, ok := tryExtractParamNames(pc, numIn); ok {
		paramNames = names
	} else {
		for i := 0; i < numIn; i++ {
			paramNames = append(paramNames, "param"+strconv.Itoa(i))
		}
	}
	// 构建参数类型映射
	paramKinds := make([]string, 0, numIn)
	paramTypes := make([]reflect.Type, 0, numIn)
	for i := 0; i < numIn; i++ {
		t := fnType.In(i)
		paramKinds = append(paramKinds, typeToKindLabel(t))
		paramTypes = append(paramTypes, t)
	}
	// 返回值信息
	errorType := reflect.TypeOf((*error)(nil)).Elem()
	hasErr := false
	nonErrCount := numOut
	if numOut > 0 && fnType.Out(numOut-1).Implements(errorType) {
		hasErr = true
		nonErrCount = numOut - 1
	}

	// 动态收集需要额外导入的包（仅当会用于类型断言时）
	usedPkgs := make(map[string]bool)
	for i := 0; i < numIn; i++ {
		switch paramKinds[i] {
		case "named_interface":
			collectPkgPaths(paramTypes[i], usedPkgs)
		case "interface", "string", "int", "int64", "bool", "array", "ptr_struct":
			// 不需要类型断言或不使用外部包
			continue
		default:
			collectPkgPaths(paramTypes[i], usedPkgs)
		}
	}
	// 移除源包（已单独导入）
	delete(usedPkgs, srcPkgPath)

	b := &strings.Builder{}
	b.WriteString("import (\n")
	if numIn > 0 {
		b.WriteString("\t\"errors\"\n")
	}
	// 先写入额外依赖包
	for pkg := range usedPkgs {
		fmt.Fprintf(b, "\t\"%s\"\n", pkg)
	}
	// 源包与必需包
	fmt.Fprintf(b, "\t\"%s\"\n", srcPkgPath)
	b.WriteString("\t\"github.com/php-any/origami/data\"\n")
	if numIn > 0 {
		b.WriteString("\t\"github.com/php-any/origami/node\"\n")
	}
	b.WriteString(")\n\n")

	// 生成一个方法结构以适配 data.Method
	typeName := upperFirst(funcName)
	fmt.Fprintf(b, "type %sFunction struct{}\n\n", typeName)
	
	// 生成构造函数
	fmt.Fprintf(b, "func New%sFunction() data.FuncStmt {\n\treturn &%sFunction{}\n}\n\n", typeName, typeName)
	fmt.Fprintf(b, "func (h *%sFunction) Call(ctx data.Context) (data.GetValue, data.Control) {\n\n", typeName)
	// 参数提取
	for i := 0; i < numIn; i++ {
		fmt.Fprintf(b, "\ta%d, ok := ctx.GetIndexValue(%d)\n\tif !ok { return nil, data.NewErrorThrow(nil, errors.New(\"缺少参数, index: %d\")) }\n\n", i, i, i)
	}
	// 参数预处理：按类型严格处理
	for i := 0; i < numIn; i++ {
		base := paramTypes[i]
		if base.Kind() == reflect.Ptr {
			base = base.Elem()
		}
		// *struct 参数：从代理类取出具体 source
		if paramTypes[i].Kind() == reflect.Ptr && paramTypes[i].Elem().Kind() == reflect.Struct {
			clsName := paramTypes[i].Elem().Name()
			fmt.Fprintf(b, "\targ%[1]dClass := a%[1]d.(*data.ClassValue).Class.(*%sClass)\n", i, clsName)
			fmt.Fprintf(b, "\targ%[1]d := arg%[1]dClass.source\n", i)
			continue
		}
		// interface / named interface
		if paramTypes[i].Kind() == reflect.Interface {
			if paramTypes[i].PkgPath() != "" && paramTypes[i].Name() != "" {
				fmt.Fprintf(b, "\targ%[1]d := a%[1]d.(*data.AnyValue).Value.(%s)\n", i, paramTypes[i].String())
			} else {
				fmt.Fprintf(b, "\targ%[1]d := a%[1]d.(*data.AnyValue).Value\n", i)
			}
			continue
		}
		// 其它常见类型
		switch base.Kind() {
		case reflect.String:
			fmt.Fprintf(b, "\targ%d := a%d.(*data.StringValue).AsString()\n", i, i)
		case reflect.Int:
			fmt.Fprintf(b, "\targ%d, err := a%d.(*data.IntValue).AsInt()\n\tif err != nil { return nil, data.NewErrorThrow(nil, err) }\n", i, i)
		case reflect.Int64:
			fmt.Fprintf(b, "\targ%[1]dInt, err := a%[1]d.(*data.IntValue).AsInt()\n\tif err != nil { return nil, data.NewErrorThrow(nil, err) }\n\targ%[1]d := int64(arg%[1]dInt)\n", i)
		case reflect.Bool:
			fmt.Fprintf(b, "\targ%d, err := a%d.(*data.BoolValue).AsBool()\n\tif err != nil { return nil, data.NewErrorThrow(nil, err) }\n", i, i)
		case reflect.Slice:
			fmt.Fprintf(b, "\targ%d := *a%d.(*data.ArrayValue)\n", i, i)
		default:
			fmt.Fprintf(b, "\targ%d := a%d.(*data.InterfaceValue).AsInterface()\n", i, i)
		}
	}
	// 调用并接收返回值
	if numOut > 0 {
		// 构造左值列表
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
	fmt.Fprintf(b, "%s.%s(", pkgBaseName(srcPkgPath), funcName)
	for i := 0; i < numIn; i++ {
		if i > 0 {
			b.WriteString(", ")
		}
		fmt.Fprintf(b, "arg%d", i)
	}
	b.WriteString(")\n")
	// 错误处理
	if hasErr {
		b.WriteString("\tif err != nil {\n\t\treturn nil, data.NewErrorThrow(nil, err)\n\t}\n")
	}
	// 成功返回值封装
	if nonErrCount == 0 {
		b.WriteString("\treturn nil, nil\n}\n\n")
	} else if nonErrCount == 1 {
		b.WriteString("\treturn data.NewAnyValue(ret0), nil\n}\n\n")
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

	fmt.Fprintf(b, "func (h *%sFunction) GetName() string { return \"%s\\\\%s\" }\n", typeName, pkgName, lowerFirst(funcName))
	fmt.Fprintf(b, "func (h *%sFunction) GetModifier() data.Modifier { return data.ModifierPublic }\n", typeName)
	fmt.Fprintf(b, "func (h *%sFunction) GetIsStatic() bool { return true }\n", typeName)
	// GetParams
	b.WriteString("func (h *")
	b.WriteString(typeName)
	b.WriteString("Function) GetParams() []data.GetValue { return []data.GetValue{\n")
	for i := 0; i < numIn; i++ {
		fmt.Fprintf(b, "\t\tnode.NewParameter(nil, %q, %d, nil, nil),\n", paramNames[i], i)
	}
	b.WriteString("\t}\n}\n")
	// GetVariables
	b.WriteString("func (h *")
	b.WriteString(typeName)
	b.WriteString("Function) GetVariables() []data.Variable { return []data.Variable{\n")
	for i := 0; i < numIn; i++ {
		fmt.Fprintf(b, "\t\tnode.NewVariable(nil, %q, %d, nil),\n", paramNames[i], i)
	}
	b.WriteString("\t}\n}\n")
	// 返回类型依旧用 void
	b.WriteString("func (h *")
	b.WriteString(typeName)
	b.WriteString("Function) GetReturnType() data.Types { return data.NewBaseType(\"void\") }\n")
	return b.String()
}

// 辅助：将类型归类为生成时使用的 kind 标签
func typeToKindLabel(t reflect.Type) string {
	if t.Kind() == reflect.String {
		return "string"
	}
	if t.Kind() == reflect.Int {
		return "int"
	}
	if t.Kind() == reflect.Int64 {
		return "int64"
	}
	if t.Kind() == reflect.Bool {
		return "bool"
	}
	if t.PkgPath() == "github.com/php-any/origami/data" && t.Name() == "ArrayValue" {
		return "array"
	}

	if t.Kind() == reflect.Interface {
		// 检查是否为具体的接口类型（如 context.Context）
		if t.PkgPath() != "" && t.Name() != "" {
			return "named_interface"
		}
		return "interface"
	}
	
	// 检查是否为指针类型（如 *struct）
	if t.Kind() == reflect.Ptr && t.Elem().Kind() == reflect.Struct {
		return "ptr_struct"
	}
	
	return "interface"
}
