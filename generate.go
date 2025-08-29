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
	// 自定义 GetName 拼接前缀；为空则使用源包名
	NamePrefix string
	// 最大递归生成层次（<=0 表示不限制）
	MaxDepth int
	// 当前递归层次（内部使用）
	currentDepth int
}

// 全局缓存，防止递归死循环
var generatedTypes = make(map[string]bool)
var generatedTypesMutex sync.Mutex

func effectiveNamePrefix(defaultPkgName string, opt GenOptions) string {
	if opt.NamePrefix != "" {
		return opt.NamePrefix
	}
	return defaultPkgName
}

// GenerateFromConstructor：先为函数本身生成函数代理；若返回 *struct，再生成类与方法代理
func GenerateFromConstructor(fn any, opt GenOptions) error {
	// 统一入口支持：结构体值 / *struct / 函数
	t := reflect.TypeOf(fn)
	if t == nil {
		return errors.New("输入为 nil，不支持")
	}
	// 结构体值：直接生成对应类（取其指针类型）
	if t.Kind() == reflect.Struct && t.PkgPath() != "" && t.Name() != "" {
		return generateClassFromType(reflect.PointerTo(t), opt)
	}
	// *struct：直接生成对应类
	if t.Kind() == reflect.Ptr && t.Elem() != nil && t.Elem().Kind() == reflect.Struct {
		return generateClassFromType(t, opt)
	}
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
		return nil
	}
	// 标记为已生成
	generatedTypes[typeKey] = true
	generatedTypesMutex.Unlock()

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

	supported := map[string]reflect.Method{}
	for _, n := range names {
		m := allMethods[n]
		file := filepath.Join(outDir, strings.ToLower(typeName)+"_"+strings.ToLower(n)+"_method.go")
		// 若主文件已存在，认为该方法已生成，直接跳过，避免生成后缀 _2/_3
		if _, statErr := os.Stat(file); statErr == nil {
			continue
		}
		body, ok := buildMethodFileBody(srcPkgPath, pkgName, typeName, m, true)
		if !ok {
			continue
		}
		if err := EmitFile(file, pkgName, body); err != nil {
			return err
		}
		supported[n] = m

		// 若方法返回 *struct，则继续为其生成代理
		mt := m.Type
		for oi := 0; oi < mt.NumOut(); oi++ {
			outType := mt.Out(oi)
			if isTypeNeedsProxy(outType) {
				_ = generateProxyFromTypeWithDepth(outType, &opt)
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
				_ = generateProxyFromTypeWithDepth(pt, &opt)
			}
		}
	}

	// 在生成类文件前，遍历导出字段并递归生成其类型的代理（接口、struct）
	for i := 0; i < structType.NumField(); i++ {
		field := structType.Field(i)
		if field.PkgPath != "" { // 非导出字段跳过
			continue
		}
		ft := field.Type
		fmt.Fprintf(os.Stderr, "字段递归: %s.%s => %s\n", typeName, field.Name, ft.String())
		// 接口类型
		if ft.Kind() == reflect.Interface && ft.PkgPath() != "" && ft.Name() != "" {
			fmt.Fprintf(os.Stderr, "  -> 生成接口代理: %s\n", ft.String())
			_ = generateProxyFromTypeWithDepth(ft, &opt)
			continue
		}
		// *struct 类型
		if isPtrToStruct(ft) {
			fmt.Fprintf(os.Stderr, "  -> 生成 *struct 代理: %s\n", ft.String())
			_ = generateProxyFromTypeWithDepth(ft, &opt)
			continue
		}
		// 值 struct 类型
		if ft.Kind() == reflect.Struct && ft.PkgPath() != "" && ft.Name() != "" {
			fmt.Fprintf(os.Stderr, "  -> 生成 struct 代理: &%s\n", ft.String())
			_ = generateProxyFromTypeWithDepth(reflect.PointerTo(ft), &opt)
			continue
		}
	}

	// 生成 class 文件（即便无方法也生成空类，便于参数/返回代理）
	classFile := filepath.Join(outDir, strings.ToLower(typeName)+"_class.go")
	classBody := buildClassFileBody(srcPkgPath, pkgName, typeName, supported, structType, effectiveNamePrefix(pkgName, opt))
	if err := EmitFile(classFile, pkgName, classBody); err != nil {
		return err
	}
	// 注册类并生成/更新 load.go
	registerClass(pkgName, typeName)
	if err := generateLoadFile(pkgName, opt); err != nil {
		fmt.Fprintf(os.Stderr, "生成 load.go 失败: %v\n", err)
	}

	return nil
}

