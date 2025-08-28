package generator

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/php-any/generator/core"
	"github.com/php-any/generator/templates"
	"github.com/php-any/generator/utils"
)

// BaseGenerator 基础生成器
type BaseGenerator struct {
	templateEngine *templates.TemplateEngine
	config         *core.GeneratorConfig
	namingUtils    *utils.NamingUtils
}

// NewBaseGenerator 创建新的基础生成器
func NewBaseGenerator(templateEngine *templates.TemplateEngine, config *core.GeneratorConfig) *BaseGenerator {
	return &BaseGenerator{
		templateEngine: templateEngine,
		config:         config,
		namingUtils:    utils.NewNamingUtils(),
	}
}

// GenerateBaseCode 生成基础代码
func (bg *BaseGenerator) GenerateBaseCode(ctx *core.GeneratorContext) (string, error) {
	if ctx == nil {
		return "", core.NewGeneratorError(core.ErrCodeCodeGeneration, "context is nil", nil)
	}

	// 生成基础包声明
	packageCode := fmt.Sprintf("package main\n\n")

	// 生成基础导入
	importCode := bg.generateBaseImports()

	// 生成基础类型定义
	typeCode := bg.generateBaseTypes()

	// 组合代码
	code := packageCode + importCode + typeCode

	return code, nil
}

// generateBaseImports 生成基础导入
func (bg *BaseGenerator) generateBaseImports() string {
	return `import (
	"fmt"
	"reflect"

	"github.com/php-any/origami/data"
	"github.com/php-any/origami/node"
)

`
}

// generateBaseTypes 生成基础类型
func (bg *BaseGenerator) generateBaseTypes() string {
	return `// Node 基础节点接口
type Node interface {
	GetValue(ctx data.Context) (data.GetValue, data.Control)
}

// BaseNode 基础节点实现
type BaseNode struct {
	name string
}

// NewBaseNode 创建新的基础节点
func NewBaseNode(name string) *BaseNode {
	return &BaseNode{
		name: name,
	}
}

// GetValue 获取值
func (n *BaseNode) GetValue(ctx data.Context) (data.GetValue, data.Control) {
	return data.NewStringValue(n.name), nil
}

// GetName 获取名称
func (n *BaseNode) GetName() string {
	return n.name
}

// SetName 设置名称
func (n *BaseNode) SetName(name string) {
	n.name = name
}

`
}

// GeneratePackageHeader 生成包头部
func (bg *BaseGenerator) GeneratePackageHeader(packageName string) string {
	return fmt.Sprintf("package %s\n\n", packageName)
}

// GenerateImports 生成导入语句
func (bg *BaseGenerator) GenerateImports(imports []core.ImportInfo) string {
	if len(imports) == 0 {
		return ""
	}

	var importCode strings.Builder
	importCode.WriteString("import (\n")

	for _, imp := range imports {
		if imp.Alias != "" {
			importCode.WriteString(fmt.Sprintf("\t%s \"%s\"\n", imp.Alias, imp.Path))
		} else {
			importCode.WriteString(fmt.Sprintf("\t\"%s\"\n", imp.Path))
		}
	}

	importCode.WriteString(")\n\n")
	return importCode.String()
}

// GenerateComment 生成注释
func (bg *BaseGenerator) GenerateComment(comment string) string {
	if comment == "" {
		return ""
	}

	// 将注释按行分割
	lines := strings.Split(comment, "\n")
	var commentCode strings.Builder

	for _, line := range lines {
		if strings.TrimSpace(line) != "" {
			commentCode.WriteString("// " + line + "\n")
		}
	}

	return commentCode.String()
}

// GenerateStructDefinition 生成结构体定义
func (bg *BaseGenerator) GenerateStructDefinition(name string, fields []core.FieldInfo) string {
	if len(fields) == 0 {
		return fmt.Sprintf("type %s struct {\n}\n\n", name)
	}

	var structCode strings.Builder
	structCode.WriteString(fmt.Sprintf("type %s struct {\n", name))

	for _, field := range fields {
		if field.IsExported {
			structCode.WriteString(fmt.Sprintf("\t%s %s\n", field.Name, field.Type.TypeName))
		}
	}

	structCode.WriteString("}\n\n")
	return structCode.String()
}

