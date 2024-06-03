package util

import (
	"strings"

	v3 "github.com/pb33f/libopenapi/datamodel/high/v3"
)

func GetOperationsMap(p *v3.PathItem) (ops map[string]*v3.Operation) {
	ops = map[string]*v3.Operation{}
	if p.Get != nil {
		ops["get"] = p.Get
	}
	if p.Delete != nil {
		ops["delete"] = p.Delete
	}
	if p.Post != nil {
		ops["post"] = p.Post
	}
	if p.Put != nil {
		ops["put"] = p.Put
	}
	if p.Patch != nil {
		ops["patch"] = p.Patch
	}
	if p.Options != nil {
		ops["options"] = p.Options
	}
	if p.Head != nil {
		ops["head"] = p.Head
	}
	if p.Trace != nil {
		ops["trace"] = p.Trace
	}
	return
}

func GetOperation(p *v3.PathItem, method string) *v3.Operation {
	switch {
	case strings.EqualFold("get", method):
		return p.Get
	case strings.EqualFold("delete", method):
		return p.Delete
	case strings.EqualFold("post", method):
		return p.Post
	case strings.EqualFold("put", method):
		return p.Put
	case strings.EqualFold("patch", method):
		return p.Patch
	case strings.EqualFold("options", method):
		return p.Options
	case strings.EqualFold("head", method):
		return p.Head
	case strings.EqualFold("trace", method):
		return p.Trace
	}
	return nil
}

func SetOperation(p *v3.PathItem, method string, val *v3.Operation) {
	switch {
	case strings.EqualFold("get", method):
		p.Get = val
	case strings.EqualFold("delete", method):
		p.Delete = val
	case strings.EqualFold("post", method):
		p.Post = val
	case strings.EqualFold("put", method):
		p.Put = val
	case strings.EqualFold("patch", method):
		p.Patch = val
	case strings.EqualFold("options", method):
		p.Options = val
	case strings.EqualFold("head", method):
		p.Head = val
	case strings.EqualFold("trace", method):
		p.Trace = val
	}
}
