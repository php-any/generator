package scr

import (
	"fmt"
	"reflect"
	"strings"
)

// buildClassFileBody 构建类文件内容
func buildClassFileBody(srcPkgPath, pkgName, typeName string, methods map[string]reflect.Method, structType reflect.Type, namePrefix string, fileCache *FileCache, config *Config) string {
	b := &strings.Builder{}
	importAlias := pkgName + "src"

	// 收集导入
	collectClassImports(srcPkgPath, pkgName, methods, structType, fileCache, config)

	// 生成构造函数
	writeClassConstructor(b, typeName, importAlias, methods, structType, fileCache, srcPkgPath)

	// 生成类结构体
	writeClassStruct(b, typeName, methods, importAlias, structType, fileCache, srcPkgPath)

	// 生成类方法
	writeClassMethods(b, typeName, methods, structType, importAlias, config, fileCache, srcPkgPath)

	// 在文件开头写入导入（在代码生成完成后，但需要插入到文件开头）
	content := b.String()
	b.Reset()
	writeImportsFromCache(b, fileCache)
	b.WriteString(content)

	return b.String()
}

// writeClassConstructor 写入类构造函数
func writeClassConstructor(b *strings.Builder, typeName, importAlias string, methods map[string]reflect.Method, structType reflect.Type, fileCache *FileCache, srcPkgPath string) {
	// 标记使用的导入
	fileCache.MarkImportUsed("github.com/php-any/origami/data")
	fileCache.MarkImportUsed("github.com/php-any/origami/node")
	if srcPkgPath != "" {
		fileCache.MarkImportUsed(srcPkgPath)
	}

	// NewXxxClass() 构造函数
	fmt.Fprintf(b, "func New%sClass() data.ClassStmt {\n", typeName)
	fmt.Fprintf(b, "\treturn &%sClass{\n", typeName)
	fmt.Fprintf(b, "\t\tsource: nil,\n")
	for methodName := range methods {
		lowerMethodName := lowerFirst(methodName)
		fmt.Fprintf(b, "\t\t%s: &%s%sMethod{source: nil},\n", lowerMethodName, typeName, methodName)
	}
	fmt.Fprintf(b, "\t}\n")
	fmt.Fprintf(b, "}\n\n")

	// NewXxxClassFrom() 构造函数
	if structType.Kind() == reflect.Interface {
		fmt.Fprintf(b, "func New%sClassFrom(source %s.%s) data.ClassStmt {\n", typeName, importAlias, typeName)
	} else {
		fmt.Fprintf(b, "func New%sClassFrom(source *%s.%s) data.ClassStmt {\n", typeName, importAlias, typeName)
	}
	fmt.Fprintf(b, "\treturn &%sClass{\n", typeName)
	fmt.Fprintf(b, "\t\tsource: source,\n")
	for methodName := range methods {
		lowerMethodName := lowerFirst(methodName)
		fmt.Fprintf(b, "\t\t%s: &%s%sMethod{source: source},\n", lowerMethodName, typeName, methodName)
	}
	fmt.Fprintf(b, "\t}\n")
	fmt.Fprintf(b, "}\n\n")
}

// writeClassStruct 写入类结构体
func writeClassStruct(b *strings.Builder, typeName string, methods map[string]reflect.Method, importAlias string, structType reflect.Type, fileCache *FileCache, srcPkgPath string) {
	// 标记使用的导入
	fileCache.MarkImportUsed("github.com/php-any/origami/node")
	if srcPkgPath != "" {
		fileCache.MarkImportUsed(srcPkgPath)
	}

	fmt.Fprintf(b, "type %sClass struct {\n", typeName)
	b.WriteString("\tnode.Node\n")

	// 根据类型决定 source 字段类型
	if structType.Kind() == reflect.Interface {
		fmt.Fprintf(b, "\tsource %s.%s\n", importAlias, typeName)
	} else {
		fmt.Fprintf(b, "\tsource *%s.%s\n", importAlias, typeName)
	}

	// 添加方法字段（小驼峰命名）
	for methodName := range methods {
		lowerMethodName := lowerFirst(methodName)
		fmt.Fprintf(b, "\t%s data.Method\n", lowerMethodName)
	}

	b.WriteString("}\n\n")
}

