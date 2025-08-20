package generator

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

var (
	modulePathOnce sync.Once
	modulePath     string
)

// getModulePath returns the current module path by reading go.mod in the repo root.
// Fallback to empty string if not found.
func getModulePath() string {
	modulePathOnce.Do(func() {
		// search go.mod from current working directory upwards
		dir, _ := os.Getwd()
		for i := 0; i < 10 && dir != ""; i++ {
			gm := filepath.Join(dir, "go.mod")
			if f, err := os.Open(gm); err == nil {
				s := bufio.NewScanner(f)
				for s.Scan() {
					line := strings.TrimSpace(s.Text())
					if strings.HasPrefix(line, "module ") {
						modulePath = strings.TrimSpace(strings.TrimPrefix(line, "module "))
						break
					}
				}
				_ = f.Close()
				break
			}
			parent := filepath.Dir(dir)
			if parent == dir {
				break
			}
			dir = parent
		}
	})
	return modulePath
}


