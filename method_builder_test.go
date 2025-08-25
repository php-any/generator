package generator

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"testing"
)

func Test_buildMethodFileBody(t *testing.T) {
	// 处理 sql 包函数
	sqlArray := []any{
		sql.Open,
	}

	outRoot := "origami"
	sqlOpt := GenOptions{OutputRoot: outRoot, NamePrefix: "database\\\\sql"}
	for _, elem := range sqlArray {
		if err := GenerateFromConstructor(elem, sqlOpt); err != nil {
			fmt.Fprintln(os.Stderr, "生成失败:", err)
			continue
		}
		fmt.Println("生成完成 ->", outRoot)
	}

	// 处理 context 包函数，使用正确的 NamePrefix
	contextArray := []any{
		context.Background,
		context.WithCancel,
		context.WithTimeout,
		context.WithValue,
		context.WithoutCancel,
		context.WithDeadline,
		context.WithCancelCause,
		context.WithDeadlineCause,
		context.WithTimeoutCause,
	}

	contextOpt := GenOptions{OutputRoot: outRoot, NamePrefix: "context"}
	for _, elem := range contextArray {
		if err := GenerateFromConstructor(elem, contextOpt); err != nil {
			fmt.Fprintln(os.Stderr, "生成失败:", err)
			continue
		}
		fmt.Println("生成完成 ->", outRoot)
	}
}