// writeClassMethods 写入类方法
func writeClassMethods(b *strings.Builder, typeName string, methods map[string]reflect.Method, structType reflect.Type, importAlias string, config *Config, fileCache *FileCache, srcPkgPath string) {
	// 标记使用的导入
	fileCache.MarkImportUsed("github.com/php-any/origami/data")
	if srcPkgPath != "" {
		fileCache.MarkImportUsed(srcPkgPath)
	}

	// GetValue 方法
	fmt.Fprintf(b, "func (s *%sClass) GetValue(ctx data.Context) (data.GetValue, data.Control) {\n", typeName)
	if structType.Kind() == reflect.Interface {
		fmt.Fprintf(b, "\treturn data.NewProxyValue(New%sClassFrom(nil), ctx.CreateBaseContext()), nil\n", typeName)
	} else {
		fmt.Fprintf(b, "\treturn data.NewProxyValue(New%sClassFrom(&%s.%s{}), ctx.CreateBaseContext()), nil\n", typeName, importAlias, typeName)
	}
	b.WriteString("}\n\n")

	// GetName 方法
	fmt.Fprintf(b, "func (s *%sClass) GetName() string { return \"%s\" }\n", typeName, typeName)

	// GetExtend 方法
	fmt.Fprintf(b, "func (s *%sClass) GetExtend() *string { return nil }\n", typeName)

	// GetImplements 方法
	fmt.Fprintf(b, "func (s *%sClass) GetImplements() []string { return nil }\n", typeName)

	// AsString 方法
	fmt.Fprintf(b, "func (s *%sClass) AsString() string { return \"%s{}\" }\n", typeName, typeName)

	// GetSource 方法
	fmt.Fprintf(b, "func (s *%sClass) GetSource() any { return s.source }\n", typeName)

	// GetMethod 方法
	writeGetMethod(b, typeName, methods)

	// GetMethods 方法
	writeGetMethods(b, typeName, methods)

	// GetConstruct 方法
	fmt.Fprintf(b, "func (s *%sClass) GetConstruct() data.Method { return nil }\n\n", typeName)

	// GetProperty 和 GetProperties 方法
	writePropertyMethods(b, typeName, structType, importAlias, config, fileCache)
}

// writeGetMethod 写入 GetMethod 方法
func writeGetMethod(b *strings.Builder, typeName string, methods map[string]reflect.Method) {
	fmt.Fprintf(b, "func (s *%sClass) GetMethod(name string) (data.Method, bool) {\n", typeName)
	b.WriteString("\tswitch name {\n")
	for methodName := range methods {
		lowerMethodName := lowerFirst(methodName)
		fmt.Fprintf(b, "\tcase \"%s\": return s.%s, true\n", lowerMethodName, lowerMethodName)
	}
	b.WriteString("\t}\n\treturn nil, false\n}\n\n")
}

// writeGetMethods 写入 GetMethods 方法
func writeGetMethods(b *strings.Builder, typeName string, methods map[string]reflect.Method) {
	fmt.Fprintf(b, "func (s *%sClass) GetMethods() []data.Method {\n", typeName)
	b.WriteString("\treturn []data.Method{\n")
	first := true
	for methodName := range methods {
		if !first {
			b.WriteString(",\n")
		} else {
			first = false
		}
		lowerMethodName := lowerFirst(methodName)
		fmt.Fprintf(b, "\t\ts.%s", lowerMethodName)
	}

	if len(methods) > 0 {
		b.WriteString(",\n")
	}
	b.WriteString("\t}\n}\n\n")
}

