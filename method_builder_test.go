package generator

import (
	"fmt"
	"os"
	"testing"

	"github.com/wailsapp/wails/v3/pkg/application"
)

func Test_buildMethodFileBody(t *testing.T) {
	sqlArray := []any{
		application.New,
		application.NewMenu,
		application.NewAppMenu,
		application.NewWindowEvent,
		application.WebviewWindowOptions{},
	}

	outRoot := "origami"
	sqlOpt := GenOptions{OutputRoot: outRoot, NamePrefix: "wails\\\\application"}
	for _, elem := range sqlArray {
		if err := GenerateFromConstructor(elem, sqlOpt); err != nil {
			fmt.Fprintln(os.Stderr, "生成失败:", err)
			continue
		}
		fmt.Println("生成完成 ->", outRoot)
	}
}
