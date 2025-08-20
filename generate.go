package generator

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"sort"
	"strings"
	"sync"
)

type GenOptions struct {
	// 输出根目录，例如: origami
	OutputRoot string
}

// 全局缓存，防止递归死循环
var generatedTypes = make(map[string]bool)
var generatedTypesMutex sync.Mutex

// GenerateFromConstructor：先为函数本身生成函数代理；若返回 *struct，再生成类与方法代理
func GenerateFromConstructor(fn any, opt GenOptions) error {
	// 先尝试生成顶级函数代理（失败则提示，不阻断后续流程）
	if err := generateTopFunction(fn, opt); err != nil {
		// 解析函数名
		name := "<unknown>"
		if f := runtime.FuncForPC(reflect.ValueOf(fn).Pointer()); f != nil {
			name = f.Name()
			if idx := strings.LastIndex(name, "."); idx >= 0 {
				name = name[idx+1:]
			}
		}
		fmt.Fprintf(os.Stderr, "跳过函数代理 (%s): %v\n", name, err)
	}

	info, err := InspectFunction(fn)
	if err != nil {
		return err
	}
	if len(info.ReturnTypes) == 0 {
		return errors.New("构造函数没有返回值")
	}

	ret := info.ReturnTypes[0]
	if ret.Kind() != reflect.Pointer || ret.Elem().Kind() != reflect.Struct {
		// 非 *struct 的函数仅生成函数代理即可
		return nil
	}

	// 检查缓存，防止重复生成
	typeKey := ret.String()
	generatedTypesMutex.Lock()
	if generatedTypes[typeKey] {
		generatedTypesMutex.Unlock()
		fmt.Fprintf(os.Stderr, "跳过已生成的类型: %s\n", typeKey)
		return nil
	}
	// 标记为已生成
	generatedTypes[typeKey] = true
	generatedTypesMutex.Unlock()
	
	fmt.Fprintf(os.Stderr, "开始生成主类型: %s\n", typeKey)

	structPtr := ret
	structType := ret.Elem()

	srcPkgPath := structType.PkgPath()
	if srcPkgPath == "" {
		return errors.New("无法获取源包路径")
	}
	pkgName := pkgBaseName(srcPkgPath)
	typeName := structType.Name()

	// 收集方法（指针方法集）
	allMethods := map[string]reflect.Method{}
	for i := 0; i < structPtr.NumMethod(); i++ {
		m := structPtr.Method(i)
		// 仅导出方法
		if m.PkgPath == "" && isExportedName(m.Name) {
			allMethods[m.Name] = m
		}
	}

	// 生成方法文件（可能为空）
	outDir := filepath.Join(opt.OutputRoot, pkgName)
	names := make([]string, 0, len(allMethods))
	for n := range allMethods {
		names = append(names, n)
	}
	sort.Strings(names)

	skipped := make([]string, 0)
	supported := map[string]reflect.Method{}
	for _, n := range names {
		m := allMethods[n]
		file := filepath.Join(outDir, strings.ToLower(typeName)+"_"+strings.ToLower(n)+"_method.go")
		body, ok := buildMethodFileBody(srcPkgPath, pkgName, typeName, m)
		if !ok {
			skipped = append(skipped, n)
			continue
		}
		if err := EmitFile(file, pkgName, body); err != nil {
			return err
		}
		supported[n] = m

		// 若方法返回 *struct，则继续为其生成代理
		mt := m.Type
		fmt.Fprintf(os.Stderr, "检查方法 %s.%s 的返回类型\n", typeName, n)
		for oi := 0; oi < mt.NumOut(); oi++ {
			outType := mt.Out(oi)
			fmt.Fprintf(os.Stderr, "  返回值 %d: %s (Kind: %s, isPtrToStruct: %v)\n", 
				oi, outType.String(), outType.Kind(), isPtrToStruct(outType))
			if isPtrToStruct(outType) {
				fmt.Fprintf(os.Stderr, "    为返回类型 %s 生成代理类\n", outType.String())
				_ = generateClassFromType(outType, opt)
			}
		}
		// 若方法参数含 *struct（排除 context.Context），也生成对应代理
		for ii := 1; ii < mt.NumIn(); ii++ {
			pt := mt.In(ii)
			if isPtrToStruct(pt) {
				// 排除 context.Context 指针与 Context 本身
				if pt.Elem().PkgPath() == "context" && pt.Elem().Name() == "Context" {
					continue
				}
				_ = generateClassFromType(pt, opt)
			}
		}
	}

	if len(skipped) > 0 {
		fmt.Fprintf(os.Stderr, "部分方法被跳过 (%s): %s\n", typeName, strings.Join(skipped, ", "))
	}

	// 生成 class 文件（即便无方法也生成空类，便于参数/返回代理）
	classFile := filepath.Join(outDir, strings.ToLower(typeName)+"_class.go")
	classBody := buildClassFileBody(srcPkgPath, pkgName, typeName, supported, structType)
	if err := EmitFile(classFile, pkgName, classBody); err != nil {
		return err
	}

	// 生成 load.go 文件
	if err := generateLoadFile(pkgName, opt); err != nil {
		fmt.Fprintf(os.Stderr, "生成 load.go 失败: %v\n", err)
	}

	return nil
}

