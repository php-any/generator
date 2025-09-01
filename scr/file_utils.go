package scr

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
)

// FileUtils 文件工具模块，负责文件名生成和文件操作

// generateExpectedFileName 根据类型信息生成预期的文件名
//
// 参数说明：
// - t: 反射类型，可以是结构体、指针或函数类型
// - cache: 生成缓存，包含配置信息
//
// 返回值：
// - 预期的文件路径，如果无法生成则返回空字符串
//
// 生成规则：
// - 结构体类型：生成 {outputRoot}/{pkgName}/{typeName}_class.go
// - 指针类型：如果指向结构体，生成对应的类文件
// - 函数类型：暂时返回空，需要更多上下文信息
func generateExpectedFileName(t reflect.Type, cache *GroupCache) string {
	if cache.Config == nil {
		return ""
	}

	// 获取包名和类型名
	var pkgName, typeName string

	switch t.Kind() {
	case reflect.Struct:
		// 结构体类型：直接获取包路径和类型名
		pkgName = pkgBaseName(t.PkgPath())
		typeName = t.Name()
	case reflect.Ptr:
		// 指针类型：检查是否指向结构体
		if t.Elem() != nil && t.Elem().Kind() == reflect.Struct {
			pkgName = pkgBaseName(t.Elem().PkgPath())
			typeName = t.Elem().Name()
		}
	case reflect.Func:
		// 函数类型：根据函数签名生成文件名
		// 尝试从返回值类型推断包名和函数名
		if t.NumOut() > 0 {
			returnType := t.Out(0)
			if returnType.Kind() == reflect.Ptr && returnType.Elem().Kind() == reflect.Struct {
				pkgName = pkgBaseName(returnType.Elem().PkgPath())
				typeName = returnType.Elem().Name()
			} else if returnType.Kind() == reflect.Struct {
				pkgName = pkgBaseName(returnType.PkgPath())
				typeName = returnType.Name()
			}
		}

		// 如果无法从返回值推断，尝试从参数推断
		if pkgName == "" || typeName == "" {
			if t.NumIn() > 0 {
				paramType := t.In(0)
				if paramType.Kind() == reflect.Ptr && paramType.Elem().Kind() == reflect.Struct {
					pkgName = pkgBaseName(paramType.Elem().PkgPath())
					typeName = paramType.Elem().Name()
				} else if paramType.Kind() == reflect.Struct {
					pkgName = pkgBaseName(paramType.PkgPath())
					typeName = paramType.Name()
				}
			}
		}

		// 如果仍然无法推断，生成一个基于函数签名的文件名
		if pkgName == "" || typeName == "" {
			// 生成一个描述性的函数名
			var nameParts []string

			// 添加参数类型信息
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

			// 添加返回值类型信息
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

			// 使用第一个有意义的类型名
			for _, part := range nameParts {
				if part != "" && part != "struct" && part != "interface" {
					typeName = part
					break
				}
			}

			// 如果仍然没有有效的类型名，使用默认值
			if typeName == "" {
				typeName = "function"
			}

			// 尝试从参数或返回值中获取包名
			if t.NumIn() > 0 {
				if pkgPath := t.In(0).PkgPath(); pkgPath != "" {
					pkgName = pkgBaseName(pkgPath)
				} else if t.In(0).Kind() == reflect.Ptr && t.In(0).Elem() != nil {
					if pkgPath := t.In(0).Elem().PkgPath(); pkgPath != "" {
						pkgName = pkgBaseName(pkgPath)
					}
				}
			}

			if t.NumOut() > 0 {
				if pkgName == "" {
					if pkgPath := t.Out(0).PkgPath(); pkgPath != "" {
						pkgName = pkgBaseName(pkgPath)
					} else if t.Out(0).Kind() == reflect.Ptr && t.Out(0).Elem() != nil {
						if pkgPath := t.Out(0).Elem().PkgPath(); pkgPath != "" {
							pkgName = pkgBaseName(pkgPath)
						}
					}
				}
			}

			// 如果仍然没有包名，使用默认值
			if pkgName == "" {
				pkgName = "main"
			}
		}
	default:
		// 其他类型：不支持
		return ""
	}

	// 验证包名和类型名是否有效
	if pkgName == "" || typeName == "" {
		return ""
	}

	// 生成预期的文件名
	var fileName string
	if t.Kind() == reflect.Func {
		// 函数类型：生成 {typeName}_func.go
		fileName = strings.ToLower(typeName) + "_func.go"
	} else {
		// 结构体类型：生成 {typeName}_class.go
		fileName = strings.ToLower(typeName) + "_class.go"
	}
	return filepath.Join(cache.Config.OutputRoot, pkgName, fileName)
}

// copyFile 复制文件
//
// 参数说明：
// - src: 源文件路径
// - dst: 目标文件路径
//
// 返回值：
// - 错误信息，成功时返回 nil
//
// 功能：
// - 读取源文件内容
// - 写入到目标文件
// - 使用 0644 权限（用户读写，组和其他用户只读）
func copyFile(src, dst string) error {
	// 读取源文件
	data, err := os.ReadFile(src)
	if err != nil {
		return err
	}

	// 写入目标文件
	return os.WriteFile(dst, data, 0644)
}

// pkgBaseName 从完整包路径提取最终包名
//
// 参数说明：
// - pkgPath: 完整的包路径，如 "github.com/user/project/pkg"
//
// 返回值：
// - 包的基础名称，如 "pkg"
//
// 处理规则：
// - 空路径返回 "main"
// - 使用 filepath.Base 提取最后一段
// - 处理特殊情况（如 "." 或路径分隔符）
func pkgBaseName(pkgPath string) string {
	if pkgPath == "" {
		return "main"
	}
	base := filepath.Base(pkgPath)
	if base == "." || base == string(filepath.Separator) {
		return pkgPath
	}
	return base
}
