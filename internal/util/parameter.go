package util

import v3 "github.com/pb33f/libopenapi/datamodel/high/v3"

type ParameterKey struct {
	name string
	in   string
}

func NewParameterKey(name string, in string) ParameterKey {
	return ParameterKey{
		name: name,
		in:   in,
	}
}

func CopyParameters(src []*v3.Parameter, add ...*v3.Parameter) (dst []*v3.Parameter) {
	copied := map[ParameterKey]struct{}{}
	dst = make([]*v3.Parameter, 0, len(src)+len(add))
	for _, p := range src {
		dst = append(dst, p)
		copied[ParameterKey{name: p.Name, in: p.In}] = struct{}{}
	}
	for _, p := range add {
		if _, ok := copied[ParameterKey{name: p.Name, in: p.In}]; ok {
			continue
		}
		dst = append(dst, p)
	}
	return
}
