package analyzer

import (
	"reflect"
	"strings"

	"github.com/php-any/generator/core"
	"github.com/php-any/generator/utils"
)

// FieldAnalyzer 字段分析器
type FieldAnalyzer struct {
	reflectionUtils *utils.ReflectionUtils
	namingUtils     *utils.NamingUtils
}

// NewFieldAnalyzer 创建新的字段分析器
func NewFieldAnalyzer() *FieldAnalyzer {
	return &FieldAnalyzer{
		reflectionUtils: utils.NewReflectionUtils(),
		namingUtils:     utils.NewNamingUtils(),
	}
}

// AnalyzeField 分析单个字段
func (fa *FieldAnalyzer) AnalyzeField(field reflect.StructField) (*core.FieldInfo, error) {
	if field.Type == nil {
		return nil, core.NewGeneratorError(core.ErrCodeTypeAnalysis, "field type is nil", nil)
	}

	// 创建字段类型信息
	fieldTypeInfo := &core.TypeInfo{
		Type:        field.Type,
		PackagePath: field.Type.PkgPath(),
		PackageName: fa.namingUtils.GetPackageName(field.Type.PkgPath()),
		TypeName:    fa.getTypeName(field.Type),
		IsPointer:   field.Type.Kind() == reflect.Ptr,
		IsInterface: field.Type.Kind() == reflect.Interface,
		IsStruct:    field.Type.Kind() == reflect.Struct,
		IsFunction:  field.Type.Kind() == reflect.Func,
		CacheKey:    field.Type.String(),
	}

	// 创建字段信息
	fieldInfo := &core.FieldInfo{
		Name:       field.Name,
		Type:       fieldTypeInfo,
		IsExported: fa.reflectionUtils.IsExported(field.Name),
		Tag:        field.Tag,
	}

	return fieldInfo, nil
}

// AnalyzeFields 分析结构体的所有字段
func (fa *FieldAnalyzer) AnalyzeFields(t reflect.Type) ([]core.FieldInfo, error) {
	if t == nil {
		return nil, core.NewGeneratorError(core.ErrCodeTypeAnalysis, "type is nil", nil)
	}

	if t.Kind() != reflect.Struct {
		return nil, core.NewGeneratorError(core.ErrCodeTypeAnalysis, "type is not a struct", nil)
	}

	var fields []core.FieldInfo

	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)

		fieldInfo, err := fa.AnalyzeField(field)
		if err != nil {
			continue // 跳过有问题的字段
		}

		fields = append(fields, *fieldInfo)
	}

	return fields, nil
}

// AnalyzeEmbeddedFields 分析嵌入字段
func (fa *FieldAnalyzer) AnalyzeEmbeddedFields(t reflect.Type) ([]core.FieldInfo, error) {
	if t == nil {
		return nil, core.NewGeneratorError(core.ErrCodeTypeAnalysis, "type is nil", nil)
	}

	if t.Kind() != reflect.Struct {
		return nil, core.NewGeneratorError(core.ErrCodeTypeAnalysis, "type is not a struct", nil)
	}

	var embeddedFields []core.FieldInfo

	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)

		// 检查是否为嵌入字段
		if field.Anonymous {
			fieldInfo, err := fa.AnalyzeField(field)
			if err != nil {
				continue
			}
			embeddedFields = append(embeddedFields, *fieldInfo)
		}
	}

	return embeddedFields, nil
}

// AnalyzeExportedFields 分析导出的字段
func (fa *FieldAnalyzer) AnalyzeExportedFields(t reflect.Type) ([]core.FieldInfo, error) {
	if t == nil {
		return nil, core.NewGeneratorError(core.ErrCodeTypeAnalysis, "type is nil", nil)
	}

	if t.Kind() != reflect.Struct {
		return nil, core.NewGeneratorError(core.ErrCodeTypeAnalysis, "type is not a struct", nil)
	}

	var exportedFields []core.FieldInfo

	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)

		// 只分析导出的字段
		if fa.reflectionUtils.IsExported(field.Name) {
			fieldInfo, err := fa.AnalyzeField(field)
			if err != nil {
				continue
			}
			exportedFields = append(exportedFields, *fieldInfo)
		}
	}

	return exportedFields, nil
}

