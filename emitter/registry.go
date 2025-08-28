package emitter

import "sync"

// RegistryImpl 简单注册器，记录每个包下的已注册项
// 用于生成 load.go 或构建索引时防止重复

type RegistryImpl struct {
	mu        sync.RWMutex
	functions map[string]map[string]bool            // pkg -> funcName
	classes   map[string]map[string]bool            // pkg -> className
	methods   map[string]map[string]map[string]bool // pkg -> class -> method
}

func NewRegistry() *RegistryImpl {
	return &RegistryImpl{
		functions: make(map[string]map[string]bool),
		classes:   make(map[string]map[string]bool),
		methods:   make(map[string]map[string]map[string]bool),
	}
}

func (r *RegistryImpl) RegisterFunction(pkg, name string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.functions[pkg] == nil {
		r.functions[pkg] = map[string]bool{}
	}
	r.functions[pkg][name] = true
}

func (r *RegistryImpl) RegisterClass(pkg, name string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.classes[pkg] == nil {
		r.classes[pkg] = map[string]bool{}
	}
	r.classes[pkg][name] = true
}

func (r *RegistryImpl) RegisterMethod(pkg, class, method string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.methods[pkg] == nil {
		r.methods[pkg] = map[string]map[string]bool{}
	}
	if r.methods[pkg][class] == nil {
		r.methods[pkg][class] = map[string]bool{}
	}
	r.methods[pkg][class][method] = true
}

func (r *RegistryImpl) GetFunctions(pkg string) []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var out []string
	for name := range r.functions[pkg] {
		out = append(out, name)
	}
	return out
}

func (r *RegistryImpl) GetClasses(pkg string) []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var out []string
	for name := range r.classes[pkg] {
		out = append(out, name)
	}
	return out
}

func (r *RegistryImpl) GetMethods(pkg, class string) []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var out []string
	for name := range r.methods[pkg][class] {
		out = append(out, name)
	}
	return out
}

func (r *RegistryImpl) ClearPackage(pkg string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.functions, pkg)
	delete(r.classes, pkg)
	delete(r.methods, pkg)
}

func (r *RegistryImpl) Clear() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.functions = map[string]map[string]bool{}
	r.classes = map[string]map[string]bool{}
	r.methods = map[string]map[string]map[string]bool{}
}
