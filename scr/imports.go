package scr

import (
	"fmt"
	"reflect"
	"strings"
)

// collectClassImports 收集类文件需要的导入包
func collectClassImports(srcPkgPath, pkgName string, methods map[string]reflect.Method, structType reflect.Type, fileCache *FileCache, config *Config) {
	// 添加源包导入
	if srcPkgPath != "" {
		fileCache.AddImport(srcPkgPath, pkgName+"src")
	}

	// 添加可能需要的导入（但不标记为使用，让代码生成时动态标记）
	fileCache.AddImport("github.com/php-any/origami/data", "data")
	fileCache.AddImport("github.com/php-any/origami/node", "node")
	fileCache.AddImport("github.com/php-any/origami/runtime", "runtime")
}

// collectMethodImportsToCache 收集方法文件需要的导入包到缓存
func collectMethodImportsToCache(srcPkgPath, pkgName string, paramTypes []reflect.Type, returnTypes []reflect.Type, fileCache *FileCache, config *Config) {
	// 添加源包导入
	if srcPkgPath != "" {
		fileCache.AddImport(srcPkgPath, pkgName+"src")
	}

	// 添加可能需要的导入（但不标记为使用，让代码生成时动态标记）
	fileCache.AddImport("github.com/php-any/origami/data", "data")
	fileCache.AddImport("github.com/php-any/origami/node", "node")
	fileCache.AddImport("fmt", "")
	fileCache.AddImport("github.com/php-any/generator/utils", "utils")
}

// collectMethodImports 收集方法文件需要的导入包（保留兼容性）
func collectMethodImports(srcPkgPath, pkgName string, paramTypes []reflect.Type, returnTypes []reflect.Type) map[string]string {
	imports := map[string]string{
		"github.com/php-any/origami/data": "data",
	}

	// 添加源包导入
	if srcPkgPath != "" {
		imports[srcPkgPath] = pkgName + "src"
	}

	// 收集参数和返回值类型的包
	allTypes := append(paramTypes, returnTypes...)
	for _, t := range allTypes {
		if t.PkgPath() != "" && t.PkgPath() != srcPkgPath {
			imports[t.PkgPath()] = pkgBaseName(t.PkgPath())
		}
	}

	return imports
}

// collectFunctionImportsToCache 收集函数文件需要的导入包到缓存
func collectFunctionImportsToCache(srcPkgPath, pkgName string, paramTypes []reflect.Type, returnTypes []reflect.Type, fileCache *FileCache, config *Config) {
	// 添加源包导入
	if srcPkgPath != "" {
		fileCache.AddImport(srcPkgPath, pkgName+"src")
	}

	// 添加可能需要的导入（但不标记为使用，让代码生成时动态标记）
	fileCache.AddImport("github.com/php-any/origami/data", "data")
	fileCache.AddImport("github.com/php-any/origami/node", "node")
	fileCache.AddImport("fmt", "")
	fileCache.AddImport("github.com/php-any/generator/utils", "utils")
}

// writeImportsFromCache 从缓存写入导入
func writeImportsFromCache(b *strings.Builder, fileCache *FileCache) {
	imports := fileCache.GetImports()
	if len(imports) == 0 {
		return
	}

	// 只写入实际使用的导入
	usedImports := make(map[string]string)
	for pkgPath, alias := range imports {
		if fileCache.ImportUsage[pkgPath] {
			usedImports[pkgPath] = alias
		}
	}

	if len(usedImports) == 0 {
		return
	}

	b.WriteString("import (\n")
	for pkgPath, alias := range usedImports {
		if alias == pkgBaseName(pkgPath) {
			fmt.Fprintf(b, "\t\"%s\"\n", pkgPath)
		} else {
			fmt.Fprintf(b, "\t%s \"%s\"\n", alias, pkgPath)
		}
	}
	b.WriteString(")\n\n")
}

// writeImports 写入导入
func writeImports(b *strings.Builder, imports map[string]string) {
	if len(imports) == 0 {
		return
	}

	b.WriteString("import (\n")
	for pkgPath, alias := range imports {
		if alias == pkgBaseName(pkgPath) {
			fmt.Fprintf(b, "\t\"%s\"\n", pkgPath)
		} else {
			fmt.Fprintf(b, "\t%s \"%s\"\n", alias, pkgPath)
		}
	}
	b.WriteString(")\n\n")
}