// writePropertyMethods 写入属性相关方法
func writePropertyMethods(b *strings.Builder, typeName string, structType reflect.Type, importAlias string, config *Config, fileCache *FileCache) {
	// 标记使用的导入
	fileCache.MarkImportUsed("github.com/php-any/origami/data")
	fileCache.MarkImportUsed("github.com/php-any/origami/node")

	if structType == nil || structType.Kind() != reflect.Struct || structType.NumField() == 0 {
		// 无字段时返回空实现
		fmt.Fprintf(b, "func (s *%sClass) GetProperty(name string) (data.Property, bool) {\n", typeName)
		fmt.Fprintf(b, "\treturn nil, false\n")
		fmt.Fprintf(b, "}\n\n")

		fmt.Fprintf(b, "func (s *%sClass) GetProperties() map[string]data.Property {\n", typeName)
		fmt.Fprintf(b, "\treturn map[string]data.Property{}\n")
		fmt.Fprintf(b, "}\n\n")
		return
	}

	// GetProperty 方法
	fmt.Fprintf(b, "func (s *%sClass) GetProperty(name string) (data.Property, bool) {\n", typeName)
	fmt.Fprintf(b, "\tswitch name {\n")
	for i := 0; i < structType.NumField(); i++ {
		field := structType.Field(i)
		fieldName := field.Name

		// 跳过小写开头的私有字段
		if !IsExportedType(fieldName) {
			continue
		}

		fmt.Fprintf(b, "\tcase \"%s\":\n", fieldName)

		// 根据字段类型生成不同的属性值
		if config != nil && isBlacklistedType(field.Type, config) {
			// 黑名单类型使用 AnyValue
			fmt.Fprintf(b, "\t\treturn node.NewProperty(nil, \"%s\", \"public\", true, data.NewAnyValue(s.source.%s)), true\n",
				fieldName, fieldName)
		} else if isStructType(field.Type) {
			fmt.Fprintf(b, "\t\treturn node.NewProperty(nil, \"%s\", \"public\", true, data.NewClassValue(New%sClassFrom(s.source.%s), runtime.NewContextToDo())), true\n",
				fieldName, getStructTypeName(field.Type), fieldName)
		} else {
			fmt.Fprintf(b, "\t\treturn node.NewProperty(nil, \"%s\", \"public\", true, data.NewAnyValue(s.source.%s)), true\n",
				fieldName, fieldName)
		}
	}
	fmt.Fprintf(b, "\t}\n")
	fmt.Fprintf(b, "\treturn nil, false\n")
	fmt.Fprintf(b, "}\n\n")

	// GetProperties 方法
	fmt.Fprintf(b, "func (s *%sClass) GetProperties() map[string]data.Property {\n", typeName)
	fmt.Fprintf(b, "\treturn map[string]data.Property{\n")
	for i := 0; i < structType.NumField(); i++ {
		field := structType.Field(i)
		fieldName := field.Name

		// 跳过小写开头的私有字段
		if !IsExportedType(fieldName) {
			continue
		}

		if config != nil && isBlacklistedType(field.Type, config) {
			// 黑名单类型使用 AnyValue
			fmt.Fprintf(b, "\t\t\"%s\": node.NewProperty(nil, \"%s\", \"public\", true, data.NewAnyValue(nil)),\n",
				fieldName, fieldName)
		} else if isStructType(field.Type) {
			fmt.Fprintf(b, "\t\t\"%s\": node.NewProperty(nil, \"%s\", \"public\", true, data.NewClassValue(nil, runtime.NewContextToDo())),\n",
				fieldName, fieldName)
		} else {
			fmt.Fprintf(b, "\t\t\"%s\": node.NewProperty(nil, \"%s\", \"public\", true, data.NewAnyValue(nil)),\n",
				fieldName, fieldName)
		}
	}
	fmt.Fprintf(b, "\t}\n")
	fmt.Fprintf(b, "}\n\n")
}

// getStructTypeName 获取结构体类型名称
func getStructTypeName(t reflect.Type) string {
	for t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	return t.Name()
}

// isBlacklistedType 检查类型是否在黑名单中
func isBlacklistedType(t reflect.Type, config *Config) bool {
	if t == nil {
		return false
	}

	// 获取类型的包路径
	pkgPath := t.PkgPath()
	if pkgPath == "" {
		return false
	}

	// 检查是否在黑名单中
	for _, blacklistedPkg := range config.Blacklist.Packages {
		if pkgPath == blacklistedPkg {
			return true
		}
	}

	return false
}
