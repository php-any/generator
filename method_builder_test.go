package generator

import (
	"fmt"
	"os"
	"testing"

	"github.com/redis/go-redis/v9"
)

func Test_buildMethodFileBody(t *testing.T) {
	sqlArray := []any{
		redis.NewClient,
	}

	outRoot := "origami"
	sqlOpt := GenOptions{OutputRoot: outRoot, NamePrefix: "redis"}
	for _, elem := range sqlArray {
		if err := GenerateFromConstructor(elem, sqlOpt); err != nil {
			fmt.Fprintln(os.Stderr, "生成失败:", err)
			continue
		}
		fmt.Println("生成完成 ->", outRoot)
	}
}
