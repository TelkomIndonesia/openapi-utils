package config

import (
	"errors"
	"fmt"
	"os"
	"path"
	"regexp"
	"strings"
	"sync"

	"github.com/pb33f/libopenapi"
	v3 "github.com/pb33f/libopenapi/datamodel/high/v3"
)

type Proxies map[string]Proxy

type Proxy struct {
	Name *string `json:"name" yaml:"name"`
	Spec string  `json:"spec" yaml:"spec"`

	once  sync.Once
	doc   libopenapi.Document
	v3Doc v3.Document
}

func (p *Proxy) buildOpenapiDocument() (err error) {
	p.once.Do(func() {
		var b []byte
		b, err = os.ReadFile(p.Spec)
		if err != nil {
			err = fmt.Errorf("fail to read openapi spec: %w", err)
			return
		}

		p.doc, err = libopenapi.NewDocument(b)
		if err != nil {
			err = fmt.Errorf("fail to build openapi doc: %w", err)
			return
		}

		d, errs := p.doc.BuildV3Model()
		if err = errors.Join(errs...); err != nil {
			err = fmt.Errorf("fail to build v3 openapi doc: %w", err)
			return
		}

		p.v3Doc = d.Model
	})
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

func (p *Proxy) GetOpenAPIModel() (doc v3.Document, err error) {
	if p.doc == nil {
		err = p.buildOpenapiDocument()
		if err != nil {
			return doc, err
		}
	}

	return p.v3Doc, nil
}

type ProxyOperation struct {
	*Proxy `json:",inline" yaml:",inline"`

	Path   string `json:"path" yaml:"path"`
	Method string `json:"method" yaml:"method"`
	Inject Inject `json:"inject" yaml:"inject"`
}

var nonAlphaNum = regexp.MustCompile("[^a-zA-Z0-9]")

func (p ProxyOperation) GetName() string {
	if p.Name == nil {
		name, _ := strings.CutSuffix(path.Base(p.Spec), path.Ext(p.Spec))
		return string(nonAlphaNum.ReplaceAll([]byte(name), nil))
	}
	return *p.Name
}

type Inject struct {
	Parameters []*ExcludedParameter `json:"parameters,omitempty" yaml:"parameters,omitempty"`
}

type ExcludedParameter struct {
	Name string `json:"name,omitempty" yaml:"name,omitempty"`
	In   string `json:"in,omitempty" yaml:"in,omitempty"`
}
