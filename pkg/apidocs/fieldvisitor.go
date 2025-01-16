package apidocs

import (
	"sort"
	"strings"

	"k8s.io/kube-openapi/pkg/util/proto"
	"k8s.io/kubectl/pkg/explain"
)

type schemaVisitor struct {
	prevPath          string
	pathSchema        map[string]proto.Schema
	err               error
	visitedReferences map[string]struct{}
}

var _ proto.SchemaVisitor = (*schemaVisitor)(nil)

func (v *schemaVisitor) VisitKind(k *proto.Kind) {
	keys := k.Keys()
	paths := make([]string, len(keys))
	for i, key := range keys {
		paths[i] = strings.Join([]string{v.prevPath, key}, ".")
	}
	for i, key := range keys {
		schema, err := explain.LookupSchemaForField(k, []string{key})
		if err != nil {
			v.err = err
			return
		}
		v.pathSchema[paths[i]] = schema
		v.prevPath = paths[i]
		schema.Accept(v)
	}
}

func (v *schemaVisitor) VisitReference(r proto.Reference) {
	if _, ok := v.visitedReferences[r.Reference()]; ok {
		return
	}
	v.visitedReferences[r.Reference()] = struct{}{}
	r.SubSchema().Accept(v)
	delete(v.visitedReferences, r.Reference())
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

func (v *schemaVisitor) listPaths() []string {
	paths := make([]string, 0, len(v.pathSchema))
	for path := range v.pathSchema {
		paths = append(paths, path)
	}
	sort.SliceStable(paths, func(i, j int) bool {
		return paths[i] < paths[j]
	})
	return paths
}
