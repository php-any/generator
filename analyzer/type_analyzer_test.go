package analyzer

import (
	"reflect"
	"testing"

	"github.com/php-any/generator/config"
	"github.com/php-any/generator/core"
)

// 测试用的简单结构体
type TestStruct struct {
	Name   string
	Age    int
	Active bool
}

// TestTypeAnalyzer_AnalyzeType 测试类型分析
func TestTypeAnalyzer_AnalyzeType(t *testing.T) {
	// 创建上下文
	ctx := core.NewGeneratorContext(&core.GenOptions{
		OutputRoot: "test_output",
		MaxDepth:   3,
	})

	// 创建配置管理器
	configManager := config.NewConfigManager(ctx.ErrorHandler)
	ctx.SetConfigManager(configManager)

	// 创建类型分析器
	analyzer := NewTypeAnalyzer(ctx)

	// 测试基本类型
	t.Run("BasicType", func(t *testing.T) {
		info, err := analyzer.AnalyzeType(reflect.TypeOf(""))
		if err != nil {
			t.Fatalf("Failed to analyze string type: %v", err)
		}

		if info == nil {
			t.Fatal("TypeInfo should not be nil")
		}

		if info.TypeName != "string" {
			t.Errorf("Expected TypeName to be 'string', got '%s'", info.TypeName)
		}

		if info.IsBasicType() {
			t.Log("String type correctly identified as basic type")
		} else {
			t.Error("String type should be identified as basic type")
		}
	})

	// 测试结构体类型
	t.Run("StructType", func(t *testing.T) {
		info, err := analyzer.AnalyzeType(reflect.TypeOf(TestStruct{}))
		if err != nil {
			t.Fatalf("Failed to analyze struct type: %v", err)
		}

		if info == nil {
			t.Fatal("TypeInfo should not be nil")
		}

		if info.TypeName != "TestStruct" {
			t.Errorf("Expected TypeName to be 'TestStruct', got '%s'", info.TypeName)
		}

		if !info.IsStruct {
			t.Error("TestStruct should be identified as struct")
		}

		// 检查字段
		if len(info.Fields) != 3 {
			t.Errorf("Expected 3 fields, got %d", len(info.Fields))
		}

		// 检查字段名称
		expectedFields := map[string]bool{"Name": true, "Age": true, "Active": true}
		for _, field := range info.Fields {
			if !expectedFields[field.Name] {
				t.Errorf("Unexpected field name: %s", field.Name)
			}
		}
	})

	// 测试指针类型
	t.Run("PointerType", func(t *testing.T) {
		info, err := analyzer.AnalyzeType(reflect.TypeOf(&TestStruct{}))
		if err != nil {
			t.Fatalf("Failed to analyze pointer type: %v", err)
		}

		if info == nil {
			t.Fatal("TypeInfo should not be nil")
		}

		if !info.IsPointer {
			t.Error("Pointer to TestStruct should be identified as pointer")
		}
	})

	// 测试切片类型
	t.Run("SliceType", func(t *testing.T) {
		info, err := analyzer.AnalyzeType(reflect.TypeOf([]string{}))
		if err != nil {
			t.Fatalf("Failed to analyze slice type: %v", err)
		}

		if info == nil {
			t.Fatal("TypeInfo should not be nil")
		}

		if !info.IsSliceType() {
			t.Error("[]string should be identified as slice type")
		}

		// 检查元素类型
		elemType := info.GetElementType()
		if elemType == nil {
			t.Fatal("Element type should not be nil for slice")
		}

		if elemType.TypeName != "string" {
			t.Errorf("Expected element type to be 'string', got '%s'", elemType.TypeName)
		}
	})

	// 测试映射类型
	t.Run("MapType", func(t *testing.T) {
		info, err := analyzer.AnalyzeType(reflect.TypeOf(map[string]int{}))
		if err != nil {
			t.Fatalf("Failed to analyze map type: %v", err)
		}

		if info == nil {
			t.Fatal("TypeInfo should not be nil")
		}

		if !info.IsMapType() {
			t.Error("map[string]int should be identified as map type")
		}

		// 检查键类型
		keyType := info.GetKeyType()
		if keyType == nil {
			t.Fatal("Key type should not be nil for map")
		}

		if keyType.TypeName != "string" {
			t.Errorf("Expected key type to be 'string', got '%s'", keyType.TypeName)
		}

		// 检查值类型
		elemType := info.GetElementType()
		if elemType == nil {
			t.Fatal("Element type should not be nil for map")
		}

		if elemType.TypeName != "int" {
			t.Errorf("Expected element type to be 'int', got '%s'", elemType.TypeName)
		}
	})
}

