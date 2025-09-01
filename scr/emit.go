package scr

import (
	"bytes"
	"go/format"
	"os"
	"path/filepath"
)

// Emit 文件输出模块

// emitFile 生成文件，自动 gofmt
func emitFile(targetPath string, pkg string, body string) error {
	var buf bytes.Buffer
	buf.WriteString("package ")
	buf.WriteString(pkg)
	buf.WriteString("\n\n")
	buf.WriteString(body)

	formatted, err := format.Source(buf.Bytes())
	if err != nil {
		// 回退到未格式化内容，便于排错
		formatted = buf.Bytes()
	}

	if err := os.MkdirAll(filepath.Dir(targetPath), 0755); err != nil {
		return err
	}
	return os.WriteFile(targetPath, formatted, 0644)
}

// emitLoadFile 生成 load.go 文件
func emitLoadFile(pkgName string, cache *GroupCache) error {
	outDir := filepath.Join(cache.Config.OutputRoot, pkgName)
	loadFile := filepath.Join(outDir, "load.go")

	// 使用注册表统一生成
	classes, functions := globalCache.ListRegistered(pkgName)
	body := buildLoadFileBody(pkgName, classes, functions)

	return emitFile(loadFile, pkgName, body)
}

// buildLoadFileBody 构建 load.go 文件内容
func buildLoadFileBody(pkgName string, classes, functions []string) string {
	b := &bytes.Buffer{}

	b.WriteString("import (\n")
	b.WriteString("\t\"github.com/php-any/origami/data\"\n")
	b.WriteString(")\n\n")

	b.WriteString("func Load(vm data.VM) {\n")

	// 添加函数
	if len(functions) > 0 {
		b.WriteString("\t// 添加顶级函数\n")
		b.WriteString("\tfor _, fun := range []data.FuncStmt{\n")
		for _, funcName := range functions {
			b.WriteString("\t\tNew" + funcName + "Function(),\n")
		}
		b.WriteString("\t} {\n")
		b.WriteString("\t\tvm.AddFunc(fun)\n")
		b.WriteString("\t}\n\n")
	}

	// 添加类
	if len(classes) > 0 {
		b.WriteString("\t// 添加类\n")
		for _, className := range classes {
			b.WriteString("\tvm.AddClass(New" + className + "Class())\n")
		}
	}

	b.WriteString("}\n")
	return b.String()
}
