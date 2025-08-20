package generator

import (
	"unicode"
	"unicode/utf8"
)

func isExportedName(name string) bool {
	if name == "" {
		return false
	}
	r, _ := utf8.DecodeRuneInString(name)
	return unicode.IsUpper(r)
}
