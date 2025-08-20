package generator

import (
	"bytes"
	"go/format"
	"os"
)

// EmitFile 生成文件，自动 gofmt
func EmitFile(targetPath string, pkg string, body string) error {
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

	if err := os.MkdirAll(filepathDir(targetPath), 0o755); err != nil {
		return err
	}
	return os.WriteFile(targetPath, formatted, 0o644)
}

func filepathDir(path string) string {
	// 避免引入 path/filepath 只为 Dir
	for i := len(path) - 1; i >= 0; i-- {
		if path[i] == '/' {
			return path[:i]
		}
	}
	return "."
}