// GenerateInterfaceDefinition 生成接口定义
func (bg *BaseGenerator) GenerateInterfaceDefinition(name string, methods []core.MethodInfo) string {
	if len(methods) == 0 {
		return fmt.Sprintf("type %s interface {\n}\n\n", name)
	}

	var interfaceCode strings.Builder
	interfaceCode.WriteString(fmt.Sprintf("type %s interface {\n", name))

	for _, method := range methods {
		if method.IsExported {
			// 生成方法签名
			methodSig := bg.generateMethodSignature(method)
			interfaceCode.WriteString(fmt.Sprintf("\t%s\n", methodSig))
		}
	}

	interfaceCode.WriteString("}\n\n")
	return interfaceCode.String()
}

// generateMethodSignature 生成方法签名
func (bg *BaseGenerator) generateMethodSignature(method core.MethodInfo) string {
	var signature strings.Builder
	signature.WriteString(method.Name)
	signature.WriteString("(")

	// 添加参数
	for i, param := range method.Parameters {
		if i > 0 {
			signature.WriteString(", ")
		}
		signature.WriteString(param.Name)
		signature.WriteString(" ")
		signature.WriteString(param.Type.TypeName)
	}

	signature.WriteString(")")

	// 添加返回值
	if len(method.Returns) > 0 {
		if len(method.Returns) == 1 {
			signature.WriteString(" ")
			signature.WriteString(method.Returns[0].TypeName)
		} else {
			signature.WriteString(" (")
			for i, ret := range method.Returns {
				if i > 0 {
					signature.WriteString(", ")
				}
				signature.WriteString(ret.TypeName)
			}
			signature.WriteString(")")
		}
	}

	return signature.String()
}

// GenerateConstructor 生成构造函数
func (bg *BaseGenerator) GenerateConstructor(typeName string, fields []core.FieldInfo) string {
	var constructorCode strings.Builder

	// 生成构造函数
	constructorCode.WriteString(fmt.Sprintf("// New%s 创建新的%s实例\n", typeName, typeName))
	constructorCode.WriteString(fmt.Sprintf("func New%s() *%s {\n", typeName, typeName))
	constructorCode.WriteString(fmt.Sprintf("\treturn &%s{\n", typeName))

	// 添加字段初始化
	for _, field := range fields {
		if field.IsExported {
			constructorCode.WriteString(fmt.Sprintf("\t\t%s: %s,\n", field.Name, bg.generateDefaultValue(field.Type)))
		}
	}

	constructorCode.WriteString("\t}\n")
	constructorCode.WriteString("}\n\n")

	return constructorCode.String()
}

// generateDefaultValue 生成默认值
func (bg *BaseGenerator) generateDefaultValue(fieldType *core.TypeInfo) string {
	switch fieldType.Type.Kind() {
	case reflect.Bool:
		return "false"
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return "0"
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return "0"
	case reflect.Float32, reflect.Float64:
		return "0.0"
	case reflect.String:
		return "\"\""
	case reflect.Ptr:
		return "nil"
	case reflect.Slice:
		return "nil"
	case reflect.Map:
		return "nil"
	case reflect.Chan:
		return "nil"
	case reflect.Interface:
		return "nil"
	default:
		return "nil"
	}
}

// GenerateGetters 生成getter方法
func (bg *BaseGenerator) GenerateGetters(typeName string, fields []core.FieldInfo) string {
	var gettersCode strings.Builder

	for _, field := range fields {
		if field.IsExported {
			getterName := "Get" + bg.namingUtils.UpperFirst(field.Name)
			gettersCode.WriteString(fmt.Sprintf("// %s 获取%s字段\n", getterName, field.Name))
			gettersCode.WriteString(fmt.Sprintf("func (t *%s) %s() %s {\n", typeName, getterName, field.Type.TypeName))
			gettersCode.WriteString(fmt.Sprintf("\treturn t.%s\n", field.Name))
			gettersCode.WriteString("}\n\n")
		}
	}

	return gettersCode.String()
}

