package generator

import (
	"fmt"
	"strings"

	"github.com/php-any/generator/core"
	"github.com/php-any/generator/templates"
	"github.com/php-any/generator/utils"
)

// FunctionGenerator 函数生成器
type FunctionGenerator struct {
	templateEngine *templates.TemplateEngine
	config         *core.GeneratorConfig
	namingUtils    *utils.NamingUtils
}

// NewFunctionGenerator 创建新的函数生成器
func NewFunctionGenerator(templateEngine *templates.TemplateEngine, config *core.GeneratorConfig) *FunctionGenerator {
	return &FunctionGenerator{
		templateEngine: templateEngine,
		config:         config,
		namingUtils:    utils.NewNamingUtils(),
	}
}

// GenerateFunction 生成函数代码
func (fg *FunctionGenerator) GenerateFunction(functionInfo *core.FunctionInfo, packageName string) (string, error) {
	if functionInfo == nil {
		return "", core.NewGeneratorError(core.ErrCodeCodeGeneration, "function info is nil", nil)
	}

	// 创建模板数据
	templateData := &core.TemplateData{
		PackageName:  packageName,
		FunctionName: functionInfo.Name,
		Parameters:   fg.convertParameters(functionInfo.Parameters),
		Returns:      fg.convertReturns(functionInfo.Returns),
	}

	// 使用模板引擎生成代码
	code, err := fg.templateEngine.GenerateFunction(templateData)
	if err != nil {
		return "", fmt.Errorf("failed to generate function: %w", err)
	}

	return code, nil
}

// GenerateFunctionProxy 生成函数代理
func (fg *FunctionGenerator) GenerateFunctionProxy(functionInfo *core.FunctionInfo, packageName string) (string, error) {
	if functionInfo == nil {
		return "", core.NewGeneratorError(core.ErrCodeCodeGeneration, "function info is nil", nil)
	}

	var code strings.Builder

	// 包声明
	code.WriteString(fmt.Sprintf("package %s\n\n", packageName))

	// 导入语句
	code.WriteString(fg.generateFunctionImports(functionInfo))

	// 函数代理结构体
	code.WriteString(fg.generateFunctionProxyStruct(functionInfo))

	// 构造函数
	code.WriteString(fg.generateFunctionProxyConstructor(functionInfo))

	// GetName 方法
	code.WriteString(fg.generateFunctionGetName(functionInfo))

	// Call 方法
	code.WriteString(fg.generateFunctionCall(functionInfo))

	return code.String(), nil
}

// generateFunctionImports 生成函数导入
func (fg *FunctionGenerator) generateFunctionImports(functionInfo *core.FunctionInfo) string {
	var imports strings.Builder
	imports.WriteString("import (\n")
	imports.WriteString("\t\"fmt\"\n")
	imports.WriteString("\t\"github.com/php-any/origami/data\"\n")
	imports.WriteString("\t\"github.com/php-any/origami/node\"\n")
	imports.WriteString(")\n\n")

	return imports.String()
}

// generateFunctionProxyStruct 生成函数代理结构体
func (fg *FunctionGenerator) generateFunctionProxyStruct(functionInfo *core.FunctionInfo) string {
	var structCode strings.Builder

	structCode.WriteString(fmt.Sprintf("// %sFunction 函数代理\n", functionInfo.Name))
	structCode.WriteString(fmt.Sprintf("type %sFunction struct {\n", functionInfo.Name))
	structCode.WriteString("\tnode.Node\n")
	structCode.WriteString("}\n\n")

	return structCode.String()
}

// generateFunctionProxyConstructor 生成函数代理构造函数
func (fg *FunctionGenerator) generateFunctionProxyConstructor(functionInfo *core.FunctionInfo) string {
	var constructorCode strings.Builder

	constructorCode.WriteString(fmt.Sprintf("// New%sFunction 创建新的函数代理\n", functionInfo.Name))
	constructorCode.WriteString(fmt.Sprintf("func New%sFunction() *%sFunction {\n", functionInfo.Name, functionInfo.Name))
	constructorCode.WriteString(fmt.Sprintf("\treturn &%sFunction{}\n", functionInfo.Name))
	constructorCode.WriteString("}\n\n")

	return constructorCode.String()
}

// generateFunctionGetName 生成函数GetName方法
func (fg *FunctionGenerator) generateFunctionGetName(functionInfo *core.FunctionInfo) string {
	var getNameCode strings.Builder

	getNameCode.WriteString("// GetName 获取函数名\n")
	getNameCode.WriteString("func (f *" + functionInfo.Name + "Function) GetName() string {\n")
	getNameCode.WriteString(fmt.Sprintf("\treturn \"%s\"\n", functionInfo.Name))
	getNameCode.WriteString("}\n\n")

	return getNameCode.String()
}

// generateFunctionCall 生成函数Call方法
func (fg *FunctionGenerator) generateFunctionCall(functionInfo *core.FunctionInfo) string {
	var callCode strings.Builder

	callCode.WriteString("// Call 调用函数\n")
	callCode.WriteString("func (f *" + functionInfo.Name + "Function) Call(ctx data.Context, args []data.Value) (data.GetValue, data.Control) {\n")

	// 参数转换
	if len(functionInfo.Parameters) > 0 {
		callCode.WriteString("\t// 参数转换\n")
		for i, param := range functionInfo.Parameters {
			callCode.WriteString(fg.generateParameterConversion(i, param))
		}
		callCode.WriteString("\n")
	}

	// 函数调用
	callCode.WriteString("\t// TODO: 实现实际的函数调用逻辑\n")
	callCode.WriteString("\t// 这里应该调用原始函数\n\n")

	// 返回值处理
	if len(functionInfo.Returns) > 0 {
		callCode.WriteString("\t// 返回值处理\n")
		for i, ret := range functionInfo.Returns {
			callCode.WriteString(fmt.Sprintf("\tvar ret%d %s\n", i, ret.TypeName))
		}
		callCode.WriteString("\n")
		callCode.WriteString("\treturn data.NewAnyValue(ret0), nil\n")
	} else {
		callCode.WriteString(fmt.Sprintf("\treturn data.NewStringValue(\"%s 调用完成\"), nil\n", functionInfo.Name))
	}

	callCode.WriteString("}\n\n")

	return callCode.String()
}