// generateLoadFile 为包生成 load.go 文件
func generateLoadFile(pkgName string, opt GenOptions) error {
	outDir := filepath.Join(opt.OutputRoot, pkgName)
	loadFile := filepath.Join(outDir, "load.go")
	// 使用注册表统一生成
	classes, functions := listRegistered(pkgName)
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
		return nil
	}
	// 标记为已生成
	generatedTypes[typeKey] = true
	generatedTypesMutex.Unlock()

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
	// 仅记录成功生成的方法，避免类文件引用不存在的方法
	supported := map[string]reflect.Method{}
	for _, n := range names {
		mm := methods[n]
		file := filepath.Join(outDir, strings.ToLower(typeName)+"_"+strings.ToLower(n)+"_method.go")
		// 若主文件已存在，认为该方法已生成，直接跳过，避免生成后缀 _2/_3
		if _, statErr := os.Stat(file); statErr == nil {
			continue
		}
		body, ok := buildMethodFileBody(srcPkgPath, pkgName, typeName, mm, true)
		if !ok {
			continue
		}
		if err := EmitFile(file, pkgName, body); err != nil {
			return err
		}
		supported[n] = mm

		// 递归处理该方法的返回值和参数，生成相关代理类
		mt := mm.Type
		// 检查方法返回值
		for oi := 0; oi < mt.NumOut(); oi++ {
			outType := mt.Out(oi)
			if isTypeNeedsProxy(outType) {
				_ = generateProxyFromTypeWithDepth(outType, &opt)
			}
		}
		// 检查方法参数
		for ii := 1; ii < mt.NumIn(); ii++ {
			pt := mt.In(ii)
			if isPtrToStruct(pt) {
				// 排除 context.Context 指针
				if pt.Elem().PkgPath() == "context" && pt.Elem().Name() == "Context" {
					continue
				}
				_ = generateProxyFromTypeWithDepth(pt, &opt)
			}
		}
	}

	// 在生成类文件前，遍历导出字段并递归生成其类型的代理（接口、struct）
	for i := 0; i < structType.NumField(); i++ {
		field := structType.Field(i)
		if field.PkgPath != "" { // 非导出字段跳过
			continue
		}
		ft := field.Type
		fmt.Fprintf(os.Stderr, "字段递归: %s.%s => %s\n", typeName, field.Name, ft.String())
		// 接口类型
		if ft.Kind() == reflect.Interface && ft.PkgPath() != "" && ft.Name() != "" {
			fmt.Fprintf(os.Stderr, "  -> 生成接口代理: %s\n", ft.String())
			_ = generateProxyFromTypeWithDepth(ft, &opt)
			continue
		}
		// *struct 类型
		if isPtrToStruct(ft) {
			fmt.Fprintf(os.Stderr, "  -> 生成 *struct 代理: %s\n", ft.String())
			_ = generateProxyFromTypeWithDepth(ft, &opt)
			continue
		}
		// 值 struct 类型
		if ft.Kind() == reflect.Struct && ft.PkgPath() != "" && ft.Name() != "" {
			fmt.Fprintf(os.Stderr, "  -> 生成 struct 代理: &%s\n", ft.String())
			_ = generateProxyFromTypeWithDepth(reflect.PointerTo(ft), &opt)
			continue
		}
	}

	// 类文件（即便无方法也生成空类）
	classFile := filepath.Join(outDir, strings.ToLower(typeName)+"_class.go")
	classBody := buildClassFileBody(srcPkgPath, pkgName, typeName, supported, structType, effectiveNamePrefix(pkgName, opt))
	if err := EmitFile(classFile, pkgName, classBody); err != nil {
		return err
	}
	// 注册类并尝试生成/更新 load.go
	registerClass(pkgName, typeName)
	if err := generateLoadFile(pkgName, opt); err != nil {
		fmt.Fprintf(os.Stderr, "递归生成 load.go 失败: %v\n", err)
	}
	return nil
}