// GenerateSetters 生成setter方法
func (bg *BaseGenerator) GenerateSetters(typeName string, fields []core.FieldInfo) string {
	var settersCode strings.Builder

	for _, field := range fields {
		if field.IsExported {
			setterName := "Set" + bg.namingUtils.UpperFirst(field.Name)
			settersCode.WriteString(fmt.Sprintf("// %s 设置%s字段\n", setterName, field.Name))
			settersCode.WriteString(fmt.Sprintf("func (t *%s) %s(value %s) {\n", typeName, setterName, field.Type.TypeName))
			settersCode.WriteString(fmt.Sprintf("\tt.%s = value\n", field.Name))
			settersCode.WriteString("}\n\n")
		}
	}

	return settersCode.String()
}

// GenerateStringMethod 生成String方法
func (bg *BaseGenerator) GenerateStringMethod(typeName string, fields []core.FieldInfo) string {
	var stringMethodCode strings.Builder

	stringMethodCode.WriteString(fmt.Sprintf("// String 返回%s的字符串表示\n", typeName))
	stringMethodCode.WriteString(fmt.Sprintf("func (t *%s) String() string {\n", typeName))
	stringMethodCode.WriteString(fmt.Sprintf("\treturn fmt.Sprintf(\"%s{", typeName))

	// 添加字段格式化
	for i, field := range fields {
		if field.IsExported {
			if i > 0 {
				stringMethodCode.WriteString(" ")
			}
			stringMethodCode.WriteString(fmt.Sprintf("%s:%%v", field.Name))
		}
	}

	stringMethodCode.WriteString("}\", ")

	// 添加字段值
	for i, field := range fields {
		if field.IsExported {
			if i > 0 {
				stringMethodCode.WriteString(", ")
			}
			stringMethodCode.WriteString(fmt.Sprintf("t.%s", field.Name))
		}
	}

	stringMethodCode.WriteString(")\n")
	stringMethodCode.WriteString("}\n\n")

	return stringMethodCode.String()
}

// GenerateEqualsMethod 生成Equals方法
func (bg *BaseGenerator) GenerateEqualsMethod(typeName string, fields []core.FieldInfo) string {
	var equalsMethodCode strings.Builder

	equalsMethodCode.WriteString(fmt.Sprintf("// Equals 检查两个%s是否相等\n", typeName))
	equalsMethodCode.WriteString(fmt.Sprintf("func (t *%s) Equals(other *%s) bool {\n", typeName, typeName))
	equalsMethodCode.WriteString("\tif other == nil {\n")
	equalsMethodCode.WriteString("\t\treturn false\n")
	equalsMethodCode.WriteString("\t}\n\n")

	// 添加字段比较
	for _, field := range fields {
		if field.IsExported {
			equalsMethodCode.WriteString(fmt.Sprintf("\tif t.%s != other.%s {\n", field.Name, field.Name))
			equalsMethodCode.WriteString(fmt.Sprintf("\t\treturn false\n"))
			equalsMethodCode.WriteString(fmt.Sprintf("\t}\n"))
		}
	}

	equalsMethodCode.WriteString("\n\treturn true\n")
	equalsMethodCode.WriteString("}\n\n")

	return equalsMethodCode.String()
}

// GenerateCloneMethod 生成Clone方法
func (bg *BaseGenerator) GenerateCloneMethod(typeName string, fields []core.FieldInfo) string {
	var cloneMethodCode strings.Builder

	cloneMethodCode.WriteString(fmt.Sprintf("// Clone 创建%s的副本\n", typeName))
	cloneMethodCode.WriteString(fmt.Sprintf("func (t *%s) Clone() *%s {\n", typeName, typeName))
	cloneMethodCode.WriteString(fmt.Sprintf("\tif t == nil {\n"))
	cloneMethodCode.WriteString(fmt.Sprintf("\t\treturn nil\n"))
	cloneMethodCode.WriteString(fmt.Sprintf("\t}\n\n"))

	cloneMethodCode.WriteString(fmt.Sprintf("\tclone := &%s{\n", typeName))

	// 添加字段复制
	for _, field := range fields {
		if field.IsExported {
			cloneMethodCode.WriteString(fmt.Sprintf("\t\t%s: t.%s,\n", field.Name, field.Name))
		}
	}

	cloneMethodCode.WriteString("\t}\n\n")
	cloneMethodCode.WriteString("\treturn clone\n")
	cloneMethodCode.WriteString("}\n\n")

	return cloneMethodCode.String()
}