// TestTypeAnalyzer_AnalyzeFunction 测试函数分析
func TestTypeAnalyzer_AnalyzeFunction(t *testing.T) {
	// 创建上下文和分析器
	ctx := core.NewGeneratorContext(&core.GenOptions{})
	configManager := config.NewConfigManager(ctx.ErrorHandler)
	ctx.SetConfigManager(configManager)
	analyzer := NewTypeAnalyzer(ctx)

	// 测试函数
	testFunc := func(name string, age int) (bool, error) {
		return age > 18, nil
	}

	info, err := analyzer.AnalyzeFunction(testFunc)
	if err != nil {
		t.Fatalf("Failed to analyze function: %v", err)
	}

	if info == nil {
		t.Fatal("FunctionInfo should not be nil")
	}

	// 检查参数
	if len(info.Parameters) != 2 {
		t.Errorf("Expected 2 parameters, got %d", len(info.Parameters))
	}

	// 检查返回值
	if len(info.Returns) != 2 {
		t.Errorf("Expected 2 return values, got %d", len(info.Returns))
	}

	// 检查参数类型
	if info.Parameters[0].Type.TypeName != "string" {
		t.Errorf("Expected first parameter type to be 'string', got '%s'", info.Parameters[0].Type.TypeName)
	}

	if info.Parameters[1].Type.TypeName != "int" {
		t.Errorf("Expected second parameter type to be 'int', got '%s'", info.Parameters[1].Type.TypeName)
	}

	// 检查返回值类型
	if info.Returns[0].TypeName != "bool" {
		t.Errorf("Expected first return type to be 'bool', got '%s'", info.Returns[0].TypeName)
	}

	if info.Returns[1].TypeName != "error" {
		t.Errorf("Expected second return type to be 'error', got '%s'", info.Returns[1].TypeName)
	}
}

// TestTypeAnalyzer_Cache 测试缓存功能
func TestTypeAnalyzer_Cache(t *testing.T) {
	ctx := core.NewGeneratorContext(&core.GenOptions{})
	configManager := config.NewConfigManager(ctx.ErrorHandler)
	ctx.SetConfigManager(configManager)
	analyzer := NewTypeAnalyzer(ctx)

	// 第一次分析
	info1, err := analyzer.AnalyzeType(reflect.TypeOf(TestStruct{}))
	if err != nil {
		t.Fatalf("Failed to analyze type first time: %v", err)
	}

	// 第二次分析（应该从缓存获取）
	info2, err := analyzer.AnalyzeType(reflect.TypeOf(TestStruct{}))
	if err != nil {
		t.Fatalf("Failed to analyze type second time: %v", err)
	}

	// 检查是否是同一个实例
	if info1 != info2 {
		t.Error("Cached type info should be the same instance")
	}

	// 检查缓存命中
	metrics := ctx.Metrics.GetMetrics()
	if metrics.CacheHits == 0 {
		t.Error("Cache hits should be greater than 0")
	}
}

// TestTypeAnalyzer_Blacklist 测试黑名单功能
func TestTypeAnalyzer_Blacklist(t *testing.T) {
	ctx := core.NewGeneratorContext(&core.GenOptions{})
	configManager := config.NewConfigManager(ctx.ErrorHandler)
	ctx.SetConfigManager(configManager)

	// 设置黑名单
	config := configManager.GetConfig()
	config.Blacklist.Types = append(config.Blacklist.Types, "TestStruct")

	analyzer := NewTypeAnalyzer(ctx)

	// 尝试分析黑名单中的类型
	_, err := analyzer.AnalyzeType(reflect.TypeOf(TestStruct{}))
	if err == nil {
		t.Error("Should reject blacklisted type")
	}

	// 检查错误类型
	if _, ok := err.(*core.GeneratorError); !ok {
		t.Error("Error should be of type GeneratorError")
	}
}
