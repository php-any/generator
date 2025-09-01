package scr

import (
	"errors"
	"os"
	"path/filepath"
	"reflect"
)

func GenerateFromAny(a any, config *Config) error {
	t := reflect.TypeOf(a)
	if t == nil {
		return errors.New("输入为 nil，不支持")
	}
	cache := NewGroupCache(config)
	return generateFromType(t, cache, a)
}

func generateFromType(t reflect.Type, cache *GroupCache, originalValue any) error {
	// 检查深度限制
	if cache.Config != nil && cache.Config.MaxDepth > 0 && cache.CurrentDepth >= cache.Config.MaxDepth {
		return nil
	}

	// 检查缓存，防止重复生成和死循环
	typeKey := t.String()
	if cache.IsTypeGenerated(typeKey) {
		return nil
	}

	// 标记为已生成
	cache.MarkTypeGenerated(typeKey)

	// 提前检查需要生成的文件名，是否是直接替换的
	// 根据类型信息生成预期的文件名
	expectedFile := generateExpectedFileName(t, cache)
	if expectedFile != "" {
		if replacementFile, ok := cache.Config.FixedReplace[expectedFile]; ok {
			// 确保目标目录存在
			dir := filepath.Dir(expectedFile)
			if err := os.MkdirAll(dir, 0755); err != nil {
				panic(err)
			}
			// 复制替换文件到目标位置
			if err := copyFile(replacementFile, expectedFile); err != nil {
				panic(err)
			}
			return nil // 文件已替换，不需要继续生成
		}
	}

	// 仅跳过空接口 interface{}，其余接口允许继续进入生成流程
	if t.Kind() == reflect.Interface && t.PkgPath() == "" && t.Name() == "" {
		return nil
	}

	// 跳过小写开头的类型（私有类型）
	if t.Name() != "" && !IsExportedType(t.Name()) {
		return nil
	}

	switch parseTypes(t) {
	case "class":
		return buildClass(t, cache, cache.Config)
	case "func":
		return buildFunc(t, cache, originalValue)
	}

	return nil
}

// IsExportedType 检查类型名是否为导出的（大写开头）
func IsExportedType(typeName string) bool {
	if typeName == "" {
		return false
	}
	firstChar := typeName[0]
	return firstChar >= 'A' && firstChar <= 'Z'
}
