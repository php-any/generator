package generator

import (
	"sort"
	"sync"
)

// 维护每个包下已生成的类与函数，用于统一生成 load.go
type packageRegistry struct {
	classes   map[string]bool
	functions map[string]bool
}

var (
	pkgRegistry   = make(map[string]*packageRegistry)
	pkgRegistryMu sync.Mutex
)

func registerClass(pkgName, typeName string) {
	pkgRegistryMu.Lock()
	defer pkgRegistryMu.Unlock()
	r := pkgRegistry[pkgName]
	if r == nil {
		r = &packageRegistry{classes: make(map[string]bool), functions: make(map[string]bool)}
		pkgRegistry[pkgName] = r
	}
	r.classes[typeName] = true
}

func registerFunction(pkgName, funcName string) {
	pkgRegistryMu.Lock()
	defer pkgRegistryMu.Unlock()
	r := pkgRegistry[pkgName]
	if r == nil {
		r = &packageRegistry{classes: make(map[string]bool), functions: make(map[string]bool)}
		pkgRegistry[pkgName] = r
	}
	r.functions[funcName] = true
}

func listRegistered(pkgName string) (classes []string, functions []string) {
	pkgRegistryMu.Lock()
	r := pkgRegistry[pkgName]
	pkgRegistryMu.Unlock()
	if r != nil {
		for k := range r.classes {
			classes = append(classes, k)
		}
		for k := range r.functions {
			functions = append(functions, k)
		}
	}
	sort.Strings(classes)
	sort.Strings(functions)
	return
}
