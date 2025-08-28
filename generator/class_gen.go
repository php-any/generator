package generator

import (
	"fmt"
	"strings"

	"github.com/php-any/generator/core"
	"github.com/php-any/generator/templates"
	"github.com/php-any/generator/utils"
)

// ClassGenerator 类生成器
type ClassGenerator struct {
	templateEngine *templates.TemplateEngine
	config         *core.GeneratorConfig
	namingUtils    *utils.NamingUtils
}

// NewClassGenerator 创建新的类生成器
func NewClassGenerator(templateEngine *templates.TemplateEngine, config *core.GeneratorConfig) *ClassGenerator {
	return &ClassGenerator{
		templateEngine: templateEngine,
		config:         config,
		namingUtils:    utils.NewNamingUtils(),
	}
}

// GenerateClass 生成类代码（含属性与方法声明）
func (cg *ClassGenerator) GenerateClass(cls *core.TypeInfo, packageName string) (string, error) {
	if cls == nil {
		return "", core.NewGeneratorError(core.ErrCodeCodeGeneration, "class info is nil", nil)
	}

	data := &core.TemplateData{
		PackageName: packageName,
		ClassName:   cls.TypeName,
		Fields:      cg.convertFields(cls.Fields),
		Methods:     cg.convertMethods(cls, cls.Methods),
	}

	code, err := cg.templateEngine.GenerateClass(data)
	if err != nil {
		return "", fmt.Errorf("failed to generate class: %w", err)
	}
	return code, nil
}

// GenerateMethodsFiles 生成每个方法的独立文件内容
func (cg *ClassGenerator) GenerateMethodsFiles(cls *core.TypeInfo, packageName string) (map[string]string, error) {
	result := make(map[string]string)
	for _, m := range cls.Methods {
		// method.tmpl 需要 MethodName，通过 Methods 列表或专用生成器
		// 这里复用模板引擎的 GenerateMethod，单个方法一个文件
		methodData := &core.TemplateData{
			PackageName: packageName,
			ClassName:   cls.TypeName,
		}
		// 为 GenerateMethod 组装简化数据（MethodName/Parameters/Returns）
		// 由于 TemplateData 没有直接字段 MethodName，使用 Methods[0] 传递
		methodData.Methods = []core.MethodTemplateData{
			{
				Name:       m.Name,
				ClassName:  cls.TypeName,
				Parameters: cg.convertParams(m.Parameters),
				Returns:    cg.convertReturns(m.Returns),
				IsVariadic: m.IsVariadic,
				IsExported: m.IsExported,
			},
		}
		code, err := cg.templateEngine.GenerateMethod(methodData)
		if err != nil {
			return nil, fmt.Errorf("failed to generate method %s: %w", m.Name, err)
		}
		fileName := cg.buildMethodFileName(packageName, cls.TypeName, m.Name)
		result[fileName] = code
	}
	return result, nil
}

func (cg *ClassGenerator) convertFields(fields []core.FieldInfo) []core.FieldTemplateData {
	var out []core.FieldTemplateData
	for _, f := range fields {
		out = append(out, core.FieldTemplateData{
			Name:       f.Name,
			Type:       f.Type.TypeName,
			IsExported: f.IsExported,
			Tag:        string(f.Tag),
		})
	}
	return out
}

func (cg *ClassGenerator) convertMethods(cls *core.TypeInfo, methods []core.MethodInfo) []core.MethodTemplateData {
	var out []core.MethodTemplateData
	for _, m := range methods {
		out = append(out, core.MethodTemplateData{
			Name:       m.Name,
			ClassName:  cls.TypeName,
			Parameters: cg.convertParams(m.Parameters),
			Returns:    cg.convertReturns(m.Returns),
			IsVariadic: m.IsVariadic,
			IsExported: m.IsExported,
		})
	}
	return out
}

func (cg *ClassGenerator) convertParams(params []core.ParameterInfo) []core.ParameterTemplateData {
	var out []core.ParameterTemplateData
	for i, p := range params {
		out = append(out, core.ParameterTemplateData{
			Name:  p.Name,
			Type:  p.Type.TypeName,
			Index: i,
		})
	}
	return out
}

func (cg *ClassGenerator) convertReturns(returns []core.TypeInfo) []core.ReturnTemplateData {
	var out []core.ReturnTemplateData
	for i := range returns {
		out = append(out, core.ReturnTemplateData{
			Type:  returns[i].TypeName,
			Index: i,
		})
	}
	return out
}

func (cg *ClassGenerator) buildMethodFileName(pkg, className, methodName string) string {
	var b strings.Builder
	b.WriteString(pkg)
	b.WriteString("_")
	b.WriteString(strings.ToLower(className))
	b.WriteString("_")
	b.WriteString(strings.ToLower(methodName))
	b.WriteString("_method.go")
	return b.String()
}
