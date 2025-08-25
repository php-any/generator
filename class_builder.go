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
	b.WriteString("import (\n")
	// 源包别名（用于 From 构造函数形参类型）
	fmt.Fprintf(b, "\t%s %q\n", importAlias, srcPkgPath)
	b.WriteString("\t\"github.com/php-any/origami/data\"\n")
	b.WriteString("\t\"github.com/php-any/origami/node\"\n")
	b.WriteString(")\n\n")

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

	// New<Class>Class() - 仅对 struct 生成；接口类型不生成无参构造
	if structType != nil {
		fmt.Fprintf(b, "func New%[1]sClass() data.ClassStmt {\n", typeName)
		fmt.Fprintf(b, "\treturn &%[1]sClass{\n\t\tsource: nil,\n", typeName)
		for _, n := range names {
			// 字段名用小写；类型名使用导出方法名，确保形如 StmtCloseMethod
			fmt.Fprintf(b, "\t\t%s: &%s%sMethod{source: nil},\n", lowerFirst(n), typeName, n)
		}
		for _, f := range fields {
			fmt.Fprintf(b, "\t\tprop%[1]s: node.NewProperty(nil, \"%[1]s\", \"public\", true, nil),\n", f.Name)
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
	for _, n := range names {
		fmt.Fprintf(b, "\t\t%s: &%s%sMethod{source: source},\n", lowerFirst(n), typeName, n)
	}
	for _, f := range fields {
		fmt.Fprintf(b, "\t\tprop%[1]s: node.NewProperty(nil, \"%[1]s\", \"public\", true, data.NewAnyValue(source.%[1]s)),\n", f.Name)
	}
	b.WriteString("\t}\n}\n\n")

	// struct（保存类级别 source，并保留方法代理字段）
	fmt.Fprintf(b, "type %sClass struct {\n\tnode.Node\n\tsource %s%s.%s\n", typeName, star, importAlias, typeName)
	for _, n := range names {
		fmt.Fprintf(b, "\t%[1]s data.Method\n", lowerFirst(n))
	}
	for _, f := range fields {
		fmt.Fprintf(b, "\tprop%[1]s data.Property\n", f.Name)
	}
	b.WriteString("}\n\n")

	// interface impls
	fmt.Fprintf(b, "func (s *%[1]sClass) GetValue(_ data.Context) (data.GetValue, data.Control) { clone := *s; return &clone, nil }\n", typeName)
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
			fmt.Fprintf(b, "\tcase \"%[1]s\": return s.prop%[1]s, true\n", field.Name)
		}
		b.WriteString("\t}\n\treturn nil, false\n}\n\n")

		// GetProperties 实现
		fmt.Fprintf(b, "func (s *%[1]sClass) GetProperties() map[string]data.Property {\n\treturn map[string]data.Property{\n", typeName)
		for _, field := range fields {
			fmt.Fprintf(b, "\t\t\"%[1]s\": s.prop%[1]s,\n", field.Name)
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
	fmt.Fprintf(b, "func (s *%[1]sClass) GetConstruct() data.Method { return nil }\n", typeName)
	return b.String()
}
