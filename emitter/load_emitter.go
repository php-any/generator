package emitter

import (
	"fmt"
	"sort"
	"strings"
)

// LoadEmitter 根据注册器内容为指定包生成 load.go

type LoadEmitter struct {
	codeManager *CodeManager
	registry    *RegistryImpl
}

func NewLoadEmitter(cm *CodeManager, reg *RegistryImpl) *LoadEmitter {
	return &LoadEmitter{codeManager: cm, registry: reg}
}

func (le *LoadEmitter) Generate(pkg string) (string, error) {
	header := le.codeManager.GenerateFileHeader(pkg)
	funcs := le.registry.GetFunctions(pkg)
	classes := le.registry.GetClasses(pkg)
	sort.Strings(funcs)
	sort.Strings(classes)

	var b strings.Builder
	b.WriteString(header)
	b.WriteString("\n\n// LoadPackage 加载包\n")
	b.WriteString("func LoadPackage() map[string]interface{} {\n")
	b.WriteString("\tpkg := make(map[string]interface{})\n\n")
	b.WriteString("\t// 注册函数\n")
	for _, fn := range funcs {
		b.WriteString(fmt.Sprintf("\tpkg[\"%s\"] = New%sFunction()\n", fn, fn))
	}
	b.WriteString("\n\t// 注册类\n")
	for _, cls := range classes {
		b.WriteString(fmt.Sprintf("\tpkg[\"%s\"] = New%sClass()\n", cls, cls))
	}
	b.WriteString("\n\treturn pkg\n")
	b.WriteString("}\n")
	return b.String(), nil
}
