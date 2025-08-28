package main

import (
	"fmt"
	"log"
	"reflect"

	"github.com/php-any/generator/analyzer"
	"github.com/php-any/generator/config"
	"github.com/php-any/generator/core"
)

// User 示例结构体
type User struct {
	ID       int
	Name     string
	Email    string
	IsActive bool
}

// UserService 示例服务
type UserService struct {
	users []User
}

// CreateUser 创建用户
func (s *UserService) CreateUser(name, email string) (*User, error) {
	user := &User{
		ID:       len(s.users) + 1,
		Name:     name,
		Email:    email,
		IsActive: true,
	}
	s.users = append(s.users, *user)
	return user, nil
}

// GetUser 获取用户
func (s *UserService) GetUser(id int) (*User, error) {
	for _, user := range s.users {
		if user.ID == id {
			return &user, nil
		}
	}
	return nil, fmt.Errorf("user not found: %d", id)
}

func main() {
	fmt.Println("Generator 重构示例")
	fmt.Println("==================")

	// 1. 创建生成器上下文
	ctx := core.NewGeneratorContext(&core.GenOptions{
		OutputRoot: "generated",
		MaxDepth:   3,
		Verbose:    true,
	})

	// 2. 创建配置管理器
	configManager := config.NewConfigManager(ctx.ErrorHandler)
	ctx.SetConfigManager(configManager)

	// 3. 创建类型分析器
	typeAnalyzer := analyzer.NewTypeAnalyzer(ctx)

	// 4. 创建包分析器
	packageAnalyzer := analyzer.NewPackageAnalyzer(ctx)

	// 5. 分析示例类型
	fmt.Println("\n1. 分析 User 结构体:")
	userType, err := typeAnalyzer.AnalyzeType(reflect.TypeOf(User{}))
	if err != nil {
		log.Fatalf("Failed to analyze User type: %v", err)
	}

	fmt.Printf("   - 类型名称: %s\n", userType.TypeName)
	fmt.Printf("   - 包路径: %s\n", userType.PackagePath)
	fmt.Printf("   - 是否为结构体: %v\n", userType.IsStruct)
	fmt.Printf("   - 字段数量: %d\n", len(userType.Fields))

	for _, field := range userType.Fields {
		fmt.Printf("     - %s: %s\n", field.Name, field.Type.TypeName)
	}

	// 6. 分析 UserService 结构体
	fmt.Println("\n2. 分析 UserService 结构体:")
	serviceType, err := typeAnalyzer.AnalyzeType(reflect.TypeOf(UserService{}))
	if err != nil {
		log.Fatalf("Failed to analyze UserService type: %v", err)
	}

	fmt.Printf("   - 类型名称: %s\n", serviceType.TypeName)
	fmt.Printf("   - 方法数量: %d\n", len(serviceType.Methods))

	for _, method := range serviceType.Methods {
		fmt.Printf("     - %s(", method.Name)
		for i, param := range method.Parameters {
			if i > 0 {
				fmt.Print(", ")
			}
			fmt.Printf("%s %s", param.Name, param.Type.TypeName)
		}
		fmt.Print(")")
		if len(method.Returns) > 0 {
			fmt.Print(" (")
			for i, ret := range method.Returns {
				if i > 0 {
					fmt.Print(", ")
				}
				fmt.Print(ret.TypeName)
			}
			fmt.Print(")")
		}
		fmt.Println()
	}

	// 7. 分析函数
	fmt.Println("\n3. 分析示例函数:")
	exampleFunc := func(name string, age int) (User, error) {
		return User{Name: name, IsActive: true}, nil
	}

	funcInfo, err := typeAnalyzer.AnalyzeFunction(exampleFunc)
	if err != nil {
		log.Fatalf("Failed to analyze function: %v", err)
	}

	fmt.Printf("   - 参数数量: %d\n", len(funcInfo.Parameters))
	fmt.Printf("   - 返回值数量: %d\n", len(funcInfo.Returns))
	fmt.Printf("   - 是否为变参函数: %v\n", funcInfo.IsVariadic)

	// 8. 收集导入依赖
	fmt.Println("\n4. 收集导入依赖:")
	imports, err := packageAnalyzer.AnalyzeImports([]*core.TypeInfo{userType, serviceType})
	if err != nil {
		log.Fatalf("Failed to analyze imports: %v", err)
	}

	fmt.Printf("   - 导入数量: %d\n", len(imports))
	for _, imp := range imports {
		fmt.Printf("     - %s (别名: %s)\n", imp.Path, imp.Alias)
	}

	// 9. 显示统计信息
	fmt.Println("\n5. 统计信息:")
	metrics := ctx.Metrics.GetMetrics()
	fmt.Printf("   - 分析的类型数量: %d\n", metrics.TypesAnalyzed)
	fmt.Printf("   - 错误数量: %d\n", metrics.ErrorsCount)
	fmt.Printf("   - 警告数量: %d\n", metrics.WarningsCount)
	fmt.Printf("   - 缓存命中率: %.2f%%\n", metrics.CacheHitRate*100)

	// 10. 测试配置功能
	fmt.Println("\n6. 测试配置功能:")
	if ctx.GetConfigManager() != nil {
		genConfig := ctx.GetConfigManager().GetConfig()
		fmt.Printf("   - 全局前缀: %s\n", genConfig.GlobalPrefix)
		fmt.Printf("   - 输出根目录: %s\n", genConfig.OutputRoot)
		fmt.Printf("   - 最大深度: %d\n", genConfig.MaxDepth)
		fmt.Printf("   - 并行处理: %v\n", genConfig.Parallel)
	}

	// 11. 测试黑名单功能
	fmt.Println("\n7. 测试黑名单功能:")
	// 添加类型到黑名单
	genConfig := ctx.GetConfigManager().GetConfig()
	genConfig.Blacklist.Types = append(genConfig.Blacklist.Types, "User")

	// 尝试分析黑名单中的类型
	_, err = typeAnalyzer.AnalyzeType(reflect.TypeOf(User{}))
	if err != nil {
		fmt.Printf("   - 成功阻止黑名单类型: %v\n", err)
	} else {
		fmt.Println("   - 警告: 黑名单类型未被阻止")
	}

	fmt.Println("\n示例完成！")
}
