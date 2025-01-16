package apidocs

import (
	"strings"

	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/kube-openapi/pkg/util/proto"
	"k8s.io/kubectl/pkg/util/openapi"
)

// TODO: cache somehow

func getPaths(restMapper meta.RESTMapper,
	openAPISchema openapi.Resources,
	gvr schema.GroupVersionResource,
) ([]string, error) {
	visitor := &schemaVisitor{
		pathSchema:        make(map[string]proto.Schema),
		prevPath:          strings.ToLower(gvr.Resource),
		err:               nil,
		visitedReferences: make(map[string]struct{}),
	}
	gvk, err := restMapper.KindFor(gvr)
	if err != nil {
		return nil, err
	}
	protoSchema := openAPISchema.LookupResource(gvk)
	if protoSchema == nil {
		return nil, err
	}
	protoSchema.Accept(visitor)
	if visitor.err != nil {
		return nil, err
	}
	visitorPathsResult := visitor.getVisitedPaths()
	return visitorPathsResult, nil
}
