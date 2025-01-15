package app

import (
	"sort"
	"strings"

	"k8s.io/kube-openapi/pkg/util/proto"
	"k8s.io/kubectl/pkg/explain"
)

//////////////////////////////////////////////////////////////////////
// schema visitor

type path struct {
	original string
}

type schemaVisitor struct {
	prevPath   path
	pathSchema map[path]proto.Schema
	err        error
}

var _ proto.SchemaVisitor = (*schemaVisitor)(nil)

func (v *schemaVisitor) VisitKind(k *proto.Kind) {
	keys := k.Keys()
	paths := make([]path, len(keys))
	for i, key := range keys {
		paths[i] = path{
			original: strings.Join([]string{v.prevPath.original, key}, "."),
		}
	}
	for i, key := range keys {
		schema, err := explain.LookupSchemaForField(k, []string{key})
		if err != nil {
			v.err = err
			return
		}
		// if _, ok := schema.(*proto.Array); ok {
		// 	// TODO: types for print on UI?
		// }
		v.pathSchema[paths[i]] = schema
		v.prevPath = paths[i]
		schema.Accept(v)
	}
}

var visitedReferences = map[string]struct{}{}

func (v *schemaVisitor) VisitReference(r proto.Reference) {
	if _, ok := visitedReferences[r.Reference()]; ok {
		return
	}
	visitedReferences[r.Reference()] = struct{}{}
	r.SubSchema().Accept(v)
	delete(visitedReferences, r.Reference())
}

func (*schemaVisitor) VisitPrimitive(*proto.Primitive) {
	// Nothing to do.
}

func (v *schemaVisitor) VisitArray(a *proto.Array) {
	a.SubType.Accept(v)
}

func (v *schemaVisitor) VisitMap(m *proto.Map) {
	m.SubType.Accept(v)
}

func (v *schemaVisitor) listPaths() []path {
	paths := make([]path, 0, len(v.pathSchema))
	for path := range v.pathSchema {
		paths = append(paths, path)
	}
	sort.SliceStable(paths, func(i, j int) bool {
		return paths[i].original < paths[j].original
	})
	return paths
}
