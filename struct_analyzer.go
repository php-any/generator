package generator

import (
	"reflect"
	"unicode"
	"unicode/utf8"
)

// PropertyInfo 描述结构体字段（公有）的基础信息
type PropertyInfo struct {
	Name  string      // 字段名（导出名）
	Type  string      // 字段类型（人类可读）
	Value interface{} // 字段当前值（从实例提取）
}

// StructAnalyzer 提供结构体字段提取工具
type StructAnalyzer struct{}

// ExtractProperties 提取结构体的所有导出字段作为属性
func (a *StructAnalyzer) ExtractProperties(typ reflect.Type, instance interface{}) []PropertyInfo {
	var properties []PropertyInfo

	val := reflect.ValueOf(instance)
	if val.IsValid() && val.Kind() == reflect.Ptr {
		val = val.Elem()
	}

	for i := 0; i < typ.NumField(); i++ {
		field := typ.Field(i)

		// 跳过未导出字段
		if !a.isPublicField(field.Name) {
			continue
		}

		var fieldValue reflect.Value
		if val.IsValid() && val.Kind() == reflect.Struct {
			fieldValue = val.Field(i)
		}

		properties = append(properties, PropertyInfo{
			Name:  field.Name,
			Type:  a.getPropertyType(field.Type),
			Value: a.getFieldValue(fieldValue),
		})
	}

	return properties
}

func (a *StructAnalyzer) isPublicField(name string) bool {
	if name == "" {
		return false
	}
	r, _ := utf8.DecodeRuneInString(name)
	return unicode.IsUpper(r)
}

func (a *StructAnalyzer) getFieldValue(v reflect.Value) interface{} {
	if !v.IsValid() {
		return nil
	}
	// 避免未初始化的零值 panic
	if v.Kind() == reflect.Ptr {
		if v.IsNil() {
			return nil
		}
		v = v.Elem()
	}
	return v.Interface()
}

func (a *StructAnalyzer) getPropertyType(t reflect.Type) string {
	if t == nil {
		return "<nil>"
	}
	return t.String()
}
