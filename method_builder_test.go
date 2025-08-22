package generator

import (
	"database/sql"
	"fmt"
	"os"
	"testing"
)

func Test_buildMethodFileBody(t *testing.T) {
	array := []any{
		sql.Open,
	}

	outRoot := "origami"
	opt := GenOptions{OutputRoot: outRoot, NamePrefix: "database\\\\sql"}
	for _, elem := range array {
		if err := GenerateFromConstructor(elem, opt); err != nil {
			fmt.Fprintln(os.Stderr, "生成失败:", err)
			continue
		}
		fmt.Println("生成完成 ->", outRoot)
	}
}
