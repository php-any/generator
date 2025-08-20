package generator

import (
	"fmt"
	"reflect"
	"strings"
)

// 生成类似 demo/log/log_class.go 的类封装
func GenerateClassLike(pkg string, typeName string, methods map[string]reflect.Type) string {
	// 构造方法列表
	body := &strings.Builder{}
	body.WriteString("import (\n")
	body.WriteString("\t\"github.com/php-any/origami/data\"\n")
	body.WriteString("\t\"github.com/php-any/origami/node\"\n")
	body.WriteString(")\n\n")

	fmt.Fprintf(body, "func New%[1]sClass() data.ClassStmt {\n", typeName)
	body.WriteString("\treturn &")
	body.WriteString(typeName)
	body.WriteString("Class{}\n}")

	// 声明类结构体
	fmt.Fprintf(body, "\ntype %[1]sClass struct {\n\tnode.Node\n", typeName)
	for name := range methods {
		fmt.Fprintf(body, "\t%[1]s data.Method\n", name)
	}
	body.WriteString("}\n\n")

	body.WriteString("func (s *")
	body.WriteString(typeName)
	body.WriteString(") GetValue(_ data.Context) (data.GetValue, data.Control) {\n")
	body.WriteString("\tclone := *s\n\treturn &clone, nil\n}\n\n")

	fmt.Fprintf(body, "func (s *%[1]sClass) GetName() string { return \"%[1]s\" }\n", typeName)
	body.WriteString("func (s *")
	body.WriteString(typeName)
	body.WriteString(") GetExtend() *string { return nil }\n")
	body.WriteString("func (s *")
	body.WriteString(typeName)
	body.WriteString(") GetImplements() []string { return nil }\n")
	body.WriteString("func (s *")
	body.WriteString(typeName)
	body.WriteString(") GetProperty(_ string) (data.Property, bool) { return nil, false }\n")
	body.WriteString("func (s *")
	body.WriteString(typeName)
	body.WriteString(") GetProperties() map[string]data.Property { return nil }\n")

	body.WriteString("func (s *")
	body.WriteString(typeName)
	body.WriteString(") GetMethod(name string) (data.Method, bool) {\n\tswitch name {\n")
	for name := range methods {
		fmt.Fprintf(body, "\tcase \"%[1]s\": return s.%[1]s, true\n", name)
	}
	body.WriteString("\t}\n\treturn nil, false\n}\n\n")

	body.WriteString("func (s *")
	body.WriteString(typeName)
	body.WriteString(") GetMethods() []data.Method {\n\treturn []data.Method{\n")
	first := true
	for name := range methods {
		if !first {
			body.WriteString(",\n")
		} else {
			first = false
		}
		fmt.Fprintf(body, "\t\ts.%s,", name)
	}
	body.WriteString("\n\t}\n}\n\n")

	body.WriteString("func (s *")
	body.WriteString(typeName)
	body.WriteString(") GetConstruct() data.Method { return nil }\n")

	return body.String()
}
