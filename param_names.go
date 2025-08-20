package generator

import (
	"go/ast"
	"go/parser"
	"go/token"
	"path/filepath"
	"runtime"
	"strings"
)

// tryExtractParamNames 通过 PC 解析源文件，从 FuncDecl 中提取参数名
// 返回值：names, ok
func tryExtractParamNames(pc uintptr, expectedNumIn int) ([]string, bool) {
	f := runtime.FuncForPC(pc)
	if f == nil {
		return nil, false
	}
	file, _ := f.FileLine(pc)
	if file == "" {
		return nil, false
	}
	// 解析文件
	fset := token.NewFileSet()
	astFile, err := parser.ParseFile(fset, file, nil, 0)
	if err != nil {
		return nil, false
	}
	// 提取简单函数名（去包路径）
	fullName := f.Name()
	simpleName := fullName
	if idx := strings.LastIndex(simpleName, "."); idx >= 0 {
		simpleName = simpleName[idx+1:]
	}
	// 方法与函数同名不会重载，这里按名字匹配
	var params []string
	ast.Inspect(astFile, func(n ast.Node) bool {
		fd, ok := n.(*ast.FuncDecl)
		if !ok {
			return true
		}
		if fd.Name == nil || fd.Name.Name != simpleName {
			return true
		}
		// 统计形参个数（不包含接收者）
		count := 0
		if fd.Type.Params != nil {
			for _, f := range fd.Type.Params.List {
				// 多个名称共享一个类型时 Names 可能有多个
				if len(f.Names) == 0 {
					count++
				} else {
					count += len(f.Names)
				}
			}
		}
		if count != expectedNumIn {
			return true
		}
		// 收集名称
		tmp := make([]string, 0, count)
		if fd.Type.Params != nil {
			for _, f := range fd.Type.Params.List {
				if len(f.Names) == 0 {
					tmp = append(tmp, "param")
					continue
				}
				for _, n := range f.Names {
					tmp = append(tmp, n.Name)
				}
			}
		}
		params = tmp
		return false
	})
	if len(params) == expectedNumIn {
		return params, true
	}
	return nil, false
}

// cleanFilepath 统一路径分隔符（防御性，当前未使用）
func cleanFilepath(p string) string {
	return filepath.Clean(p)
}
