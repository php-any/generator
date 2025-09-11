package scr

import (
	"fmt"
	"path/filepath"
	"reflect"
	"strings"
)

func buildClass(t reflect.Type, cache *GroupCache, config *Config) error {
	// 验证和预处理类型（支持 struct/interface）
	structType, err := validateAndPrepareStructType(t)
	if err != nil {
		panic(err)
	}

	// 收集导出方法（支持 struct/interface）
	allMethods := collectExportedMethods(structType)

	// 检查方法的递归生成
	checkMethodsRecursiveGeneration(allMethods, cache)

	// 接口无字段，跳过字段递归检查
	if structType.Kind() != reflect.Interface {
		checkFieldsRecursiveGeneration(structType, cache)
	}

	// 生成类文件
	if err := generateClassFile(structType, allMethods, cache); err != nil {
		panic(err)
	}

	// 生成方法文件
	if err := generateMethodFiles(structType, allMethods, cache); err != nil {
		panic(err)
	}

	// 注册类并生成 load.go
	globalCache.RegisterClass(pkgBaseName(structType.PkgPath()), structType.Name())
	if err := emitLoadFile(pkgBaseName(structType.PkgPath()), cache); err != nil {
		panic(err)
	}

	return nil
}

// validateAndPrepareStructType 验证和预处理结构体/接口类型
func validateAndPrepareStructType(t reflect.Type) (reflect.Type, error) {
	// 处理指针类型
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}

	switch t.Kind() {
	case reflect.Struct:
		return t, nil
	case reflect.Interface:
		// 接口类型直接返回
		return t, nil
	default:
		return nil, fmt.Errorf("期望结构体或接口类型，实际: %s", t.String())
	}
}

// collectExportedMethods 收集导出方法（支持 struct/interface）
func collectExportedMethods(structType reflect.Type) map[string]reflect.Method {
	allMethods := map[string]reflect.Method{}

	if structType.Kind() == reflect.Interface {
		for i := 0; i < structType.NumMethod(); i++ {
			m := structType.Method(i)
			if m.PkgPath == "" && isExportedName(m.Name) {
				allMethods[m.Name] = m
			}
		}
		return allMethods
	}

	ptrType := reflect.PointerTo(structType)
	for i := 0; i < ptrType.NumMethod(); i++ {
		m := ptrType.Method(i)
		// 仅导出方法
		if m.PkgPath == "" && isExportedName(m.Name) {
			allMethods[m.Name] = m
		}
	}

	return allMethods
}

// checkMethodsRecursiveGeneration 检查方法的递归生成
func checkMethodsRecursiveGeneration(allMethods map[string]reflect.Method, cache *GroupCache) {
	for _, m := range allMethods {
		checkMethodRecursiveGeneration(m, cache)
	}
}

// checkFieldsRecursiveGeneration 检查字段的递归生成（仅结构体）
func checkFieldsRecursiveGeneration(structType reflect.Type, cache *GroupCache) {
	for i := 0; i < structType.NumField(); i++ {
		field := structType.Field(i)
		if field.PkgPath != "" { // 非导出字段跳过
			continue
		}
		checkFieldRecursiveGeneration(field, cache)
	}
}

// checkFieldRecursiveGeneration 检查单个字段的递归生成
func checkFieldRecursiveGeneration(field reflect.StructField, cache *GroupCache) {
	fieldType := field.Type

	// 接口类型
	if fieldType.Kind() == reflect.Interface && fieldType.PkgPath() != "" && fieldType.Name() != "" {
		_ = generateFromType(fieldType, cache, nil)
		return
	}

	// *struct 类型
	if isPtrToStruct(fieldType) {
		_ = generateFromType(fieldType, cache, nil)
		return
	}

	// 值 struct 类型
	if fieldType.Kind() == reflect.Struct && fieldType.PkgPath() != "" && fieldType.Name() != "" {
		_ = generateFromType(reflect.PointerTo(fieldType), cache, nil)
		return
	}
}

// generateClassFile 生成类文件
func generateClassFile(structType reflect.Type, allMethods map[string]reflect.Method, cache *GroupCache) error {
	srcPkgPath := structType.PkgPath()
	pkgName := pkgBaseName(srcPkgPath)
	typeName := structType.Name()

	// 生成类文件路径
	outDir := filepath.Join(cache.Config.OutputRoot, pkgName)
	classFile := filepath.Join(outDir, strings.ToLower(typeName)+"_class.go")

	// 创建文件缓存
	fileCache := NewFileCache()

	// 构建类文件内容
	classBody := buildClassFileBody(srcPkgPath, pkgName, typeName, allMethods, structType, pkgName, fileCache, cache.Config)

	// 输出文件
	return emitFile(classFile, pkgName, classBody)
}

// generateMethodFiles 生成方法文件
func generateMethodFiles(structType reflect.Type, allMethods map[string]reflect.Method, cache *GroupCache) error {
	srcPkgPath := structType.PkgPath()
	pkgName := pkgBaseName(srcPkgPath)
	typeName := structType.Name()

	// 生成方法文件路径
	outDir := filepath.Join(cache.Config.OutputRoot, pkgName)

	// 结构体方法使用指针接收者；接口方法没有接收者
	sourceIsPtr := structType.Kind() == reflect.Struct
	// 仅为冲突消解后的选中方法生成文件
	selected := buildMethodFieldMapping(allMethods)
	for _, chosenName := range selected {
		method := allMethods[chosenName]
		methodFile := filepath.Join(outDir, strings.ToLower(typeName)+"_"+strings.ToLower(chosenName)+"_method.go")

		// 创建文件缓存
		fileCache := NewFileCache()

		// 构建方法文件内容
		methodBody, ok := buildMethodFileBody(srcPkgPath, pkgName, typeName, method, sourceIsPtr, fileCache, structType, cache.Config)
		if !ok {
			continue
		}

		// 输出文件
		if err := emitFile(methodFile, pkgName, methodBody); err != nil {
			return err
		}
	}

	return nil
}
