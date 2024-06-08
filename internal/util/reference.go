package util

import (
	"strings"

	"github.com/pb33f/libopenapi/datamodel/high/base"
	"github.com/pb33f/libopenapi/index"
)

func LocalizeReference(ref *index.Reference, prefix string) {
	name := prefix + ref.Name
	refdef := strings.TrimSuffix(ref.Definition, ref.Name) + name
	ref.Node.Content = base.CreateSchemaProxyRef(refdef).GetReferenceNode().Content
}