// generateLoadFile 为包生成 load.go 文件
func generateLoadFile(pkgName string, opt GenOptions) error {
	outDir := filepath.Join(opt.OutputRoot, pkgName)
	loadFile := filepath.Join(outDir, "load.go")
	
	// 扫描目录中的文件来确定可用的类和函数
	var classes []string
	var functions []string
	
	// 检查类文件
	entries, err := os.ReadDir(outDir)
	if err != nil {
		return err
	}
	
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if strings.HasSuffix(name, "_class.go") {
			// 提取类名：db_class.go -> DB, txoptions_class.go -> TxOptions
			className := strings.TrimSuffix(name, "_class.go")
			// 处理特殊类名映射
			switch className {
			case "db":
				className = "DB"
			case "txoptions":
				className = "TxOptions"
			default:
				// 处理下划线分隔的类名，如 txoptions -> TxOptions
				parts := strings.Split(className, "_")
				for i, part := range parts {
					parts[i] = upperFirst(part)
				}
				className = strings.Join(parts, "")
			}
			classes = append(classes, className)
		} else if strings.HasSuffix(name, "_func.go") {
			// 提取函数名：open_func.go -> Open
			funcName := strings.TrimSuffix(name, "_func.go")
			funcName = upperFirst(funcName)
			functions = append(functions, funcName)
		}
	}
	
	// 生成 load.go 内容
	body := buildLoadFileBody(pkgName, classes, functions)
	return EmitFile(loadFile, pkgName, body)
}

// buildLoadFileBody 构建 load.go 文件内容
func buildLoadFileBody(pkgName string, classes, functions []string) string {
	b := &strings.Builder{}
	
	b.WriteString("import (\n")
	b.WriteString("\t\"github.com/php-any/origami/data\"\n")
	b.WriteString(")\n\n")
	
	b.WriteString("func Load(vm data.VM) {\n")
	
	// 添加函数
	if len(functions) > 0 {
		b.WriteString("\t// 添加顶级函数\n")
		b.WriteString("\tfor _, fun := range []data.FuncStmt{\n")
		for _, funcName := range functions {
			fmt.Fprintf(b, "\t\tNew%sFunction(),\n", funcName)
		}
		b.WriteString("\t} {\n")
		b.WriteString("\t\tvm.AddFunc(fun)\n")
		b.WriteString("\t}\n\n")
	}
	
	// 添加类
	if len(classes) > 0 {
		b.WriteString("\t// 添加类\n")
		for _, className := range classes {
			fmt.Fprintf(b, "\tvm.AddClass(New%sClass())\n", className)
		}
	}
	
	b.WriteString("}\n")
	return b.String()
}

