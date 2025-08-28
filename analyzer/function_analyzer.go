package analyzer

import (
	"reflect"
	"strings"

	"github.com/php-any/generator/core"
	"github.com/php-any/generator/utils"
)

// FunctionAnalyzer 函数分析器
type FunctionAnalyzer struct {
	reflectionUtils *utils.ReflectionUtils
	namingUtils     *utils.NamingUtils
}

// NewFunctionAnalyzer 创建新的函数分析器
func NewFunctionAnalyzer() *FunctionAnalyzer {
	return &FunctionAnalyzer{
		reflectionUtils: utils.NewReflectionUtils(),
		namingUtils:     utils.NewNamingUtils(),
	}
}

// AnalyzeFunction 分析函数
func (fa *FunctionAnalyzer) AnalyzeFunction(fn interface{}) (*core.FunctionInfo, error) {
	if fn == nil {
		return nil, core.NewGeneratorError(core.ErrCodeTypeAnalysis, "function is nil", nil)
	}

	fnType := reflect.TypeOf(fn)
	if fnType.Kind() != reflect.Func {
		return nil, core.NewGeneratorError(core.ErrCodeTypeAnalysis, "not a function", nil)
	}

	// 获取函数名
	fnName := fa.getFunctionName(fn)

	// 分析参数
	parameters, err := fa.analyzeFunctionParameters(fnType)
	if err != nil {
		return nil, err
	}

	// 分析返回值
	returns, err := fa.analyzeFunctionReturns(fnType)
	if err != nil {
		return nil, err
	}

	// 创建函数信息
	functionInfo := &core.FunctionInfo{
		Name:       fnName,
		Package:    fnType.PkgPath(),
		Parameters: parameters,
		Returns:    returns,
		IsVariadic: fnType.IsVariadic(),
	}

	return functionInfo, nil
}

// AnalyzeFunctionType 分析函数类型
func (fa *FunctionAnalyzer) AnalyzeFunctionType(fnType reflect.Type) (*core.FunctionInfo, error) {
	if fnType == nil {
		return nil, core.NewGeneratorError(core.ErrCodeTypeAnalysis, "function type is nil", nil)
	}

	if fnType.Kind() != reflect.Func {
		return nil, core.NewGeneratorError(core.ErrCodeTypeAnalysis, "not a function type", nil)
	}

	// 分析参数
	parameters, err := fa.analyzeFunctionParameters(fnType)
	if err != nil {
		return nil, err
	}

	// 分析返回值
	returns, err := fa.analyzeFunctionReturns(fnType)
	if err != nil {
		return nil, err
	}

	// 创建函数信息
	functionInfo := &core.FunctionInfo{
		Name:       "func", // 匿名函数
		Package:    fnType.PkgPath(),
		Parameters: parameters,
		Returns:    returns,
		IsVariadic: fnType.IsVariadic(),
	}

	return functionInfo, nil
}

// analyzeFunctionParameters 分析函数参数
func (fa *FunctionAnalyzer) analyzeFunctionParameters(fnType reflect.Type) ([]core.ParameterInfo, error) {
	var parameters []core.ParameterInfo

	for i := 0; i < fnType.NumIn(); i++ {
		paramType := fnType.In(i)

		// 创建参数类型信息
		paramTypeInfo := &core.TypeInfo{
			Type:        paramType,
			PackagePath: paramType.PkgPath(),
			PackageName: fa.namingUtils.GetPackageName(paramType.PkgPath()),
			TypeName:    fa.getTypeName(paramType),
			IsPointer:   paramType.Kind() == reflect.Ptr,
			IsInterface: paramType.Kind() == reflect.Interface,
			IsStruct:    paramType.Kind() == reflect.Struct,
			IsFunction:  paramType.Kind() == reflect.Func,
			CacheKey:    paramType.String(),
		}

		// 生成参数名
		paramName := fa.generateParameterName(i, paramType)

		parameter := core.ParameterInfo{
			Name:  paramName,
			Type:  paramTypeInfo,
			Index: i,
		}

		parameters = append(parameters, parameter)
	}

	return parameters, nil
}

// analyzeFunctionReturns 分析函数返回值
func (fa *FunctionAnalyzer) analyzeFunctionReturns(fnType reflect.Type) ([]core.TypeInfo, error) {
	var returns []core.TypeInfo

	for i := 0; i < fnType.NumOut(); i++ {
		returnType := fnType.Out(i)

		// 创建返回类型信息
		returnTypeInfo := core.TypeInfo{
			Type:        returnType,
			PackagePath: returnType.PkgPath(),
			PackageName: fa.namingUtils.GetPackageName(returnType.PkgPath()),
			TypeName:    fa.getTypeName(returnType),
			IsPointer:   returnType.Kind() == reflect.Ptr,
			IsInterface: returnType.Kind() == reflect.Interface,
			IsStruct:    returnType.Kind() == reflect.Struct,
			IsFunction:  returnType.Kind() == reflect.Func,
			CacheKey:    returnType.String(),
		}

		returns = append(returns, returnTypeInfo)
	}

	return returns, nil
}

// getFunctionName 获取函数名
func (fa *FunctionAnalyzer) getFunctionName(fn interface{}) string {
	// 尝试从函数值获取名称
	fnValue := reflect.ValueOf(fn)
	if fnValue.Kind() == reflect.Func {
		// 对于匿名函数，返回 "func"
		return "func"
	}

	// 对于命名函数，尝试获取名称
	fnType := reflect.TypeOf(fn)
	if fnType.Kind() == reflect.Func {
		// 这里无法直接获取函数名，返回 "func"
		return "func"
	}

	return "unknown"
}

