package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"reflect"

	"github.com/php-any/generator/analyzer"
	"github.com/php-any/generator/config"
	"github.com/php-any/generator/converter"
	"github.com/php-any/generator/core"
	"github.com/php-any/generator/emitter"
	"github.com/php-any/generator/generator"
	"github.com/php-any/generator/templates"
)

func main() {
	// 解析命令行参数
	var configFile = flag.String("config", "", "配置文件路径 (YAML)")
	var maxDepth = flag.Int("depth", 100, "递归生成的最大深度")
	flag.Parse()

	fmt.Println("重构版本代码生成器启动...")

	// 1. 创建错误处理器
	errorHandler := core.NewDefaultErrorHandler()

	// 2. 创建配置管理器
	var configManager core.ConfigManager

	if *configFile != "" {
		// 创建带配置的配置管理器
		configManager = config.NewConfigManager(errorHandler)
		if err := configManager.LoadConfig(*configFile); err != nil {
			log.Fatalf("加载配置文件失败: %v", err)
		}
		fmt.Printf("已加载配置文件: %s\n", *configFile)
	} else {
		// 使用默认配置
		configManager = config.NewConfigManager(errorHandler)
		fmt.Println("使用默认配置")
	}

	// 3. 创建生成器上下文
	cfg := configManager.GetConfig()
	outRoot := "origami"
	namePrefix := "origami"
	if cfg != nil {
		if cfg.OutputRoot != "" {
			outRoot = cfg.OutputRoot
		}
		if cfg.GlobalPrefix != "" {
			namePrefix = cfg.GlobalPrefix
		}
	}
	ctx := core.NewGeneratorContext(&core.GenOptions{
		MaxDepth:   100,
		OutputRoot: outRoot,
		NamePrefix: namePrefix,
	})
	ctx.SetConfigManager(configManager)

	// 4. 创建转换器
	typeConverter := converter.NewTypeConverter(ctx)

	// 5. 创建代码管理器和模板引擎
	// codeManager := emitter.NewCodeManager(ctx.GetConfigManager().GetConfig())
	templateEngine := templates.NewTemplateEngine("./templates")
	if err := templateEngine.LoadTemplates(); err != nil {
		log.Fatalf("加载模板失败: %v", err)
	}
	var templateGenerator core.TemplateGenerator = templateEngine

	// 6. 创建文件输出器（使用真正的文件输出器）
	fileEmitter := emitter.NewFileEmitter(outRoot)

	// 7. 示例：分析 http.ServeMux 类型
	fmt.Println("\n分析 http.ServeMux 类型（包含依赖类型）...")

	// 分析 http.ServeMux 类型
	muxType := reflect.TypeOf((*http.ServeMux)(nil)).Elem()

	typeAnalyzer := analyzer.NewTypeAnalyzer(ctx)
	typeInfo, err := typeAnalyzer.AnalyzeType(muxType)
	if err != nil {
		log.Fatalf("分析 http.ServeMux 类型失败: %v", err)
	}

	fmt.Printf("类型名称: %s\n", typeInfo.TypeName)
	fmt.Printf("字段数量: %d\n", len(typeInfo.Fields))
	fmt.Printf("方法数量: %d\n", len(typeInfo.Methods))
	fmt.Printf("递归深度: %d\n", *maxDepth)

	// 调试信息：显示方法的详细信息
	fmt.Println("\n方法详细信息:")
	for i, method := range typeInfo.Methods {
		fmt.Printf("  方法 %d: %s\n", i, method.Name)
		for j, param := range method.Parameters {
			fmt.Printf("    参数 %d: %s (%s)\n", j, param.Name, param.Type.TypeName)
		}
		for j, ret := range method.Returns {
			fmt.Printf("    返回值 %d: %s\n", j, ret.Type.String())
		}
	}

	// 8. 示例：生成代码
	fmt.Println("\n生成代码...")

	// 创建代码生成器
	codeGenerator := generator.NewCodeGenerator(ctx, typeConverter, templateGenerator)

	// 生成类和方法文件（使用指定的递归深度）
	err = codeGenerator.GenerateClassWithMethodsRecursive(ctx, typeInfo, fileEmitter, *maxDepth)
	if err != nil {
		log.Fatalf("生成代码失败: %v", err)
	}

	fmt.Println("代码生成完成")

	// 10. 生成加载文件
	fmt.Println("\n生成加载文件...")
	// 使用上下文内实际包集合生成 load 文件，避免硬编码 example
	packages := fileEmitter.GetAllPackages()
	for _, pkg := range packages {
		_ = fileEmitter.EmitLoadFile(pkg, []string{}, []string{"ServeMux"})
	}

	fmt.Println("文件输出成功！")

	// 11. 显示统计信息
	fmt.Println("\n统计信息:")
	metrics := ctx.Metrics.GetMetrics()
	fmt.Printf("- 分析的类型数量: %d\n", metrics.TypesAnalyzed)
	fmt.Printf("- 错误数量: %d\n", metrics.ErrorsCount)
	fmt.Printf("- 警告数量: %d\n", metrics.WarningsCount)
	fmt.Printf("- 缓存命中率: %.2f%%\n", metrics.CacheHitRate*100)

	// 12. 显示生成的文件
	fmt.Println("\n生成的文件:")
	packages = fileEmitter.GetAllPackages()
	for _, pkg := range packages {
		files := fileEmitter.GetPackageFiles(pkg)
		fmt.Printf("- 包 %s:\n", pkg)
		for _, file := range files {
			fmt.Printf("  - %s\n", file)
		}
	}

	fmt.Println("\n重构版本运行完成！")
}