// AnalyzeUnexportedFields 分析未导出的字段
func (fa *FieldAnalyzer) AnalyzeUnexportedFields(t reflect.Type) ([]core.FieldInfo, error) {
	if t == nil {
		return nil, core.NewGeneratorError(core.ErrCodeTypeAnalysis, "type is nil", nil)
	}

	if t.Kind() != reflect.Struct {
		return nil, core.NewGeneratorError(core.ErrCodeTypeAnalysis, "type is not a struct", nil)
	}

	var unexportedFields []core.FieldInfo

	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)

		// 只分析未导出的字段
		if !fa.reflectionUtils.IsExported(field.Name) {
			fieldInfo, err := fa.AnalyzeField(field)
			if err != nil {
				continue
			}
			unexportedFields = append(unexportedFields, *fieldInfo)
		}
	}

	return unexportedFields, nil
}

// GetFieldByName 根据名称获取字段
func (fa *FieldAnalyzer) GetFieldByName(t reflect.Type, name string) (*core.FieldInfo, error) {
	if t == nil {
		return nil, core.NewGeneratorError(core.ErrCodeTypeAnalysis, "type is nil", nil)
	}

	if t.Kind() != reflect.Struct {
		return nil, core.NewGeneratorError(core.ErrCodeTypeAnalysis, "type is not a struct", nil)
	}

	field, found := t.FieldByName(name)
	if !found {
		return nil, core.NewGeneratorError(core.ErrCodeTypeAnalysis, "field not found: "+name, nil)
	}

	return fa.AnalyzeField(field)
}

// GetFieldByIndex 根据索引获取字段
func (fa *FieldAnalyzer) GetFieldByIndex(t reflect.Type, index int) (*core.FieldInfo, error) {
	if t == nil {
		return nil, core.NewGeneratorError(core.ErrCodeTypeAnalysis, "type is nil", nil)
	}

	if t.Kind() != reflect.Struct {
		return nil, core.NewGeneratorError(core.ErrCodeTypeAnalysis, "type is not a struct", nil)
	}

	if index < 0 || index >= t.NumField() {
		return nil, core.NewGeneratorError(core.ErrCodeTypeAnalysis, "field index out of range", nil)
	}

	field := t.Field(index)
	return fa.AnalyzeField(field)
}

// GetFieldCount 获取字段数量
func (fa *FieldAnalyzer) GetFieldCount(t reflect.Type) (int, error) {
	if t == nil {
		return 0, core.NewGeneratorError(core.ErrCodeTypeAnalysis, "type is nil", nil)
	}

	if t.Kind() != reflect.Struct {
		return 0, core.NewGeneratorError(core.ErrCodeTypeAnalysis, "type is not a struct", nil)
	}

	return t.NumField(), nil
}

// IsFieldExported 检查字段是否导出
func (fa *FieldAnalyzer) IsFieldExported(fieldName string) bool {
	return fa.reflectionUtils.IsExported(fieldName)
}

// IsFieldEmbedded 检查字段是否嵌入
func (fa *FieldAnalyzer) IsFieldEmbedded(t reflect.Type, fieldName string) bool {
	if t == nil || t.Kind() != reflect.Struct {
		return false
	}

	field, found := t.FieldByName(fieldName)
	if !found {
		return false
	}

	return field.Anonymous
}

// GetFieldTag 获取字段标签
func (fa *FieldAnalyzer) GetFieldTag(t reflect.Type, fieldName string) (reflect.StructTag, error) {
	if t == nil {
		return "", core.NewGeneratorError(core.ErrCodeTypeAnalysis, "type is nil", nil)
	}

	if t.Kind() != reflect.Struct {
		return "", core.NewGeneratorError(core.ErrCodeTypeAnalysis, "type is not a struct", nil)
	}

	field, found := t.FieldByName(fieldName)
	if !found {
		return "", core.NewGeneratorError(core.ErrCodeTypeAnalysis, "field not found: "+fieldName, nil)
	}

	return field.Tag, nil
}

// getTypeName 获取类型名称
func (fa *FieldAnalyzer) getTypeName(t reflect.Type) string {
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
func (fa *FieldAnalyzer) generateFunctionSignature(t reflect.Type) string {
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