// generateParameterName 生成参数名
func (fa *FunctionAnalyzer) generateParameterName(index int, paramType reflect.Type) string {
	// 尝试从类型名生成有意义的参数名
	typeName := fa.getTypeName(paramType)

	// 移除包前缀
	if strings.Contains(typeName, ".") {
		parts := strings.Split(typeName, ".")
		typeName = parts[len(parts)-1]
	}

	// 移除指针符号
	typeName = strings.TrimPrefix(typeName, "*")

	// 移除切片符号
	if strings.HasPrefix(typeName, "[]") {
		typeName = typeName[2:]
	}

	// 转换为小写作为参数名
	paramName := strings.ToLower(typeName)

	// 如果参数名是Go关键字，添加前缀
	if fa.namingUtils.IsReservedWord(paramName) {
		paramName = "arg" + paramName
	}

	// 如果参数名以数字开头，添加前缀
	if len(paramName) > 0 && paramName[0] >= '0' && paramName[0] <= '9' {
		paramName = "arg" + paramName
	}

	// 如果参数名为空，使用默认名称
	if paramName == "" {
		paramName = "arg" + string(rune(index+'0'))
	}

	return paramName
}

// getTypeName 获取类型名称
func (fa *FunctionAnalyzer) getTypeName(t reflect.Type) string {
	if t == nil {
		return "interface{}"
	}

	switch t.Kind() {
	case reflect.Bool:
		return "bool"
	case reflect.Int:
		return "int"
	case reflect.Int8:
		return "int8"
	case reflect.Int16:
		return "int16"
	case reflect.Int32:
		return "int32"
	case reflect.Int64:
		return "int64"
	case reflect.Uint:
		return "uint"
	case reflect.Uint8:
		return "uint8"
	case reflect.Uint16:
		return "uint16"
	case reflect.Uint32:
		return "uint32"
	case reflect.Uint64:
		return "uint64"
	case reflect.Uintptr:
		return "uintptr"
	case reflect.Float32:
		return "float32"
	case reflect.Float64:
		return "float64"
	case reflect.Complex64:
		return "complex64"
	case reflect.Complex128:
		return "complex128"
	case reflect.String:
		return "string"
	case reflect.Interface:
		return "interface{}"
	case reflect.Ptr:
		return "*" + fa.getTypeName(t.Elem())
	case reflect.Slice:
		return "[]" + fa.getTypeName(t.Elem())
	case reflect.Array:
		return "[" + string(rune(t.Len())) + "]" + fa.getTypeName(t.Elem())
	case reflect.Map:
		return "map[" + fa.getTypeName(t.Key()) + "]" + fa.getTypeName(t.Elem())
	case reflect.Chan:
		switch t.ChanDir() {
		case reflect.RecvDir:
			return "<-" + fa.getTypeName(t.Elem())
		case reflect.SendDir:
			return "chan<- " + fa.getTypeName(t.Elem())
		default:
			return "chan " + fa.getTypeName(t.Elem())
		}
	case reflect.Func:
		return fa.generateFunctionSignature(t)
	case reflect.Struct:
		if t.PkgPath() != "" {
			return t.PkgPath() + "." + t.Name()
		}
		return t.Name()
	default:
		return t.String()
	}
}

// generateFunctionSignature 生成函数签名
func (fa *FunctionAnalyzer) generateFunctionSignature(t reflect.Type) string {
	if t.Kind() != reflect.Func {
		return "func"
	}

	var signature strings.Builder
	signature.WriteString("func(")

	// 处理参数
	numIn := t.NumIn()
	for i := 0; i < numIn; i++ {
		if i > 0 {
			signature.WriteString(", ")
		}
		paramType := t.In(i)
		signature.WriteString(fa.getTypeName(paramType))
	}

	signature.WriteString(")")

	// 处理返回值
	numOut := t.NumOut()
	if numOut > 0 {
		if numOut == 1 {
			signature.WriteString(" ")
			signature.WriteString(fa.getTypeName(t.Out(0)))
		} else {
			signature.WriteString(" (")
			for i := 0; i < numOut; i++ {
				if i > 0 {
					signature.WriteString(", ")
				}
				signature.WriteString(fa.getTypeName(t.Out(i)))
			}
			signature.WriteString(")")
		}
	}

	return signature.String()
}

// IsFunctionVariadic 检查函数是否为可变参数
func (fa *FunctionAnalyzer) IsFunctionVariadic(fn interface{}) bool {
	if fn == nil {
		return false
	}

	fnType := reflect.TypeOf(fn)
	return fnType.Kind() == reflect.Func && fnType.IsVariadic()
}

// GetFunctionParameterCount 获取函数的参数数量
func (fa *FunctionAnalyzer) GetFunctionParameterCount(fn interface{}) int {
	if fn == nil {
		return 0
	}

	fnType := reflect.TypeOf(fn)
	if fnType.Kind() != reflect.Func {
		return 0
	}

	return fnType.NumIn()
}

// GetFunctionReturnCount 获取函数的返回值数量
func (fa *FunctionAnalyzer) GetFunctionReturnCount(fn interface{}) int {
	if fn == nil {
		return 0
	}

	fnType := reflect.TypeOf(fn)
	if fnType.Kind() != reflect.Func {
		return 0
	}

	return fnType.NumOut()
}

// GetFunctionType 获取函数类型
func (fa *FunctionAnalyzer) GetFunctionType(fn interface{}) (reflect.Type, error) {
	if fn == nil {
		return nil, core.NewGeneratorError(core.ErrCodeTypeAnalysis, "function is nil", nil)
	}

	fnType := reflect.TypeOf(fn)
	if fnType.Kind() != reflect.Func {
		return nil, core.NewGeneratorError(core.ErrCodeTypeAnalysis, "not a function", nil)
	}

	return fnType, nil
}
