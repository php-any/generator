package converter

import (
	"reflect"
	"testing"

	"github.com/php-any/generator/analyzer"
	"github.com/php-any/generator/config"
	"github.com/php-any/generator/core"
)

// TestStruct 测试结构体
type TestStruct struct {
	Name   string
	Age    int
	Active bool
}

// TestMapStruct 包含映射的结构体
type TestMapStruct struct {
	Data map[string]int
	List []string
}

// TestTypeConverter_BasicTypes 测试基本类型转换
func TestTypeConverter_BasicTypes(t *testing.T) {
	ctx := core.NewGeneratorContext(&core.GenOptions{})
	converter := NewTypeConverter(ctx)

	tests := []struct {
		name     string
		typeInfo *core.TypeInfo
		expected string
	}{
		{"string", &core.TypeInfo{TypeName: "string"}, "string"},
		{"int", &core.TypeInfo{TypeName: "int"}, "int"},
		{"bool", &core.TypeInfo{TypeName: "bool"}, "bool"},
		{"error", &core.TypeInfo{TypeName: "error"}, "error"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := converter.ConvertType(tt.typeInfo)
			if err != nil {
				t.Fatalf("ConvertType failed: %v", err)
			}
			if result != tt.expected {
				t.Errorf("expected %s, got %s", tt.expected, result)
			}
		})
	}
}

// TestTypeConverter_StructTypes 测试结构体类型转换
func TestTypeConverter_StructTypes(t *testing.T) {
	ctx := core.NewGeneratorContext(&core.GenOptions{})
	converter := NewTypeConverter(ctx)

	// 创建类型分析器来分析结构体
	configManager := config.NewConfigManager(ctx.ErrorHandler)
	ctx.SetConfigManager(configManager)
	analyzer := analyzer.NewTypeAnalyzer(ctx)

	// 分析TestStruct
	testStructType, err := analyzer.AnalyzeType(reflect.TypeOf(TestStruct{}))
	if err != nil {
		t.Fatalf("Failed to analyze TestStruct: %v", err)
	}

	// 测试结构体转换
	result, err := converter.ConvertType(testStructType)
	if err != nil {
		t.Fatalf("ConvertType failed: %v", err)
	}

	// 由于是本地类型，应该直接返回类型名
	if result != "origamisrc.TestStruct" {
		t.Errorf("expected origamisrc.TestStruct, got %s", result)
	}
}

// TestTypeConverter_PointerTypes 测试指针类型转换
func TestTypeConverter_PointerTypes(t *testing.T) {
	ctx := core.NewGeneratorContext(&core.GenOptions{})
	converter := NewTypeConverter(ctx)

	// 创建指针类型
	pointerType := &core.TypeInfo{
		TypeName:  "*TestStruct",
		IsPointer: true,
		Type:      reflect.TypeOf(&TestStruct{}),
	}

	// 设置元素类型
	elemType := &core.TypeInfo{
		TypeName: "TestStruct",
		Type:     reflect.TypeOf(TestStruct{}),
	}
	pointerType.Type = reflect.PtrTo(elemType.Type)

	result, err := converter.ConvertType(pointerType)
	if err != nil {
		t.Fatalf("ConvertType failed: %v", err)
	}

	if result != "*TestStruct" {
		t.Errorf("expected *TestStruct, got %s", result)
	}
}

// TestTypeConverter_SliceTypes 测试切片类型转换
func TestTypeConverter_SliceTypes(t *testing.T) {
	ctx := core.NewGeneratorContext(&core.GenOptions{})
	converter := NewTypeConverter(ctx)

	// 创建切片类型
	sliceType := &core.TypeInfo{
		TypeName: "[]string",
		Type:     reflect.TypeOf([]string{}),
	}

	// 设置元素类型
	elemType := &core.TypeInfo{
		TypeName: "string",
		Type:     reflect.TypeOf(""),
	}
	sliceType.Type = reflect.SliceOf(elemType.Type)

	result, err := converter.ConvertType(sliceType)
	if err != nil {
		t.Fatalf("ConvertType failed: %v", err)
	}

	if result != "[]string" {
		t.Errorf("expected []string, got %s", result)
	}
}

// TestTypeConverter_MapTypes 测试映射类型转换
func TestTypeConverter_MapTypes(t *testing.T) {
	ctx := core.NewGeneratorContext(&core.GenOptions{})
	converter := NewTypeConverter(ctx)

	// 创建映射类型
	mapType := &core.TypeInfo{
		TypeName: "map[string]int",
		Type:     reflect.TypeOf(map[string]int{}),
	}

	// 设置键类型和值类型
	keyType := &core.TypeInfo{
		TypeName: "string",
		Type:     reflect.TypeOf(""),
	}
	elemType := &core.TypeInfo{
		TypeName: "int",
		Type:     reflect.TypeOf(0),
	}
	mapType.Type = reflect.MapOf(keyType.Type, elemType.Type)

	result, err := converter.ConvertType(mapType)
	if err != nil {
		t.Fatalf("ConvertType failed: %v", err)
	}

	if result != "map[string]int" {
		t.Errorf("expected map[string]int, got %s", result)
	}
}

