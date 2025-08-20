package generator

import (
	"fmt"
	"reflect"
)

type FunctionInfo struct {
	RuntimeName string
	Signature   string
	IsVariadic  bool
	ParamTypes  []reflect.Type
	ReturnTypes []reflect.Type
}

func InspectFunction(fn any) (*FunctionInfo, error) {
	val := reflect.ValueOf(fn)
	typ := val.Type()
	if typ.Kind() != reflect.Func {
		return nil, fmt.Errorf("不是函数: %T", fn)
	}

	info := &FunctionInfo{
		Signature:   typ.String(),
		IsVariadic:  typ.IsVariadic(),
		ParamTypes:  make([]reflect.Type, typ.NumIn()),
		ReturnTypes: make([]reflect.Type, typ.NumOut()),
	}

	for i := 0; i < typ.NumIn(); i++ {
		info.ParamTypes[i] = typ.In(i)
	}
	for i := 0; i < typ.NumOut(); i++ {
		info.ReturnTypes[i] = typ.Out(i)
	}
	return info, nil
}