// generateProxyFromType: 为需要代理的类型（*struct 或接口）生成代理类
func generateProxyFromType(typ reflect.Type, opt GenOptions) error {
	if isPtrToStruct(typ) {
		return generateClassFromType(typ, opt)
	}
	// 兼容值类型 struct：取其指针类型再生成
	if typ.Kind() == reflect.Struct && typ.PkgPath() != "" && typ.Name() != "" {
		return generateClassFromType(reflect.PointerTo(typ), opt)
	}
	if typ.Kind() == reflect.Interface && typ.PkgPath() != "" && typ.Name() != "" {
		return generateInterfaceProxy(typ, opt)
	}
	return fmt.Errorf("不支持的类型: %s", typ.String())
}

// 带深度控制的代理生成功能
func generateProxyFromTypeWithDepth(typ reflect.Type, opt *GenOptions) error {
	// 深度检查：首次进入为 0 层
	if opt != nil && opt.MaxDepth > 0 {
		if opt.currentDepth >= opt.MaxDepth {
			return nil
		}
		opt.currentDepth++
		defer func() { opt.currentDepth-- }()
	}
	return generateProxyFromType(typ, *opt)
}

// generateInterfaceProxy: 为接口类型生成代理类，复用现有的生成逻辑
func generateInterfaceProxy(iface reflect.Type, opt GenOptions) error {
	// 生成类型标识符
	typeKey := iface.String()

	// 检查缓存，防止重复生成
	generatedTypesMutex.Lock()
	if generatedTypes[typeKey] {
		generatedTypesMutex.Unlock()
		return nil
	}
	// 标记为已生成
	generatedTypes[typeKey] = true
	generatedTypesMutex.Unlock()

	srcPkgPath := iface.PkgPath()
	if srcPkgPath == "" {
		return errors.New("无法获取接口源包路径")
	}
	pkgName := pkgBaseName(srcPkgPath)
	typeName := iface.Name()

	// 收集接口的所有方法
	allMethods := map[string]reflect.Method{}
	for i := 0; i < iface.NumMethod(); i++ {
		m := iface.Method(i)
		// 仅导出方法
		if m.PkgPath == "" && isExportedName(m.Name) {
			allMethods[m.Name] = m
		}
	}

	// 生成方法文件，复用现有的 buildMethodFileBody
	outDir := filepath.Join(opt.OutputRoot, pkgName)
	names := make([]string, 0, len(allMethods))
	for n := range allMethods {
		names = append(names, n)
	}
	sort.Strings(names)

	for _, n := range names {
		m := allMethods[n]

		// 检查方法的返回值，看是否需要生成代理类
		mt := m.Type
		for oi := 0; oi < mt.NumOut(); oi++ {
			outType := mt.Out(oi)
			if isTypeNeedsProxy(outType) {
				_ = generateProxyFromTypeWithDepth(outType, &opt)
			}
		}

		file := filepath.Join(outDir, strings.ToLower(typeName)+"_"+strings.ToLower(n)+"_method.go")
		body, ok := buildMethodFileBody(srcPkgPath, pkgName, typeName, m, false)
		if !ok {
			continue
		}
		if err := EmitFile(file, pkgName, body); err != nil {
			return err
		}
	}

	// 生成类文件，复用现有的 buildClassFileBody
	classFile := filepath.Join(outDir, strings.ToLower(typeName)+"_class.go")
	// 对于接口类型，我们传入一个空的 reflect.Type 作为 structType，因为接口没有具体的结构
	// 但 buildClassFileBody 需要这个参数，所以我们传入 nil 或者创建一个空的类型
	classBody := buildClassFileBody(srcPkgPath, pkgName, typeName, allMethods, nil, effectiveNamePrefix(pkgName, opt))
	if err := EmitFile(classFile, pkgName, classBody); err != nil {
		return err
	}

	// 接口类不注册到 load.go（避免需要无参构造）
	return nil
}

func isPtrToStruct(t reflect.Type) bool {
	return t.Kind() == reflect.Pointer && t.Elem() != nil && t.Elem().Kind() == reflect.Struct
}

// isTypeNeedsProxy 检查类型是否需要生成代理类
func isTypeNeedsProxy(t reflect.Type) bool {
	// 检查 *struct 类型
	if t.Kind() == reflect.Ptr && t.Elem() != nil && t.Elem().Kind() == reflect.Struct {
		return true
	}
	// 检查 struct 类型（值类型结构体）
	if t.Kind() == reflect.Struct && t.PkgPath() != "" && t.Name() != "" {
		return true
	}
	// 检查具名接口类型（如 sql.Result）
	if t.Kind() == reflect.Interface && t.PkgPath() != "" && t.Name() != "" {
		return true
	}
	return false
}
