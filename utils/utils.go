package utils

import (
	"fmt"
	"github.com/php-any/origami/data"
)

func ConvertFromIndex[S any](ctx data.Context, index int) (S, error) {
	var opt S
	v, _ := ctx.GetIndexValue(index)
	switch opts := v.(type) {
	case data.GetSource:
		opt = opts.GetSource().(S)
	case *data.ClassValue:
		if p, ok := opts.Class.(data.GetSource); ok {
			// 检查 GetSource 返回的类型，如果是指针则解引用
			if src := p.GetSource(); src != nil {
				if ptr, ok := src.(S); ok {
					return ptr, nil
				}
			}
		}
		return opt, fmt.Errorf("invalid options type: %T", opts)
	case *data.AnyValue:
		opt = opts.Value.(S)
	}
	return opt, nil
}
