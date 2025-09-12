package main

import (
	"fmt"
	"os"

	"github.com/redis/go-redis/v9"

	"github.com/php-any/generator/scr"
)

var config = scr.Config{
	OutputRoot: "origami",
	NamePrefix: "redis",
	MaxDepth:   1000,
	Blacklist: scr.BlacklistConfig{
		Packages: []string{"time"},
	},
	PackageMappings: map[string]string{},
}

var genList = []any{
	//// 函数测试
	//demo.NewUser,
	//
	//// 循环引用测试
	//demo.Node{},
	redis.NewClient,
}

func main() {
	for _, a := range genList {
		err := scr.GenerateFromAny(a, &config)
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
	}
}
