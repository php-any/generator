package emitter

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"text/template"
)

// FileEmitterImpl 文件输出器实现
type FileEmitterImpl struct {
	outputRoot  string
	registry    map[string][]string // 包名 -> 文件列表
	codeManager *CodeManager
	pkgClasses  map[string]map[string]bool // 包名 -> 类名集合
	pkgFuncs    map[string]map[string]bool // 包名 -> 函数名集合
}

// 创建新的文件输出器
func NewFileEmitter(outputRoot string) *FileEmitterImpl {
	if outputRoot == "" {
		outputRoot = "origami"
	}

	return &FileEmitterImpl{
		outputRoot:  outputRoot,
		registry:    make(map[string][]string),
		codeManager: NewCodeManager(nil), // 暂时传入nil，后续可以通过参数传入
		pkgClasses:  make(map[string]map[string]bool),
		pkgFuncs:    make(map[string]map[string]bool),
	}
}

// EmitFile 输出文件
func (fe *FileEmitterImpl) EmitFile(pkgName, fileName, content string) error {
	if pkgName == "" {
		return fmt.Errorf("包名不能为空")
	}
	if fileName == "" {
		return fmt.Errorf("文件名不能为空")
	}
	if content == "" {
		return fmt.Errorf("文件内容不能为空")
	}

	// 创建包目录
	pkgDir := filepath.Join(fe.outputRoot, pkgName)
	if err := fe.CreateDirectory(pkgDir); err != nil {
		return fmt.Errorf("创建包目录失败: %v", err)
	}

	// 构建完整文件路径
	filePath := filepath.Join(pkgDir, fileName)

	// 写入文件
	if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
		return fmt.Errorf("写入文件失败: %v", err)
	}

	// 注册文件
	fe.registerFile(pkgName, fileName)

	return nil
}

// EmitLoadFile 生成加载文件
func (fe *FileEmitterImpl) EmitLoadFile(pkgName string, functions []string, classes []string) error {
	if pkgName == "" {
		return fmt.Errorf("包名不能为空")
	}

	// 创建包目录
	pkgDir := filepath.Join(fe.outputRoot, pkgName)
	if err := fe.CreateDirectory(pkgDir); err != nil {
		return fmt.Errorf("创建包目录失败: %v", err)
	}

	// 生成加载文件内容
	content, err := fe.generateLoadFileContent(pkgName, functions, classes)
	if err != nil {
		return fmt.Errorf("生成加载文件内容失败: %v", err)
	}

	// 写入加载文件
	loadFilePath := filepath.Join(pkgDir, "load.go")
	if err := os.WriteFile(loadFilePath, []byte(content), 0644); err != nil {
		return fmt.Errorf("写入加载文件失败: %v", err)
	}

	// 注册文件
	fe.registerFile(pkgName, "load.go")

	return nil
}

// EmitClassFile 输出类文件，自动管理包与 imports
func (fe *FileEmitterImpl) EmitClassFile(pkgName, fileName string, className string, fields []interface{}, methods []interface{}, header string, body string) error {
	if pkgName == "" || fileName == "" {
		return fmt.Errorf("包名或文件名不能为空")
	}

	// 创建包目录
	pkgDir := filepath.Join(fe.outputRoot, pkgName)
	if err := fe.CreateDirectory(pkgDir); err != nil {
		return fmt.Errorf("创建包目录失败: %v", err)
	}

	// 生成文件头（已由调用方提供，或使用 codeManager 生成）
	full := header
	if full != "" {
		full += "\n\n"
	}
	full += body

	// 写入文件
	filePath := filepath.Join(pkgDir, fileName)
	if err := os.WriteFile(filePath, []byte(full), 0644); err != nil {
		return fmt.Errorf("写入文件失败: %v", err)
	}

	fe.registerFile(pkgName, fileName)
	if fe.pkgClasses[pkgName] == nil {
		fe.pkgClasses[pkgName] = make(map[string]bool)
	}
	fe.pkgClasses[pkgName][className] = true
	return nil
}