// generateParameterConversion 生成参数转换代码
func (fg *FunctionGenerator) generateParameterConversion(index int, param core.ParameterInfo) string {
	var conversionCode strings.Builder

	conversionCode.WriteString(fmt.Sprintf("\tvar arg%d %s\n", index, param.Type.TypeName))
	conversionCode.WriteString(fmt.Sprintf("\tif len(args) > %d {\n", index))
	conversionCode.WriteString(fmt.Sprintf("\t\tswitch v := args[%d].(type) {\n", index))
	conversionCode.WriteString("\t\tcase *data.StringValue:\n")
	conversionCode.WriteString(fmt.Sprintf("\t\t\targ%d = v.Value\n", index))
	conversionCode.WriteString("\t\tcase *data.IntValue:\n")
	conversionCode.WriteString(fmt.Sprintf("\t\t\targ%d = v.Value\n", index))
	conversionCode.WriteString("\t\tcase *data.BoolValue:\n")
	conversionCode.WriteString(fmt.Sprintf("\t\t\targ%d = v.Value\n", index))
	conversionCode.WriteString("\t\tcase *data.FloatValue:\n")
	conversionCode.WriteString(fmt.Sprintf("\t\t\targ%d = v.Value\n", index))
	conversionCode.WriteString("\t\tcase *data.ClassValue:\n")
	conversionCode.WriteString("\t\t\tif p, ok := v.Class.(interface{ GetSource() any }); ok {\n")
	conversionCode.WriteString("\t\t\t\tif src := p.GetSource(); src != nil {\n")
	conversionCode.WriteString(fmt.Sprintf("\t\t\t\t\targ%d = src.(%s)\n", index, param.Type.TypeName))
	conversionCode.WriteString("\t\t\t\t}\n")
	conversionCode.WriteString("\t\t\t}\n")
	conversionCode.WriteString("\t\tcase *data.ProxyValue:\n")
	conversionCode.WriteString("\t\t\tif p, ok := v.Class.(interface{ GetSource() any }); ok {\n")
	conversionCode.WriteString("\t\t\t\tif src := p.GetSource(); src != nil {\n")
	conversionCode.WriteString(fmt.Sprintf("\t\t\t\t\targ%d = src.(%s)\n", index, param.Type.TypeName))
	conversionCode.WriteString("\t\t\t\t}\n")
	conversionCode.WriteString("\t\t\t}\n")
	conversionCode.WriteString("\t\tcase *data.AnyValue:\n")
	conversionCode.WriteString(fmt.Sprintf("\t\t\targ%d = v.Value.(%s)\n", index, param.Type.TypeName))
	conversionCode.WriteString("\t\tdefault:\n")
	conversionCode.WriteString(fmt.Sprintf("\t\t\treturn nil, data.NewErrorThrow(nil, fmt.Errorf(\"函数 参数类型不支持, index: %d\"))\n", index))
	conversionCode.WriteString("\t\t}\n")
	conversionCode.WriteString("\t} else {\n")
	conversionCode.WriteString(fmt.Sprintf("\t\treturn nil, data.NewErrorThrow(nil, fmt.Errorf(\"函数 缺少参数, index: %d\"))\n", index))
	conversionCode.WriteString("\t}\n")

	return conversionCode.String()
}

// convertParameters 转换参数信息
func (fg *FunctionGenerator) convertParameters(parameters []core.ParameterInfo) []core.ParameterTemplateData {
	var result []core.ParameterTemplateData

	for i, param := range parameters {
		paramData := core.ParameterTemplateData{
			Name:  param.Name,
			Type:  param.Type.TypeName,
			Index: i,
		}
		result = append(result, paramData)
	}

	return result
}

// convertReturns 转换返回值信息
func (fg *FunctionGenerator) convertReturns(returns []core.TypeInfo) []core.ReturnTemplateData {
	var result []core.ReturnTemplateData

	for _, ret := range returns {
		retData := core.ReturnTemplateData{
			Type: ret.TypeName,
		}
		result = append(result, retData)
	}

	return result
}

// GenerateFunctionRegistry 生成函数注册代码
func (fg *FunctionGenerator) GenerateFunctionRegistry(functions []*core.FunctionInfo, packageName string) (string, error) {
	var registryCode strings.Builder

	// 包声明
	registryCode.WriteString(fmt.Sprintf("package %s\n\n", packageName))

	// 导入语句
	registryCode.WriteString("import (\n")
	registryCode.WriteString("\t\"github.com/php-any/origami/data\"\n")
	registryCode.WriteString("\t\"github.com/php-any/origami/node\"\n")
	registryCode.WriteString(")\n\n")

	// 注册函数
	registryCode.WriteString("// RegisterFunctions 注册所有函数\n")
	registryCode.WriteString("func RegisterFunctions() map[string]node.Node {\n")
	registryCode.WriteString("\tfunctions := make(map[string]node.Node)\n\n")

	for _, function := range functions {
		registryCode.WriteString(fmt.Sprintf("\tfunctions[\"%s\"] = New%sFunction()\n", function.Name, function.Name))
	}

	registryCode.WriteString("\n\treturn functions\n")
	registryCode.WriteString("}\n\n")

	return registryCode.String(), nil
}
