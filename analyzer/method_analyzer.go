package analyzer

import (
	"reflect"
	"strings"

	"github.com/php-any/generator/core"
	"github.com/php-any/generator/utils"
)

// MethodAnalyzer 方法分析器
type MethodAnalyzer struct {
	reflectionUtils *utils.ReflectionUtils
	namingUtils     *utils.NamingUtils
}

// NewMethodAnalyzer 创建新的方法分析器
func NewMethodAnalyzer() *MethodAnalyzer {
	return &MethodAnalyzer{
		reflectionUtils: utils.NewReflectionUtils(),
		namingUtils:     utils.NewNamingUtils(),
	}
}

// AnalyzeMethod 分析方法
func (ma *MethodAnalyzer) AnalyzeMethod(m reflect.Method, receiverType *core.TypeInfo) (*core.MethodInfo, error) {
	if m.Type == nil {
		return nil, core.NewGeneratorError(core.ErrCodeTypeAnalysis, "method type is nil", nil)
	}

	// 获取方法信息
	methodInfo := &core.MethodInfo{
		Name:       m.Name,
		IsExported: ma.reflectionUtils.IsExported(m.Name),
		IsVariadic: ma.reflectionUtils.IsMethodVariadic(m),
		Receiver:   receiverType,
	}

	// 分析参数
	parameters, err := ma.analyzeMethodParameters(m)
	if err != nil {
		return nil, err
	}
	methodInfo.Parameters = parameters

	// 分析返回值
	returns, err := ma.analyzeMethodReturns(m)
	if err != nil {
		return nil, err
	}
	methodInfo.Returns = returns

	return methodInfo, nil
}

// AnalyzeMethods 分析类型的所有方法
func (ma *MethodAnalyzer) AnalyzeMethods(t reflect.Type) ([]*core.MethodInfo, error) {
	var methods []*core.MethodInfo

	// 分析值接收者的方法
	if t.Kind() == reflect.Struct {
		for i := 0; i < t.NumMethod(); i++ {
			method := t.Method(i)
			methodInfo, err := ma.AnalyzeMethod(method, nil)
			if err != nil {
				continue // 跳过有问题的方法
			}
			methods = append(methods, methodInfo)
		}
	}

	// 分析指针接收者的方法
	ptrType := reflect.PtrTo(t)
	for i := 0; i < ptrType.NumMethod(); i++ {
		method := ptrType.Method(i)

		// 检查是否已经添加过同名方法
		if !ma.containsMethod(methods, method.Name) {
			methodInfo, err := ma.AnalyzeMethod(method, nil)
			if err != nil {
				continue // 跳过有问题的方法
			}
			methods = append(methods, methodInfo)
		}
	}

	return methods, nil
}

// analyzeMethodParameters 分析方法参数
func (ma *MethodAnalyzer) analyzeMethodParameters(m reflect.Method) ([]core.ParameterInfo, error) {
	var parameters []core.ParameterInfo

	// 跳过接收者参数（索引0）
	for i := 1; i < m.Type.NumIn(); i++ {
		paramType := m.Type.In(i)

		// 创建参数类型信息
		paramTypeInfo := &core.TypeInfo{
			Type:        paramType,
			PackagePath: paramType.PkgPath(),
			PackageName: ma.namingUtils.GetPackageName(paramType.PkgPath()),
			TypeName:    ma.getTypeName(paramType),
			IsPointer:   paramType.Kind() == reflect.Ptr,
			IsInterface: paramType.Kind() == reflect.Interface,
			IsStruct:    paramType.Kind() == reflect.Struct,
			IsFunction:  paramType.Kind() == reflect.Func,
			CacheKey:    paramType.String(),
		}

		// 生成参数名
		paramName := ma.generateParameterName(i-1, paramType)

		parameter := core.ParameterInfo{
			Name:  paramName,
			Type:  paramTypeInfo,
			Index: i - 1, // 相对于实际参数的索引
		}

		parameters = append(parameters, parameter)
	}

	return parameters, nil
}