// generateClassFromType: 直接基于返回的 *struct 类型生成类与方法代理
func generateClassFromType(structPtr reflect.Type, opt GenOptions) error {
	if !isPtrToStruct(structPtr) {
		return fmt.Errorf("期望 *struct 类型，实际: %s", structPtr.String())
	}
	
	// 生成类型标识符
	typeKey := structPtr.String()
	
	// 检查缓存，防止重复生成
	generatedTypesMutex.Lock()
	if generatedTypes[typeKey] {
		generatedTypesMutex.Unlock()
		fmt.Fprintf(os.Stderr, "跳过已生成的类型: %s\n", typeKey)
		return nil
	}
	// 标记为已生成
	generatedTypes[typeKey] = true
	generatedTypesMutex.Unlock()
	
	fmt.Fprintf(os.Stderr, "开始生成类型: %s\n", typeKey)
	
	structType := structPtr.Elem()

	srcPkgPath := structType.PkgPath()
	if srcPkgPath == "" {
		return errors.New("无法获取源包路径")
	}
	pkgName := pkgBaseName(srcPkgPath)
	typeName := structType.Name()

	// 收集方法（指针方法集）
	methods := map[string]reflect.Method{}
	for i := 0; i < structPtr.NumMethod(); i++ {
		mm := structPtr.Method(i)
		if mm.PkgPath == "" && isExportedName(mm.Name) {
			methods[mm.Name] = mm
		}
	}

	outDir := filepath.Join(opt.OutputRoot, pkgName)
	// 方法文件（可能为空）
	names := make([]string, 0, len(methods))
	for n := range methods {
		names = append(names, n)
	}
	sort.Strings(names)
	for _, n := range names {
		mm := methods[n]
		file := filepath.Join(outDir, strings.ToLower(typeName)+"_"+strings.ToLower(n)+"_method.go")
		body, ok := buildMethodFileBody(srcPkgPath, pkgName, typeName, mm)
		if !ok {
			continue
		}
		if err := EmitFile(file, pkgName, body); err != nil {
			return err
		}
		
		// 递归处理该方法的返回值和参数，生成相关代理类
		mt := mm.Type
		fmt.Fprintf(os.Stderr, "检查方法 %s.%s 的返回类型和参数\n", typeName, n)
		// 检查方法返回值
		for oi := 0; oi < mt.NumOut(); oi++ {
			outType := mt.Out(oi)
			fmt.Fprintf(os.Stderr, "  返回值 %d: %s (Kind: %s, isPtrToStruct: %v)\n", 
				oi, outType.String(), outType.Kind(), isPtrToStruct(outType))
			if isPtrToStruct(outType) {
				fmt.Fprintf(os.Stderr, "    为返回类型 %s 递归生成代理类\n", outType.String())
				_ = generateClassFromType(outType, opt)
			}
		}
		// 检查方法参数
		for ii := 1; ii < mt.NumIn(); ii++ {
			pt := mt.In(ii)
			fmt.Fprintf(os.Stderr, "  参数 %d: %s (Kind: %s, isPtrToStruct: %v)\n", 
				ii, pt.String(), pt.Kind(), isPtrToStruct(pt))
			if isPtrToStruct(pt) {
				// 排除 context.Context 指针
				if pt.Elem().PkgPath() == "context" && pt.Elem().Name() == "Context" {
					fmt.Fprintf(os.Stderr, "    跳过 context.Context 参数\n")
					continue
				}
				fmt.Fprintf(os.Stderr, "    为参数类型 %s 递归生成代理类\n", pt.String())
				_ = generateClassFromType(pt, opt)
			}
		}
	}
	// 类文件（即便无方法也生成空类）
	classFile := filepath.Join(outDir, strings.ToLower(typeName)+"_class.go")
	classBody := buildClassFileBody(srcPkgPath, pkgName, typeName, methods, structType)
	if err := EmitFile(classFile, pkgName, classBody); err != nil {
		return err
	}
	return nil
}

func isPtrToStruct(t reflect.Type) bool {
	return t.Kind() == reflect.Pointer && t.Elem() != nil && t.Elem().Kind() == reflect.Struct
}