// CreateDirectory 创建目录
func (fe *FileEmitterImpl) CreateDirectory(path string) error {
	return os.MkdirAll(path, 0755)
}

// FileExists 检查文件是否存在
func (fe *FileEmitterImpl) FileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// GetOutputRoot 获取输出根目录
func (fe *FileEmitterImpl) GetOutputRoot() string {
	return fe.outputRoot
}

// GetCodeManager 获取内部的代码管理器
func (fe *FileEmitterImpl) GetCodeManager() *CodeManager {
	return fe.codeManager
}

// GetPackageFiles 获取包的所有文件
func (fe *FileEmitterImpl) GetPackageFiles(pkgName string) []string {
	if files, exists := fe.registry[pkgName]; exists {
		return files
	}
	return []string{}
}

// GetAllPackages 获取所有包名
func (fe *FileEmitterImpl) GetAllPackages() []string {
	var packages []string
	for pkg := range fe.registry {
		packages = append(packages, pkg)
	}
	return packages
}

// 注册文件到包
func (fe *FileEmitterImpl) registerFile(pkgName, fileName string) {
	if fe.registry[pkgName] == nil {
		fe.registry[pkgName] = []string{}
	}
	fe.registry[pkgName] = append(fe.registry[pkgName], fileName)
}

// 生成加载文件内容
func (fe *FileEmitterImpl) generateLoadFileContent(pkgName string, functions, classes []string) (string, error) {
	fe.codeManager.AddImport(pkgName, fe.codeManager.mapPackagePath("data"))
	fileHeader := fe.codeManager.GenerateFileHeader(pkgName)
	funcSet := map[string]bool{}
	for _, n := range functions {
		funcSet[n] = true
	}
	if m := fe.pkgFuncs[pkgName]; m != nil {
		for n := range m {
			funcSet[n] = true
		}
	}
	functions = functions[:0]
	for n := range funcSet {
		functions = append(functions, n)
	}
	sort.Strings(functions)
	classSet := map[string]bool{}
	for _, n := range classes {
		classSet[n] = true
	}
	if m := fe.pkgClasses[pkgName]; m != nil {
		for n := range m {
			classSet[n] = true
		}
	}
	classes = classes[:0]
	for n := range classSet {
		// 仅收集确实属于本包且已生成的类文件
		filePath := filepath.Join(fe.outputRoot, pkgName, strings.ToLower(n)+"_class.go")
		if fe.FileExists(filePath) {
			classes = append(classes, n)
		}
	}
	sort.Strings(classes)
	const loadTemplate = `// Load 由生成器产生：注册本包函数与类
func Load(vm data.VM) {
	for _, fun := range []data.FuncStmt{
		{{- range .Functions }}
		New{{.}}Function(),
		{{- end }}
	} {
		vm.AddFunc(fun)
	}
	{{- range .Classes }}
	vm.AddClass(New{{.}}Class())
	{{- end }}
}
`
	data := struct {
		Functions []string
		Classes   []string
	}{
		Functions: functions,
		Classes:   classes,
	}
	tmpl, err := template.New("load").Parse(loadTemplate)
	if err != nil {
		return "", fmt.Errorf("解析加载文件模板失败: %v", err)
	}
	var buf strings.Builder
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("执行加载文件模板失败: %v", err)
	}
	content := fileHeader + "\n\n" + buf.String()
	return content, nil
}

func (fe *FileEmitterImpl) RegisterClass(pkgName, className string) {
	if fe.pkgClasses == nil {
		fe.pkgClasses = make(map[string]map[string]bool)
	}
	if fe.pkgClasses[pkgName] == nil {
		fe.pkgClasses[pkgName] = make(map[string]bool)
	}
	fe.pkgClasses[pkgName][className] = true
}
