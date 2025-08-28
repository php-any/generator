package generator

import (
	"fmt"

	"github.com/php-any/generator/core"
	"github.com/php-any/generator/templates"
)

// MethodGenerator 方法生成器
type MethodGenerator struct {
	templateEngine *templates.TemplateEngine
}

// NewMethodGenerator 创建新的方法生成器
func NewMethodGenerator(templateEngine *templates.TemplateEngine) *MethodGenerator {
	return &MethodGenerator{templateEngine: templateEngine}
}

// GenerateMethod 生成单个方法文件内容
func (mg *MethodGenerator) GenerateMethod(packageName, className string, method core.MethodInfo) (string, error) {
	data := &core.TemplateData{
		PackageName: packageName,
		ClassName:   className,
		Methods: []core.MethodTemplateData{
			{
				Name:       method.Name,
				ClassName:  className,
				Parameters: mg.convertParams(method.Parameters),
				Returns:    mg.convertReturns(method.Returns),
				IsVariadic: method.IsVariadic,
				IsExported: method.IsExported,
			},
		},
	}

	code, err := mg.templateEngine.GenerateMethod(data)
	if err != nil {
		return "", fmt.Errorf("failed to generate method %s: %w", method.Name, err)
	}
	return code, nil
}

func (mg *MethodGenerator) convertParams(params []core.ParameterInfo) []core.ParameterTemplateData {
	var out []core.ParameterTemplateData
	for i, p := range params {
		out = append(out, core.ParameterTemplateData{Name: p.Name, Type: p.Type.TypeName, Index: i})
	}
	return out
}

func (mg *MethodGenerator) convertReturns(returns []core.TypeInfo) []core.ReturnTemplateData {
	var out []core.ReturnTemplateData
	for i := range returns {
		out = append(out, core.ReturnTemplateData{Type: returns[i].TypeName, Index: i})
	}
	return out
}
