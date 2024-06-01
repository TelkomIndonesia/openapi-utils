package proxy

import (
	"strings"

	"github.com/pb33f/libopenapi"
	"github.com/pb33f/libopenapi/datamodel/high/base"
	v3 "github.com/pb33f/libopenapi/datamodel/high/v3"
	"github.com/pb33f/libopenapi/orderedmap"
	"gopkg.in/yaml.v3"
)

func initComponents(doc *libopenapi.DocumentModel[v3.Document]) {
	if doc.Model.Components == nil {
		doc.Model.Components = &v3.Components{}
	}
	if doc.Model.Components.Schemas == nil {
		doc.Model.Components.Schemas = &orderedmap.Map[string, *base.SchemaProxy]{}
	}
	if doc.Model.Components.Parameters == nil {
		doc.Model.Components.Parameters = &orderedmap.Map[string, *v3.Parameter]{}
	}
	if doc.Model.Components.RequestBodies == nil {
		doc.Model.Components.RequestBodies = &orderedmap.Map[string, *v3.RequestBody]{}
	}
	if doc.Model.Components.Responses == nil {
		doc.Model.Components.Responses = &orderedmap.Map[string, *v3.Response]{}
	}
	if doc.Model.Components.Headers == nil {
		doc.Model.Components.Headers = &orderedmap.Map[string, *v3.Header]{}
	}
	if doc.Model.Components.Links == nil {
		doc.Model.Components.Links = &orderedmap.Map[string, *v3.Link]{}
	}
	if doc.Model.Components.SecuritySchemes == nil {
		doc.Model.Components.SecuritySchemes = &orderedmap.Map[string, *v3.SecurityScheme]{}
	}
	if doc.Model.Components.Examples == nil {
		doc.Model.Components.Examples = &orderedmap.Map[string, *base.Example]{}
	}
	if doc.Model.Components.Extensions == nil {
		doc.Model.Components.Extensions = &orderedmap.Map[string, *yaml.Node]{}
	}
	if doc.Model.Components.Callbacks == nil {
		doc.Model.Components.Callbacks = &orderedmap.Map[string, *v3.Callback]{}
	}
}

func getOperationsMap(pi *v3.PathItem) (ops map[string]*v3.Operation) {
	ops = map[string]*v3.Operation{}
	if pi.Get != nil {
		ops["get"] = pi.Get
	}
	if pi.Delete != nil {
		ops["delete"] = pi.Delete
	}
	if pi.Post != nil {
		ops["post"] = pi.Post
	}
	if pi.Put != nil {
		ops["put"] = pi.Put
	}
	if pi.Patch != nil {
		ops["patch"] = pi.Patch
	}
	if pi.Options != nil {
		ops["options"] = pi.Options
	}
	if pi.Head != nil {
		ops["head"] = pi.Head
	}
	if pi.Trace != nil {
		ops["trace"] = pi.Trace
	}
	return
}

func getOperation(pi *v3.PathItem, method string) *v3.Operation {
	switch {
	case strings.EqualFold("get", method):
		return pi.Get
	case strings.EqualFold("delete", method):
		return pi.Delete
	case strings.EqualFold("post", method):
		return pi.Post
	case strings.EqualFold("put", method):
		return pi.Put
	case strings.EqualFold("patch", method):
		return pi.Patch
	case strings.EqualFold("options", method):
		return pi.Options
	case strings.EqualFold("head", method):
		return pi.Head
	case strings.EqualFold("trace", method):
		return pi.Trace
	}
	return nil
}

func setOperation(pi *v3.PathItem, method string, val *v3.Operation) {
	switch {
	case strings.EqualFold("get", method):
		pi.Get = val
	case strings.EqualFold("delete", method):
		pi.Delete = val
	case strings.EqualFold("post", method):
		pi.Post = val
	case strings.EqualFold("put", method):
		pi.Put = val
	case strings.EqualFold("patch", method):
		pi.Patch = val
	case strings.EqualFold("options", method):
		pi.Options = val
	case strings.EqualFold("head", method):
		pi.Head = val
	case strings.EqualFold("trace", method):
		pi.Trace = val
	}
}

type parameterKey struct {
	name string
	in   string
}

func copyParameters(src []*v3.Parameter, add ...*v3.Parameter) (dst []*v3.Parameter) {
	copied := map[parameterKey]struct{}{}
	dst = make([]*v3.Parameter, 0, len(src)+len(add))
	for _, p := range src {
		dst = append(dst, p)
		copied[parameterKey{name: p.Name, in: p.In}] = struct{}{}
	}
	for _, p := range add {
		if _, ok := copied[parameterKey{name: p.Name, in: p.In}]; ok {
			continue
		}
		dst = append(dst, p)
	}
	return
}
