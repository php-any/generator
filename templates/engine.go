package templates

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"text/template"

	"github.com/php-any/generator/core"
)

// TemplateEngine 模板引擎
type TemplateEngine struct {
	templatePath string
	templates    map[string]*template.Template
}

// NewTemplateEngine 创建新的模板引擎
func NewTemplateEngine(templatePath string) *TemplateEngine {
	return &TemplateEngine{
		templatePath: templatePath,
		templates:    make(map[string]*template.Template),
	}
}

// LoadTemplates 加载所有模板
func (te *TemplateEngine) LoadTemplates() error {
	// 加载函数模板
	if err := te.loadTemplate("function", "function.tmpl"); err != nil {
		return fmt.Errorf("failed to load function template: %w", err)
	}

	// 加载类模板
	if err := te.loadTemplate("class", "class.tmpl"); err != nil {
		return fmt.Errorf("failed to load class template: %w", err)
	}

	// 加载方法模板
	if err := te.loadTemplate("method", "method.tmpl"); err != nil {
		return fmt.Errorf("failed to load method template: %w", err)
	}

	return nil
}

// LoadTemplate 兼容接口，返回模板对象
func (te *TemplateEngine) LoadTemplate(name string) (interface{}, error) {
	tmpl, ok := te.templates[name]
	if !ok {
		return nil, fmt.Errorf("template %s not found", name)
	}
	return tmpl, nil
}

// loadTemplate 加载单个模板
func (te *TemplateEngine) loadTemplate(name, filename string) error {
	templatePath := filepath.Join(te.templatePath, filename)

	// 读取模板文件
	content, err := os.ReadFile(templatePath)
	if err != nil {
		return fmt.Errorf("failed to read template file %s: %w", templatePath, err)
	}

	// 解析模板
	tmpl, err := template.New(name).Parse(string(content))
	if err != nil {
		return fmt.Errorf("failed to parse template %s: %w", filename, err)
	}

	te.templates[name] = tmpl
	return nil
}

// ExecuteTemplate 执行模板
func (te *TemplateEngine) ExecuteTemplate(name string, data interface{}) (string, error) {
	tmpl, exists := te.templates[name]
	if !exists {
		return "", fmt.Errorf("template %s not found", name)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("failed to execute template %s: %w", name, err)
	}

	return buf.String(), nil
}

// GenerateFunction 生成函数代码
func (te *TemplateEngine) GenerateFunction(data *core.TemplateData) (string, error) {
	return te.ExecuteTemplate("function", data)
}

// GenerateClass 生成类代码
func (te *TemplateEngine) GenerateClass(data *core.TemplateData) (string, error) {
	return te.ExecuteTemplate("class", data)
}

// GenerateMethod 生成方法代码
func (te *TemplateEngine) GenerateMethod(data *core.TemplateData) (string, error) {
	// 如果使用 data.Methods[0] 传入方法名，适配模板
	if len(data.Methods) == 1 {
		// 将第一项方法名赋予 data 中便于模板访问的字段（MethodName）
		// TemplateData 没有 MethodName 字段，模板通过 .Methods[0] 访问
	}
	return te.ExecuteTemplate("method", data)
}

// GetTemplateNames 获取所有模板名称
func (te *TemplateEngine) GetTemplateNames() []string {
	names := make([]string, 0, len(te.templates))
	for name := range te.templates {
		names = append(names, name)
	}
	return names
}

// ReloadTemplates 重新加载模板
func (te *TemplateEngine) ReloadTemplates() error {
	te.templates = make(map[string]*template.Template)
	return te.LoadTemplates()
}