// analyzeMethodReturns 分析方法返回值
func (ma *MethodAnalyzer) analyzeMethodReturns(m reflect.Method) ([]core.TypeInfo, error) {
	var returns []core.TypeInfo

	for i := 0; i < m.Type.NumOut(); i++ {
		returnType := m.Type.Out(i)

		// 创建返回类型信息
		returnTypeInfo := core.TypeInfo{
			Type:        returnType,
			PackagePath: returnType.PkgPath(),
			PackageName: ma.namingUtils.GetPackageName(returnType.PkgPath()),
			TypeName:    ma.getTypeName(returnType),
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

// getTypeName 获取类型名称
func (ma *MethodAnalyzer) getTypeName(t reflect.Type) string {
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
		return "*" + ma.getTypeName(t.Elem())
	case reflect.Slice:
		return "[]" + ma.getTypeName(t.Elem())
	case reflect.Array:
		return "[" + string(rune(t.Len())) + "]" + ma.getTypeName(t.Elem())
	case reflect.Map:
		return "map[" + ma.getTypeName(t.Key()) + "]" + ma.getTypeName(t.Elem())
	case reflect.Chan:
		switch t.ChanDir() {
		case reflect.RecvDir:
			return "<-" + ma.getTypeName(t.Elem())
		case reflect.SendDir:
			return "chan<- " + ma.getTypeName(t.Elem())
		default:
			return "chan " + ma.getTypeName(t.Elem())
		}
	case reflect.Func:
		return ma.generateFunctionSignature(t)
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
func (ma *MethodAnalyzer) generateFunctionSignature(t reflect.Type) string {
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
		signature.WriteString(ma.getTypeName(paramType))
	}

	signature.WriteString(")")

	// 处理返回值
	numOut := t.NumOut()
	if numOut > 0 {
		if numOut == 1 {
			signature.WriteString(" ")
			signature.WriteString(ma.getTypeName(t.Out(0)))
		} else {
			signature.WriteString(" (")
			for i := 0; i < numOut; i++ {
				if i > 0 {
					signature.WriteString(", ")
				}
				signature.WriteString(ma.getTypeName(t.Out(i)))
			}
			signature.WriteString(")")
		}
	}

	return signature.String()
}

// generateParameterName 生成参数名
func (ma *MethodAnalyzer) generateParameterName(index int, paramType reflect.Type) string {
	// 尝试从类型名生成有意义的参数名
	typeName := ma.getTypeName(paramType)

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
	if ma.namingUtils.IsReservedWord(paramName) {
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

// generateReturnName 生成返回值名
func (ma *MethodAnalyzer) generateReturnName(index int) string {
	return "ret" + string(rune(index+'0'))
}

// containsMethod 检查方法列表中是否包含指定名称的方法
func (ma *MethodAnalyzer) containsMethod(methods []*core.MethodInfo, name string) bool {
	for _, m := range methods {
		if m.Name == name {
			return true
		}
	}
	return false
}

// IsMethodExported 检查方法是否导出
func (ma *MethodAnalyzer) IsMethodExported(methodName string) bool {
	return ma.reflectionUtils.IsExported(methodName)
}

// GetMethodReceiverType 获取方法的接收者类型
func (ma *MethodAnalyzer) GetMethodReceiverType(m reflect.Method) reflect.Type {
	return ma.reflectionUtils.GetMethodReceiverType(m)
}

// GetMethodParameterCount 获取方法的参数数量
func (ma *MethodAnalyzer) GetMethodParameterCount(m reflect.Method) int {
	return m.Type.NumIn() - 1 // 减去接收者
}

// GetMethodReturnCount 获取方法的返回值数量
func (ma *MethodAnalyzer) GetMethodReturnCount(m reflect.Method) int {
	return m.Type.NumOut()
}
