package emitter

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/template"
)

// FileEmitterImpl 文件输出器实现
type FileEmitterImpl struct {
	outputRoot  string
	registry    map[string][]string // 包名 -> 文件列表
	codeManager *CodeManager
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
	// 使用代码管理器生成文件头部
	fileHeader := fe.codeManager.GenerateFileHeader(pkgName)

	// 加载文件模板 - 不包含硬编码的import
	const loadTemplate = `// LoadPackage 加载包
func LoadPackage() map[string]interface{} {
	pkg := make(map[string]interface{})

	// 注册函数
	{{range .Functions}}
	pkg["{{.}}"] = New{{.}}Function()
	{{end}}

	// 注册类
	{{range .Classes}}
	pkg["{{.}}"] = New{{.}}Class()
	{{end}}

	return pkg
}

// GetFunction 获取函数
func GetFunction(name string) (interface{}, bool) {
	pkg := LoadPackage()
	fn, exists := pkg[name]
	return fn, exists
}

// GetClass 获取类
func GetClass(name string) (interface{}, bool) {
	pkg := LoadPackage()
	cls, exists := pkg[name]
	return cls, exists
}
`

	// 准备模板数据
	data := struct {
		Functions []string
		Classes   []string
	}{
		Functions: functions,
		Classes:   classes,
	}

	// 执行模板
	tmpl, err := template.New("load").Parse(loadTemplate)
	if err != nil {
		return "", fmt.Errorf("解析加载文件模板失败: %v", err)
	}

	var buf strings.Builder
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("执行加载文件模板失败: %v", err)
	}

	// 组合文件头部和内容
	content := fileHeader + "\n\n" + buf.String()
	return content, nil
}
