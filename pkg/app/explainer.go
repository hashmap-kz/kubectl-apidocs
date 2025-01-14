package app

import (
	"fmt"
	"io"
	"strings"

	"k8s.io/apimachinery/pkg/runtime/schema"

	openapiclient "k8s.io/client-go/openapi"
	explainv2 "k8s.io/kubectl/pkg/explain/v2"
)

//////////////////////////////////////////////////////////////////////
// explainer

type Explainer struct {
	gvr                 schema.GroupVersionResource
	openAPIV3Client     openapiclient.Client
	enablePrintPath     bool
	enablePrintBrackets bool
}

func (e Explainer) Explain(w io.Writer, path string) error {
	if len(path) == 0 {
		return fmt.Errorf("path must not be empty: %#v", path)
	}
	fields := strings.Split(path, ".")
	if len(fields) > 0 {
		// Remove resource name
		fields = fields[1:]
	}

	return explainv2.PrintModelDescription(
		fields,
		w,
		e.openAPIV3Client,
		e.gvr,
		false,
		"plaintext",
	)
}
