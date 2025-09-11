package scr

import (
	"fmt"
	"reflect"
	"strings"
)

// getTypeString 获取类型的字符串表示
func getTypeString(t reflect.Type, fileCache *FileCache) string {
	if t == nil {
		return "interface{}"
	}

	// 具名类型（含包路径与类型名）优先返回“包名.类型名”，以保留别名/定义类型
	if t.PkgPath() != "" && t.Name() != "" {
		pkgName := pkgBaseName(t.PkgPath())
		return pkgName + "." + t.Name()
	}

	switch t.Kind() {
	case reflect.Ptr:
		return "*" + getTypeString(t.Elem(), fileCache)
	case reflect.Slice:
		return "[]" + getTypeString(t.Elem(), fileCache)
	case reflect.Array:
		return fmt.Sprintf("[%d]%s", t.Len(), getTypeString(t.Elem(), fileCache))
	case reflect.Map:
		return fmt.Sprintf("map[%s]%s", getTypeString(t.Key(), fileCache), getTypeString(t.Elem(), fileCache))
	case reflect.Chan:
		return "chan " + getTypeString(t.Elem(), fileCache)
	case reflect.Func:
		// 生成完整的函数类型签名
		// 例如: func(a int, b string) error 或 func(a int, rest ...string) (int, error)
		var b strings.Builder
		b.WriteString("func(")
		numIn := t.NumIn()
		isVar := t.IsVariadic()
		for i := 0; i < numIn; i++ {
			if i > 0 {
				b.WriteString(", ")
			}
			pt := t.In(i)
			if isVar && i == numIn-1 && pt.Kind() == reflect.Slice {
				b.WriteString("...")
				b.WriteString(getTypeString(pt.Elem(), fileCache))
			} else {
				b.WriteString(getTypeString(pt, fileCache))
			}
		}
		b.WriteString(")")
		numOut := t.NumOut()
		if numOut == 1 {
			b.WriteString(" ")
			b.WriteString(getTypeString(t.Out(0), fileCache))
		} else if numOut > 1 {
			b.WriteString(" (")
			for i := 0; i < numOut; i++ {
				if i > 0 {
					b.WriteString(", ")
				}
				b.WriteString(getTypeString(t.Out(i), fileCache))
			}
			b.WriteString(")")
		}
		return b.String()
	case reflect.Struct, reflect.Interface:
		// 空接口特殊处理
		if t.Kind() == reflect.Interface && t.PkgPath() == "" && t.Name() == "" {
			return "interface{}"
		}
		// 匿名空结构体（如 map[string]struct{} 的元素类型）
		if t.Kind() == reflect.Struct && t.PkgPath() == "" && t.Name() == "" && t.NumField() == 0 {
			return "struct{}"
		}
		if t.PkgPath() != "" {
			// 对于标准库类型，直接使用包名
			if isStandardLibrary(t.PkgPath()) {
				return t.PkgPath() + "." + t.Name()
			}
			// 对于第三方包，使用包名
			pkgName := pkgBaseName(t.PkgPath())
			return pkgName + "." + t.Name()
		}
		return t.Name()
	default:
		// 对于具名的基础类型/别名类型（非结构体/接口），需要补全包前缀
		if t.PkgPath() != "" && t.Name() != "" {
			if isStandardLibrary(t.PkgPath()) {
				return t.PkgPath() + "." + t.Name()
			}
			pkgName := pkgBaseName(t.PkgPath())
			return pkgName + "." + t.Name()
		}
		return t.Name()
	}
}

// isContextType 判断是否为 context.Context
func isContextType(t reflect.Type) bool {
	if t == nil {
		return false
	}
	for t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	return t.PkgPath() == "context" && t.Name() == "Context"
}

// isStandardLibrary 检查包路径是否属于标准库
func isStandardLibrary(pkgPath string) bool {
	// 标准库包路径不包含域名，直接以包名开头
	// 例如: "context", "fmt", "time", "net/http" 等
	return !strings.Contains(pkgPath, ".") && !strings.Contains(pkgPath, "/")
}

// lowerFirst 将名称首字母小写
func lowerFirst(s string) string {
	if s == "" {
		return s
	}
	r := []rune(s)
	r[0] = []rune(strings.ToLower(string(r[0])))[0]
	return string(r)
}

// sanitizeIdentifier 将可能与 Go 关键字或内部保留名冲突的标识符做安全化处理
func sanitizeIdentifier(s string) string {
	keywords := map[string]struct{}{
		"break": {}, "default": {}, "func": {}, "interface": {}, "select": {},
		"case": {}, "defer": {}, "go": {}, "map": {}, "struct": {},
		"chan": {}, "else": {}, "goto": {}, "package": {}, "switch": {},
		"const": {}, "fallthrough": {}, "if": {}, "range": {}, "type": {},
		"continue": {}, "for": {}, "import": {}, "return": {}, "var": {},
	}
	// 内部已使用字段名
	reserved := map[string]struct{}{
		"source": {},
	}
	if _, ok := keywords[s]; ok {
		return "_" + s
	}
	if _, ok := reserved[s]; ok {
		return s + "_"
	}
	return s
}

