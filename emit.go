package generator

import (
	"bytes"
	"go/format"
	"os"
	"path/filepath"
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
	// 使用标准库的 filepath.Dir 来处理跨平台路径
	return filepath.Dir(path)
}
