package utils

import (
	"errors"
	"fmt"
	"github.com/php-any/origami/data"
)

func ConvertFromIndex[S any](ctx data.Context, index int) (S, error) {
	var opt S
	v, _ := ctx.GetIndexValue(index)
	switch opts := v.(type) {
	case data.GetSource:
		return opts.GetSource().(S), nil
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
		return opts.Value.(S), nil
	case *data.IntValue:
		var a any
		a, err := opts.AsInt()
		if err == nil {
			if opt, ok := a.(S); ok {
				return opt, nil
			}
		}
	case *data.StringValue:
		var a any
		a = opts.AsString()
		if opt, ok := a.(S); ok {
			return opt, nil
		}
	case *data.FloatValue:
		var a any
		a, err := opts.AsFloat()
		if err == nil {
			if opt, ok := a.(S); ok {
				return opt, nil
			}
		}
	case *data.BoolValue:
		var a any
		a, err := opts.AsBool()
		if err == nil {
			if opt, ok := a.(S); ok {
				return opt, nil
			}
		}
	}

	//switch any(opt).(type) {
	//case int:
	//	if opt, ok := a.(S); ok {
	//		return opt, nil
	//	}
	//}

	return opt, errors.New("invalid options type")
}

func Convert[S any](v data.Value) (S, error) {
	var opt S
	switch opts := v.(type) {
	case data.GetSource:
		return opts.GetSource().(S), nil
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
		return opts.Value.(S), nil
	case *data.IntValue:
		var a any
		a, err := opts.AsInt()
		if err == nil {
			if opt, ok := a.(S); ok {
				return opt, nil
			}
		}
	case *data.StringValue:
		var a any
		a = opts.AsString()
		if opt, ok := a.(S); ok {
			return opt, nil
		}
	case *data.FloatValue:
		var a any
		a, err := opts.AsFloat()
		if err == nil {
			if opt, ok := a.(S); ok {
				return opt, nil
			}
		}
	case *data.BoolValue:
		var a any
		a, err := opts.AsBool()
		if err == nil {
			if opt, ok := a.(S); ok {
				return opt, nil
			}
		}
	}

	switch any(opt).(type) {
	case int:
		if v, ok := v.(data.AsInt); ok {
			i, err := v.AsInt()
			return any(i).(S), err
		}
	case string:
		if v, ok := v.(data.AsString); ok {
			i := v.AsString()
			return any(i).(S), nil
		}
	case float32:
		if v, ok := v.(data.AsFloat); ok {
			f64, err := v.AsFloat()
			return any(float32(f64)).(S), err
		}
	case float64:
		if v, ok := v.(data.AsFloat); ok {
			f64, err := v.AsFloat()
			return any(f64).(S), err
		}
	case int8:
		if v, ok := v.(data.AsInt); ok {
			i, err := v.AsInt()
			return any(int8(i)).(S), err
		}
	case int16:
		if v, ok := v.(data.AsInt); ok {
			i, err := v.AsInt()
			return any(int16(i)).(S), err
		}
	case int32:
		if v, ok := v.(data.AsInt); ok {
			i, err := v.AsInt()
			return any(int32(i)).(S), err
		}
	case int64:
		if v, ok := v.(data.AsInt); ok {
			i, err := v.AsInt()
			return any(int64(i)).(S), err
		}
	case uint8:
		if v, ok := v.(data.AsInt); ok {
			i, err := v.AsInt()
			return any(uint8(i)).(S), err
		}
	case uint16:
		if v, ok := v.(data.AsInt); ok {
			i, err := v.AsInt()
			return any(uint16(i)).(S), err
		}
	case uint32:
		if v, ok := v.(data.AsInt); ok {
			i, err := v.AsInt()
			return any(uint32(i)).(S), err
		}
	case uint64:
		if v, ok := v.(data.AsInt); ok {
			i, err := v.AsInt()
			return any(uint64(i)).(S), err
		}
	case bool:
		if v, ok := v.(data.AsBool); ok {
			i, err := v.AsBool()
			return any(i).(S), err
		}
	}

	return opt, errors.New("invalid options type")
}
