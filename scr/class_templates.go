package scr

import (
	"fmt"
	"reflect"
	"sort"
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
	writeClassMethods(b, namePrefix, typeName, methods, structType, importAlias, config, fileCache, srcPkgPath)

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
	for keyName, chosenMethod := range buildMethodFieldMapping(methods) {
		safeName := sanitizeIdentifier(keyName)
		fmt.Fprintf(b, "\t\t%s: &%s%sMethod{source: nil},\n", safeName, typeName, chosenMethod)
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
	for keyName, chosenMethod := range buildMethodFieldMapping(methods) {
		safeName := sanitizeIdentifier(keyName)
		fmt.Fprintf(b, "\t\t%s: &%s%sMethod{source: source},\n", safeName, typeName, chosenMethod)
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
	for keyName := range buildMethodFieldMapping(methods) {
		safeName := sanitizeIdentifier(keyName)
		fmt.Fprintf(b, "\t%s data.Method\n", safeName)
	}

	b.WriteString("}\n\n")
}

// writeClassMethods 写入类方法
func writeClassMethods(b *strings.Builder, namePrefix, typeName string, methods map[string]reflect.Method, structType reflect.Type, importAlias string, config *Config, fileCache *FileCache, srcPkgPath string) {
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
	fmt.Fprintf(b, "func (s *%sClass) GetName() string { return \"%s\\\\%s\" }\n", typeName, namePrefix, typeName)

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
	for keyName := range buildMethodFieldMapping(methods) {
		safeName := sanitizeIdentifier(keyName)
		fmt.Fprintf(b, "\tcase \"%s\": return s.%s, true\n", keyName, safeName)
	}
	b.WriteString("\t}\n\treturn nil, false\n}\n\n")
}

// writeGetMethods 写入 GetMethods 方法
func writeGetMethods(b *strings.Builder, typeName string, methods map[string]reflect.Method) {
	fmt.Fprintf(b, "func (s *%sClass) GetMethods() []data.Method {\n", typeName)
	b.WriteString("\treturn []data.Method{\n")
	first := true
	for keyName := range buildMethodFieldMapping(methods) {
		if !first {
			b.WriteString(",\n")
		} else {
			first = false
		}
		safeName := sanitizeIdentifier(keyName)
		fmt.Fprintf(b, "\t\ts.%s", safeName)
	}

	if len(methods) > 0 {
		b.WriteString(",\n")
	}
	b.WriteString("\t}\n}\n\n")
}

// buildMethodFieldMapping 将方法集合映射为 字段键名->选中的方法名
// 规则：
// - 键名：首次出现的 lowerFirst(methodName)
// - 归一键：strings.ToLower(键名)
// - 冲突时选择“尾部连续大写字母计数”更大的方法名（偏向全大写缩写，如 RO）
// - 保留首次出现的键名不变，仅替换映射的目标方法名
func buildMethodFieldMapping(methods map[string]reflect.Method) map[string]string {
	// 收集并排序，确保稳定性
	names := make([]string, 0, len(methods))
	for name := range methods {
		names = append(names, name)
	}
	sort.Strings(names)

	type group struct {
		// 用于字段键名的候选（选择尾部连续大写更少者）
		keyName  string
		keyScore int
		// 用于绑定的方法名（同样选择尾部连续大写更少者，优先更“驼峰”的版本）
		bestName  string
		bestScore int
	}
	byNorm := make(map[string]*group)

	for _, methodName := range names {
		key := lowerFirst(methodName)
		norm := strings.ToLower(key)
		sc := countTrailingUpper(methodName)
		if g, ok := byNorm[norm]; ok {
			// 更新键名：更低的 score 更优（例如 Ro 比 RO 更低）
			if sc < g.keyScore {
				g.keyName = key
				g.keyScore = sc
			}
			// 更新映射的方法名：更低的 score 更优（例如 Ro 比 RO 更低）
			if sc < g.bestScore {
				g.bestName = methodName
				g.bestScore = sc
			}
		} else {
			byNorm[norm] = &group{keyName: key, keyScore: sc, bestName: methodName, bestScore: sc}
		}
	}

	result := make(map[string]string, len(byNorm))
	for _, g := range byNorm {
		result[g.keyName] = g.bestName
	}
	return result
}

// 计算方法名末尾连续大写字母数量
func countTrailingUpper(s string) int {
	cnt := 0
	for i := len(s) - 1; i >= 0; i-- {
		c := s[i]
		if c >= 'A' && c <= 'Z' {
			cnt++
			continue
		}
		break
	}
	return cnt
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
			// 为避免引用未生成的 Class，这里统一回退 AnyValue
			fmt.Fprintf(b, "\t\treturn node.NewProperty(nil, \"%s\", \"public\", true, data.NewAnyValue(s.source.%s)), true\n",
				fieldName, fieldName)
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
			// 为避免引用未生成的 Class，这里统一回退 AnyValue
			fmt.Fprintf(b, "\t\t\"%s\": node.NewProperty(nil, \"%s\", \"public\", true, data.NewAnyValue(nil)),\n",
				fieldName, fieldName)
		} else {
			fmt.Fprintf(b, "\t\t\"%s\": node.NewProperty(nil, \"%s\", \"public\", true, data.NewAnyValue(nil)),\n",
				fieldName, fieldName)
		}
	}
	fmt.Fprintf(b, "\t}\n")
	fmt.Fprintf(b, "}\n\n")

	// SetProperty 方法
	writeSetPropertyMethod(b, typeName, structType, importAlias, config, fileCache)
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

// writeSetPropertyMethod 生成 SetProperty 方法
func writeSetPropertyMethod(b *strings.Builder, typeName string, structType reflect.Type, importAlias string, config *Config, fileCache *FileCache) {
	// 标记使用的导入
	fileCache.MarkImportUsed("github.com/php-any/origami/data")
	fileCache.MarkImportUsed("errors")

	fmt.Fprintf(b, "func (s *%sClass) SetProperty(name string, value data.Value) data.Control {\n", typeName)
	fmt.Fprintf(b, "\tif s.source == nil {\n")
	fmt.Fprintf(b, "\t\treturn data.NewErrorThrow(nil, errors.New(\"无法设置属性，source 为 nil\"))\n")
	fmt.Fprintf(b, "\t}\n\n")

	if structType == nil || structType.Kind() != reflect.Struct || structType.NumField() == 0 {
		// 无字段时返回属性不存在错误
		fmt.Fprintf(b, "\treturn data.NewErrorThrow(nil, errors.New(\"属性不存在: \" + name))\n")
		fmt.Fprintf(b, "}\n\n")
		return
	}

	fmt.Fprintf(b, "\tswitch name {\n")
	for i := 0; i < structType.NumField(); i++ {
		field := structType.Field(i)
		fieldName := field.Name

		// 跳过小写开头的私有字段
		if !IsExportedType(fieldName) {
			continue
		}

		fmt.Fprintf(b, "\tcase \"%s\":\n", fieldName)

		// 获取字段类型
		fieldType := field.Type

		// 标记字段类型的包为已使用
		MarkTypePackageUsed(fieldType, fileCache)

		// 处理指针类型
		if fieldType.Kind() == reflect.Ptr {
			elemType := fieldType.Elem()
			// 按需标记 utils 导入（仅当生成 Convert 时）
			fileCache.MarkImportUsed("github.com/php-any/generator/utils")
			fmt.Fprintf(b, "\t\tval, err := utils.Convert[%s](value)\n", getTypeString(elemType, fileCache))
			fmt.Fprintf(b, "\t\tif err != nil {\n")
			fmt.Fprintf(b, "\t\t\treturn data.NewErrorThrow(nil, err)\n")
			fmt.Fprintf(b, "\t\t}\n")
			fmt.Fprintf(b, "\t\tconverted := new(%s)\n", getTypeString(elemType, fileCache))
			fmt.Fprintf(b, "\t\t*converted = val\n")
			fmt.Fprintf(b, "\t\ts.source.%s = converted\n", fieldName)
		} else {
			// 按需标记 utils 导入（仅当生成 Convert 时）
			fileCache.MarkImportUsed("github.com/php-any/generator/utils")
			fmt.Fprintf(b, "\t\tval, err := utils.Convert[%s](value)\n", getTypeString(fieldType, fileCache))
			fmt.Fprintf(b, "\t\tif err != nil {\n")
			fmt.Fprintf(b, "\t\t\treturn data.NewErrorThrow(nil, err)\n")
			fmt.Fprintf(b, "\t\t}\n")
			fmt.Fprintf(b, "\t\ts.source.%s = val\n", fieldName)
		}
		fmt.Fprintf(b, "\t\treturn nil\n")
	}
	fmt.Fprintf(b, "\tdefault:\n")
	fmt.Fprintf(b, "\t\treturn data.NewErrorThrow(nil, errors.New(\"属性不存在: \" + name))\n")
	fmt.Fprintf(b, "\t}\n")
	fmt.Fprintf(b, "}\n\n")
}

// MarkTypePackageUsed 标记类型使用的包为已使用
func MarkTypePackageUsed(t reflect.Type, fileCache *FileCache) {
	if t == nil || fileCache == nil {
		return
	}

	// 处理指针类型
	for t.Kind() == reflect.Ptr {
		t = t.Elem()
	}

	// 处理复合类型
	switch t.Kind() {
	case reflect.Slice, reflect.Array:
		MarkTypePackageUsed(t.Elem(), fileCache)
	case reflect.Map:
		MarkTypePackageUsed(t.Key(), fileCache)
		MarkTypePackageUsed(t.Elem(), fileCache)
	case reflect.Chan:
		MarkTypePackageUsed(t.Elem(), fileCache)
	case reflect.Func:
		// 处理函数类型的参数和返回值
		numIn := t.NumIn()
		for i := 0; i < numIn; i++ {
			MarkTypePackageUsed(t.In(i), fileCache)
		}
		numOut := t.NumOut()
		for i := 0; i < numOut; i++ {
			MarkTypePackageUsed(t.Out(i), fileCache)
		}
	case reflect.Interface, reflect.Struct:
		// 处理具名类型（包括 time.Duration 等）
		if t.PkgPath() != "" {
			fileCache.MarkImportUsed(t.PkgPath())
		}
	default:
		// 处理其他具名类型
		if t.PkgPath() != "" {
			fileCache.MarkImportUsed(t.PkgPath())
		}
	}
}