// collectStructFieldImports 收集结构体字段的导入
func collectStructFieldImports(structType reflect.Type, srcPkgPath string, fileCache *FileCache, config *Config) {
	for i := 0; i < structType.NumField(); i++ {
		field := structType.Field(i)
		collectTypeImports(field.Type, srcPkgPath, fileCache, config)
	}
}

// collectDirectTypeImports 收集直接类型的导入（不递归收集嵌套字段）
func collectDirectTypeImports(t reflect.Type, srcPkgPath string, fileCache *FileCache, config *Config) {
	if t == nil {
		return
	}

	switch t.Kind() {
	case reflect.Ptr:
		collectDirectTypeImports(t.Elem(), srcPkgPath, fileCache, config)
	case reflect.Slice, reflect.Array:
		collectDirectTypeImports(t.Elem(), srcPkgPath, fileCache, config)
	case reflect.Map:
		collectDirectTypeImports(t.Key(), srcPkgPath, fileCache, config)
		collectDirectTypeImports(t.Elem(), srcPkgPath, fileCache, config)
	case reflect.Chan:
		collectDirectTypeImports(t.Elem(), srcPkgPath, fileCache, config)
	case reflect.Struct, reflect.Interface:
		// 只收集直接类型，不递归收集字段
		if t.PkgPath() != "" && t.PkgPath() != srcPkgPath {
			// 检查是否在黑名单中
			if config != nil && !isBlacklistedPackage(t.PkgPath(), config) {
				fileCache.AddImport(t.PkgPath(), pkgBaseName(t.PkgPath()))
			}
		}
	default:
		// 基本类型不需要导入
	}
}

// collectTypeImports 收集类型的导入（递归收集所有依赖）
func collectTypeImports(t reflect.Type, srcPkgPath string, fileCache *FileCache, config *Config) {
	visited := make(map[reflect.Type]bool)
	collectTypeImportsWithVisited(t, srcPkgPath, fileCache, config, visited)
}

// collectTypeImportsWithVisited 带访问记录的递归收集函数
func collectTypeImportsWithVisited(t reflect.Type, srcPkgPath string, fileCache *FileCache, config *Config, visited map[reflect.Type]bool) {
	if t == nil {
		return
	}

	// 防止循环引用
	if visited[t] {
		return
	}
	visited[t] = true

	switch t.Kind() {
	case reflect.Ptr:
		collectTypeImportsWithVisited(t.Elem(), srcPkgPath, fileCache, config, visited)
	case reflect.Slice, reflect.Array:
		collectTypeImportsWithVisited(t.Elem(), srcPkgPath, fileCache, config, visited)
	case reflect.Map:
		collectTypeImportsWithVisited(t.Key(), srcPkgPath, fileCache, config, visited)
		collectTypeImportsWithVisited(t.Elem(), srcPkgPath, fileCache, config, visited)
	case reflect.Chan:
		collectTypeImportsWithVisited(t.Elem(), srcPkgPath, fileCache, config, visited)
	case reflect.Struct, reflect.Interface:
		if t.PkgPath() != "" && t.PkgPath() != srcPkgPath {
			// 检查是否在黑名单中
			if config != nil && !isBlacklistedPackage(t.PkgPath(), config) {
				fileCache.AddImport(t.PkgPath(), pkgBaseName(t.PkgPath()))
			}
		}
		// 对于结构体，递归收集字段类型
		if t.Kind() == reflect.Struct {
			for i := 0; i < t.NumField(); i++ {
				collectTypeImportsWithVisited(t.Field(i).Type, srcPkgPath, fileCache, config, visited)
			}
		}
	default:
		// 基本类型不需要导入
	}
}

// isBlacklistedPackage 检查包是否在黑名单中
func isBlacklistedPackage(pkgPath string, config *Config) bool {
	if config == nil || config.Blacklist.Packages == nil {
		return false
	}

	for _, blacklistedPkg := range config.Blacklist.Packages {
		if pkgPath == blacklistedPkg {
			return true
		}
	}

	return false
}

// isStructType 检查类型是否为结构体类型
func isStructType(t reflect.Type) bool {
	for t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	return t.Kind() == reflect.Struct
}
