package proxy

import (
	"errors"
	"fmt"
	"os"
	"path"
	"regexp"
	"strings"

	"github.com/oapi-codegen/oapi-codegen/v2/pkg/codegen"
	"github.com/pb33f/libopenapi"
	v3 "github.com/pb33f/libopenapi/datamodel/high/v3"
	"github.com/telkomindonesia/openapi-utils/internal/util"
)

type Proxies map[string]Proxy

type Proxy struct {
	Name string `json:"name" yaml:"name"`
	Spec string `json:"spec" yaml:"spec"`

	doc libopenapi.Document
}

func (p *Proxy) buildOpenapiDocument() (err error) {
	var b []byte
	b, err = os.ReadFile(p.Spec)
	if err != nil {
		err = fmt.Errorf("fail to read openapi spec: %w", err)
		return
	}

	doc, err := libopenapi.NewDocument(b)
	if err != nil {
		return fmt.Errorf("fail to build openapi doc: %w", err)
	}

	_, errs := doc.BuildV3Model()
	if err = errors.Join(errs...); err != nil {
		return fmt.Errorf("fail to build v3 openapi doc: %w", err)
	}

	p.doc = doc
	return
}

func (p *Proxy) GetOpenAPIDoc() (doc libopenapi.Document, err error) {
	if p.doc == nil {
		err = p.buildOpenapiDocument()
		if err != nil {
			return doc, err
		}
	}

	return p.doc, nil
}

func (p *Proxy) GetOpenAPIV3Doc() (docv3 *libopenapi.DocumentModel[v3.Document], err error) {
	doc, err := p.GetOpenAPIDoc()
	if err != nil {
		return
	}

	docv3, errs := doc.BuildV3Model()
	err = errors.Join(errs...)
	return
}

var nonAlphaNum = regexp.MustCompile("[^a-zA-Z0-9]")

func (p Proxy) GetName() string {
	name := p.Name
	if name == "" {
		name, _ = strings.CutSuffix(path.Base(p.Spec), path.Ext(p.Spec))
	}
	return codegen.UppercaseFirstCharacter(name)
}

type ProxyOperation struct {
	*Proxy `json:",inline" yaml:",inline"`

	Path   string `json:"path" yaml:"path"`
	Method string `json:"method" yaml:"method"`
	Inject Inject `json:"inject" yaml:"inject"`

	up      *v3.PathItem
	uop     *v3.Operation
	uparams []*v3.Parameter
}

func (pop ProxyOperation) WithReloadedDoc(doc libopenapi.Document) ProxyOperation {
	npop := ProxyOperation{
		Proxy: &Proxy{
			doc: doc,
		},
		Path:   pop.Path,
		Method: pop.Method,
		Inject: pop.Inject,
	}
	if pop.Proxy != nil {
		npop.Name = pop.Name
		npop.Spec = pop.Spec
	}
	return npop
}

func (pop *ProxyOperation) GetUpstreamOperation() (uop *v3.Operation, err error) {
	if pop.uop == nil {
		doc, err := pop.GetOpenAPIDoc()
		if err != nil {
			return nil, fmt.Errorf("fail to load `x-proxy` :%w", err)
		}

		docv3, _ := doc.BuildV3Model()
		up, ok := docv3.Model.Paths.PathItems.Get(pop.Path)
		if !ok {
			return nil, fmt.Errorf("path '%s' not found inside upstream doc", pop.Path)
		}

		uop = util.GetOperation(up, pop.Method)
		if uop == nil {
			return nil, fmt.Errorf("operation '%s %s' not found inside upstream doc", pop.Method, pop.Path)
		}

		pop.up = up
		pop.uop = uop
	}

	return pop.uop, nil
}

func (pop *ProxyOperation) GetProxiedParameters() (uparams []*v3.Parameter, err error) {
	if pop.uparams == nil {
		if _, err = pop.GetUpstreamOperation(); err != nil {
			return nil, err
		}

		injectedParamMap := map[util.ParameterKey]struct{}{}
		for _, p := range pop.Inject.Parameters {
			injectedParamMap[util.NewParameterKey(p.Name, p.In)] = struct{}{}
		}
		for _, p := range util.CopyParameters(pop.uop.Parameters, pop.up.Parameters...) {
			if _, ok := injectedParamMap[util.NewParameterKey(p.Name, p.In)]; ok {
				continue
			}
			pop.uparams = append(pop.uparams, p)
		}

	}

	return pop.uparams, nil
}

type Inject struct {
	Parameters []*ExcludedParameter `json:"parameters,omitempty" yaml:"parameters,omitempty"`
}

type ExcludedParameter struct {
	Name string `json:"name,omitempty" yaml:"name,omitempty"`
	In   string `json:"in,omitempty" yaml:"in,omitempty"`
}