// writeVariadicInterfaceUnpack 生成 ...interface{} 形参的展开逻辑
// paramName: 生成的切片变量名（与最后一个参数名一致）
// index:     该参数在上下文中的索引
func writeVariadicInterfaceUnpack(b *strings.Builder, paramName string, index int) {
	fmt.Fprintf(b, "\t%s := make([]interface{}, 0)\n", paramName)
	fmt.Fprintf(b, "\tv, _ := ctx.GetIndexValue(%d)\n", index)
	fmt.Fprintf(b, "\tif av, ok := v.(*data.ArrayValue); ok {\n")
	fmt.Fprintf(b, "\t\tfor _, avv := range av.Value {\n")
	fmt.Fprintf(b, "\t\t\tswitch vv := avv.(type) {\n")
	fmt.Fprintf(b, "\t\t\tcase data.GetSource:\n\t\t\t\t%s = append(%s, vv.GetSource())\n", paramName, paramName)
	fmt.Fprintf(b, "\t\t\tcase *data.ClassValue:\n\t\t\t\tif p, ok := vv.Class.(data.GetSource); ok { %s = append(%s, p.GetSource()) } else { %s = append(%s, vv) }\n", paramName, paramName, paramName, paramName)
	fmt.Fprintf(b, "\t\t\tcase *data.AnyValue:\n\t\t\t\t%s = append(%s, vv.Value)\n", paramName, paramName)
	fmt.Fprintf(b, "\t\t\tdefault:\n\t\t\t\t%s = append(%s, avv)\n\t\t\t}\n\t\t}\n\t}\n", paramName, paramName)
}

// markTypeImportsUsed 根据实际类型递归标记需要的导入为已使用
func markTypeImportsUsed(t reflect.Type, fileCache *FileCache, srcPkgPath string) {
	if t == nil {
		return
	}
	switch t.Kind() {
	case reflect.Ptr, reflect.Slice, reflect.Array, reflect.Chan:
		markTypeImportsUsed(t.Elem(), fileCache, srcPkgPath)
	case reflect.Map:
		markTypeImportsUsed(t.Key(), fileCache, srcPkgPath)
		markTypeImportsUsed(t.Elem(), fileCache, srcPkgPath)
	}
	if t.PkgPath() != "" {
		if srcPkgPath != "" && t.PkgPath() == srcPkgPath {
			fileCache.MarkImportUsed(srcPkgPath)
		} else {
			fileCache.MarkImportUsed(t.PkgPath())
		}
	}
}

// writeParameterConversion 写入参数类型转换代码
func writeParameterConversion(b *strings.Builder, paramTypes []reflect.Type, paramNames []string, endIdx int, fileCache *FileCache, origPkgName, importAlias string) {
	for i := 0; i < endIdx; i++ {
		// 特殊处理：context.Context 不从 ctx 读取，直接在调用处使用 ctx.GoContext()
		if isContextType(paramTypes[i]) {
			continue
		}
		pName := paramNames[i]
		typeStr := getTypeString(paramTypes[i], fileCache)
		// 统一替换源包名为别名（适配切片/指针/数组/映射等复杂嵌套）
		if origPkgName != "" && strings.Contains(typeStr, origPkgName+".") {
			typeStr = strings.ReplaceAll(typeStr, origPkgName+".", importAlias+".")
		}
		// 根据真实类型自动标记导入（避免硬编码包名）
		markTypeImportsUsed(paramTypes[i], fileCache, "")
		fmt.Fprintf(b, "\t%s, err := utils.ConvertFromIndex[%s](ctx, %d)\n", pName, typeStr, i)
		fmt.Fprintf(b, "\tif err != nil { return nil, data.NewErrorThrow(nil, fmt.Errorf(\"参数转换失败: %%v\", err)) }\n")
	}
}

// writeVariadicParameterHandling 写入可变参数处理代码
func writeVariadicParameterHandling(b *strings.Builder, isVariadic bool, variadicElem reflect.Type, paramNames []string, fileCache *FileCache, origPkgName, importAlias string) {
	if !isVariadic || variadicElem == nil {
		return
	}

	if variadicElem.Kind() == reflect.Interface && variadicElem.PkgPath() == "" && variadicElem.Name() == "" {
		// ...interface{}
		writeVariadicInterfaceUnpack(b, paramNames[len(paramNames)-1], len(paramNames)-1)
		b.WriteString("\n")
	} else {
		// 通用具体类型：统一使用 utils.Convert[T]
		fileCache.MarkImportUsed("github.com/php-any/generator/utils")
		varArgName := paramNames[len(paramNames)-1]
		elemTypeStr := getTypeString(variadicElem, fileCache)
		if origPkgName != "" && strings.Contains(elemTypeStr, origPkgName+".") {
			elemTypeStr = strings.ReplaceAll(elemTypeStr, origPkgName+".", importAlias+".")
		}
		fmt.Fprintf(b, "\t%s := make([]%s, 0)\n", varArgName, elemTypeStr)
		fmt.Fprintf(b, "\tv, _ := ctx.GetIndexValue(%d)\n", len(paramNames)-1)
		fmt.Fprintf(b, "\tif av, ok := v.(*data.ArrayValue); ok {\n")
		fmt.Fprintf(b, "\t\tfor _, avv := range av.Value {\n")
		fmt.Fprintf(b, "\t\t\tif vv, err := utils.Convert[%s](avv); err == nil { %s = append(%s, vv) }\n", elemTypeStr, varArgName, varArgName)
		fmt.Fprintf(b, "\t\t}\n\t}\n")
		b.WriteString("\n")
	}
}
