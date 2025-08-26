package main

import (
	"fmt"
	"github.com/php-any/wails/application"
	"os"

	"github.com/php-any/origami/parser"
	"github.com/php-any/origami/runtime"
	"github.com/php-any/origami/std"
)

func main() {
	// 创建解析器
	p := parser.NewParser()

	// 创建程序运行的环境
	vm := runtime.NewVM(p)
	std.Load(vm)
	application.Load(vm)

	// 获取脚本路径参数
	scriptPath := "./sql.zy"

	// 检查文件是否存在
	if _, err := os.Stat(scriptPath); os.IsNotExist(err) {
		_, _ = fmt.Fprintf(os.Stderr, "错误: 文件 '%s' 不存在\n\n", scriptPath)
		os.Exit(1)
	}

	_, err := vm.LoadAndRun(scriptPath)
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "错误: %v\n", err)
		if !parser.InLSP {
			panic(err)
		}
	}
}