// TestTypeConverter_Parameters 测试参数转换
func TestTypeConverter_Parameters(t *testing.T) {
	ctx := core.NewGeneratorContext(&core.GenOptions{})
	converter := NewTypeConverter(ctx)

	// 创建参数
	param := core.ParameterInfo{
		Name: "name",
		Type: &core.TypeInfo{
			TypeName: "string",
			Type:     reflect.TypeOf(""),
		},
		Index: 0,
	}

	result, err := converter.ConvertParameter(param)
	if err != nil {
		t.Fatalf("ConvertParameter failed: %v", err)
	}

	if result != "string" {
		t.Errorf("expected string, got %s", result)
	}
}

// TestTypeConverter_Returns 测试返回值转换
func TestTypeConverter_Returns(t *testing.T) {
	ctx := core.NewGeneratorContext(&core.GenOptions{})
	converter := NewTypeConverter(ctx)

	// 创建返回值
	ret := core.TypeInfo{
		TypeName: "bool",
		Type:     reflect.TypeOf(false),
	}

	result, err := converter.ConvertReturn(ret)
	if err != nil {
		t.Fatalf("ConvertReturn failed: %v", err)
	}

	if result != "bool" {
		t.Errorf("expected bool, got %s", result)
	}
}

// TestTypeConverter_Fields 测试字段转换
func TestTypeConverter_Fields(t *testing.T) {
	ctx := core.NewGeneratorContext(&core.GenOptions{})
	converter := NewTypeConverter(ctx)

	// 创建字段
	field := core.FieldInfo{
		Name: "Name",
		Type: &core.TypeInfo{
			TypeName: "string",
			Type:     reflect.TypeOf(""),
		},
		IsExported: true,
	}

	result, err := converter.ConvertField(field)
	if err != nil {
		t.Fatalf("ConvertField failed: %v", err)
	}

	if result != "string" {
		t.Errorf("expected string, got %s", result)
	}
}

// TestTypeConverter_Strategies 测试策略注册和获取
func TestTypeConverter_Strategies(t *testing.T) {
	ctx := core.NewGeneratorContext(&core.GenOptions{})
	converter := NewTypeConverter(ctx)

	// 测试策略获取
	strategy := converter.GetStrategy("basic")
	if strategy == nil {
		t.Error("expected basic strategy, got nil")
	}

	// 测试策略优先级
	if strategy.GetPriority() != 100 {
		t.Errorf("expected priority 100, got %d", strategy.GetPriority())
	}

	// 测试策略能力
	typeInfo := &core.TypeInfo{
		TypeName: "string",
		Type:     reflect.TypeOf(""),
	}

	if !strategy.CanConvert(typeInfo) {
		t.Error("basic strategy should be able to convert string type")
	}
}

// TestTypeConverter_ComplexTypes 测试复杂类型转换
func TestTypeConverter_ComplexTypes(t *testing.T) {
	ctx := core.NewGeneratorContext(&core.GenOptions{})
	converter := NewTypeConverter(ctx)

	// 创建配置管理器
	configManager := config.NewConfigManager(ctx.ErrorHandler)
	ctx.SetConfigManager(configManager)

	// 创建类型分析器
	typeAnalyzer := analyzer.NewTypeAnalyzer(ctx)

	// 分析复杂结构体
	complexType, err := typeAnalyzer.AnalyzeType(reflect.TypeOf(TestMapStruct{}))
	if err != nil {
		t.Fatalf("Failed to analyze TestMapStruct: %v", err)
	}

	// 转换复杂类型
	result, err := converter.ConvertType(complexType)
	if err != nil {
		t.Fatalf("ConvertType failed: %v", err)
	}

	// 应该是结构体名称
	if result != "origamisrc.TestMapStruct" {
		t.Errorf("expected origamisrc.TestMapStruct, got %s", result)
	}

	// 测试字段转换
	for _, field := range complexType.Fields {
		fieldResult, err := converter.ConvertField(core.FieldInfo{
			Name:       field.Name,
			Type:       field.Type,
			IsExported: field.IsExported,
		})
		if err != nil {
			t.Fatalf("ConvertField failed for field %s: %v", field.Name, err)
		}

		t.Logf("Field %s: %s", field.Name, fieldResult)
	}
}

// TestTypeConverter_ErrorHandling 测试错误处理
func TestTypeConverter_ErrorHandling(t *testing.T) {
	ctx := core.NewGeneratorContext(&core.GenOptions{})
	converter := NewTypeConverter(ctx)

	// 测试nil类型
	_, err := converter.ConvertType(nil)
	if err == nil {
		t.Error("expected error for nil type, got nil")
	}

	// 测试nil参数
	_, err = converter.ConvertParameter(core.ParameterInfo{Type: nil})
	if err == nil {
		t.Error("expected error for nil parameter type, got nil")
	}

	// 测试nil字段类型
	_, err = converter.ConvertField(core.FieldInfo{Type: nil})
	if err == nil {
		t.Error("expected error for nil field type, got nil")
	}

	// 测试获取nil类型的策略
	_, err = converter.GetConversionStrategy(nil)
	if err == nil {
		t.Error("expected error for nil type strategy, got nil")
	}
}
