package proxy

import (
	"strings"

	v3 "github.com/pb33f/libopenapi/datamodel/high/v3"
)

func getOperationsMap(pi *v3.PathItem) (ops map[string]*v3.Operation) {
	ops = map[string]*v3.Operation{}
	if pi.Get != nil {
		ops["Get"] = pi.Get
	}
	if pi.Delete != nil {
		ops["Delete"] = pi.Delete
	}
	if pi.Post != nil {
		ops["Post"] = pi.Post
	}
	if pi.Patch != nil {
		ops["Patch"] = pi.Patch
	}
	if pi.Options != nil {
		ops["Options"] = pi.Options
	}
	if pi.Head != nil {
		ops["Head"] = pi.Head
	}
	if pi.Trace != nil {
		ops["Trace"] = pi.Trace
	}
	return
}

func getOperation(pi *v3.PathItem, method string) *v3.Operation {
	switch {
	case strings.EqualFold("Get", method):
		return pi.Get
	case strings.EqualFold("Delete", method):
		return pi.Delete
	case strings.EqualFold("Post", method):
		return pi.Post
	case strings.EqualFold("Patch", method):
		return pi.Patch
	case strings.EqualFold("Options", method):
		return pi.Options
	case strings.EqualFold("Head", method):
		return pi.Head
	case strings.EqualFold("Trace", method):
		return pi.Trace
	}
	return nil
}

func setOperation(pi *v3.PathItem, method string, val *v3.Operation) {
	switch {
	case strings.EqualFold("Get", method):
		pi.Get = val
	case strings.EqualFold("Delete", method):
		pi.Delete = val
	case strings.EqualFold("Post", method):
		pi.Post = val
	case strings.EqualFold("Patch", method):
		pi.Patch = val
	case strings.EqualFold("Options", method):
		pi.Options = val
	case strings.EqualFold("Head", method):
		pi.Head = val
	case strings.EqualFold("Trace", method):
		pi.Trace = val
	}
}
