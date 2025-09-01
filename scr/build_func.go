package scr

import (
	"fmt"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
)

func buildFunc(t reflect.Type, cache *GroupCache, originalValue any) error {
	if t.Kind() != reflect.Func {
		panic(fmt.Errorf("期望函数类型，实际: %s", t.String()))
	}

	// 检查函数参数和返回值，看是否需要生成代理类
	checkFunctionRecursiveGeneration(t, cache)

	// 生成函数文件
	if err := generateFunctionFile(t, cache, originalValue); err != nil {
		panic(err)
	}

	// 注册函数并生成 load.go
	funcName, err := getFunctionName(t, originalValue)
	if err != nil {
		panic(err)
	}
	if funcName != "" {
		pkgName := getFunctionPackageName(t, originalValue)
		globalCache.RegisterFunction(pkgName, funcName)
		if err := emitLoadFile(pkgName, cache); err != nil {
			panic(err)
		}
	}

	return nil
}

// generateFunctionFile 生成函数文件
func generateFunctionFile(t reflect.Type, cache *GroupCache, originalValue any) error {
	funcName, err := getFunctionName(t, originalValue)
	if err != nil {
		panic(err)
	}

	// 获取包信息
	pkgName := getFunctionPackageName(t, originalValue)
	srcPkgPath := getFunctionPackagePath(t)
	namePrefix := cache.Config.NamePrefix

	// 生成函数文件路径
	outDir := filepath.Join(cache.Config.OutputRoot, pkgName)
	funcFile := filepath.Join(outDir, strings.ToLower(funcName)+"_func.go")

	// 创建文件缓存
	fileCache := NewFileCache()

	// 构建函数文件内容（传入源包路径以保证 import alias 一致）
	funcBody := buildFunctionFileBody(srcPkgPath, pkgName, namePrefix, funcName, t, fileCache, cache.Config)

	// 输出文件
	return emitFile(funcFile, pkgName, funcBody)
}

// getFunctionPackageName 从函数类型推断包名
func getFunctionPackageName(t reflect.Type, originalValue any) string {
	// 先用函数类型自身的包路径（命名函数类型）
	if t.PkgPath() != "" {
		return pkgBaseName(t.PkgPath())
	}

	// 尝试从 originalValue 中获取包信息
	if originalValue != nil {
		if f := runtime.FuncForPC(reflect.ValueOf(originalValue).Pointer()); f != nil {
			name := f.Name()
			if idx := strings.LastIndex(name, "."); idx >= 0 {
				// 提取包名（去掉函数名部分）
				pkgPath := name[:idx]
				// 处理嵌套包的情况，取最后一部分作为包名
				if lastDot := strings.LastIndex(pkgPath, "/"); lastDot >= 0 {
					return pkgPath[lastDot+1:]
				}
				return pkgPath
			}
		}
	}

	// 尝试从参数和返回值中获取包信息
	for i := 0; i < t.NumIn(); i++ {
		paramType := t.In(i)
		if pkgPath := paramType.PkgPath(); pkgPath != "" {
			return pkgBaseName(pkgPath)
		}
		// 检查指针类型的元素
		if paramType.Kind() == reflect.Ptr && paramType.Elem() != nil {
			if pkgPath := paramType.Elem().PkgPath(); pkgPath != "" {
				return pkgBaseName(pkgPath)
			}
		}
	}

	for i := 0; i < t.NumOut(); i++ {
		returnType := t.Out(i)
		if pkgPath := returnType.PkgPath(); pkgPath != "" {
			return pkgBaseName(pkgPath)
		}
		// 检查指针类型的元素
		if returnType.Kind() == reflect.Ptr && returnType.Elem() != nil {
			if pkgPath := returnType.Elem().PkgPath(); pkgPath != "" {
				return pkgBaseName(pkgPath)
			}
		}
	}

	// 如果无法推断，直接抛出错误
	panic(fmt.Sprintf("无法从函数类型 %s 推断包名", t.String()))
}

// getFunctionPackagePath 从函数类型推断完整包路径
func getFunctionPackagePath(t reflect.Type) string {
	if t.PkgPath() != "" {
		return t.PkgPath()
	}
	for i := 0; i < t.NumIn(); i++ {
		paramType := t.In(i)
		if pkgPath := paramType.PkgPath(); pkgPath != "" {
			return pkgPath
		}
		if paramType.Kind() == reflect.Ptr && paramType.Elem() != nil {
			if pkgPath := paramType.Elem().PkgPath(); pkgPath != "" {
				return pkgPath
			}
		}
	}
	for i := 0; i < t.NumOut(); i++ {
		returnType := t.Out(i)
		if pkgPath := returnType.PkgPath(); pkgPath != "" {
			return pkgPath
		}
		if returnType.Kind() == reflect.Ptr && returnType.Elem() != nil {
			if pkgPath := returnType.Elem().PkgPath(); pkgPath != "" {
				return pkgPath
			}
		}
	}
	return ""
}

// getFunctionName 获取函数名（优先真实函数名，回退到类型/签名推断）
func getFunctionName(t reflect.Type, originalValue any) (string, error) {
	// 优先使用 runtime.FuncForPC 获取真实函数名
	if originalValue != nil {
		if f := runtime.FuncForPC(reflect.ValueOf(originalValue).Pointer()); f != nil {
			name := f.Name()
			if idx := strings.LastIndex(name, "."); idx >= 0 {
				return name[idx+1:], nil
			}
			return name, nil
		}
	}
	// 命名函数类型（如 auth.UnsubscribeFunc）
	if t.Name() != "" {
		return t.Name(), nil
	}

	// 回退：从返回值类型推断
	if t.NumOut() > 0 {
		returnType := t.Out(0)
		if returnType.Kind() == reflect.Ptr && returnType.Elem().Kind() == reflect.Struct {
			return returnType.Elem().Name(), nil
		}
	}

	// 回退：从参数类型推断
	if t.NumIn() > 0 {
		paramType := t.In(0)
		if paramType.Kind() == reflect.Ptr && paramType.Elem().Kind() == reflect.Struct {
			return paramType.Elem().Name(), nil
		}
	}

	// 回退：基于签名拼接
	var nameParts []string
	for i := 0; i < t.NumIn(); i++ {
		paramType := t.In(i)
		if paramType.Kind() == reflect.Ptr && paramType.Elem().Kind() == reflect.Struct {
			nameParts = append(nameParts, paramType.Elem().Name())
		} else if paramType.Kind() == reflect.Struct {
			nameParts = append(nameParts, paramType.Name())
		} else {
			nameParts = append(nameParts, paramType.Kind().String())
		}
	}
	for i := 0; i < t.NumOut(); i++ {
		returnType := t.Out(i)
		if returnType.Kind() == reflect.Ptr && returnType.Elem().Kind() == reflect.Struct {
			nameParts = append(nameParts, returnType.Elem().Name())
		} else if returnType.Kind() == reflect.Struct {
			nameParts = append(nameParts, returnType.Name())
		} else {
			nameParts = append(nameParts, returnType.Kind().String())
		}
	}
	if len(nameParts) > 0 {
		for _, part := range nameParts {
			if part != "" && part != "struct" && part != "interface" {
				return part, nil
			}
		}
	}
	return "", fmt.Errorf("无法从函数类型 %s 推断函数名", t.String())
}
