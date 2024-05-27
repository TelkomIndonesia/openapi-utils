package config

import (
	"path"
	"regexp"
	"strings"
)

type Proxy struct {
	Spec   string  `json:"spec" yaml:"spec"`
	Path   string  `json:"path" yaml:"path"`
	Method string  `json:"method" yaml:"method"`
	Suffix *string `json:"suffix" yaml:"suffix"`
	Inject Inject  `json:"inject" yaml:"inject"`
}

var nonAlphaNum = regexp.MustCompile("[^a-zA-Z0-9]")

func (p Proxy) GetSuffix() string {
	if p.Suffix == nil {
		name, _ := strings.CutSuffix(path.Base(p.Spec), path.Ext(p.Spec))
		return string(nonAlphaNum.ReplaceAll([]byte(name), nil))
	}
	return *p.Suffix
}

type Inject struct {
	Parameters []*ExcludedParameter `json:"parameters,omitempty" yaml:"parameters,omitempty"`
}

type ExcludedParameter struct {
	Name string `json:"name,omitempty" yaml:"name,omitempty"`
	In   string `json:"in,omitempty" yaml:"in,omitempty"`
}
