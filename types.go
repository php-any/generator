package generator

import "reflect"

// collectPkgPaths collects only the base package paths needed for type assertions.
// It avoids deep traversal into struct fields or interface method sets to prevent
// bringing in unused imports.
func collectPkgPaths(t reflect.Type, dest map[string]bool) {
    visited := make(map[reflect.Type]bool)
    collectBasePkgPaths(t, dest, visited)
}

func collectBasePkgPaths(t reflect.Type, dest map[string]bool, visited map[reflect.Type]bool) {
    if t == nil {
        return
    }
    if visited[t] {
        return
    }
    visited[t] = true

    // Dereference pointers
    for t.Kind() == reflect.Ptr {
        t = t.Elem()
        if visited[t] {
            return
        }
        visited[t] = true
    }

    // Record this type's package if any
    if t.PkgPath() != "" {
        dest[t.PkgPath()] = true
    }

    switch t.Kind() {
    case reflect.Slice, reflect.Array, reflect.Chan:
        collectBasePkgPaths(t.Elem(), dest, visited)
    case reflect.Map:
        collectBasePkgPaths(t.Key(), dest, visited)
        collectBasePkgPaths(t.Elem(), dest, visited)
    case reflect.Func:
        // Only record packages of direct in/out types, not their internal fields
        for i := 0; i < t.NumIn(); i++ {
            collectBasePkgPaths(t.In(i), dest, visited)
        }
        for i := 0; i < t.NumOut(); i++ {
            collectBasePkgPaths(t.Out(i), dest, visited)
        }
    case reflect.Struct:
        // Do not traverse fields; asserting a struct type only needs its package
        return
    case reflect.Interface:
        // Do not traverse interface methods
        return
    default:
        return
    }
}


