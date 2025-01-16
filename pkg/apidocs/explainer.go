package apidocs

import (
	"fmt"
	"io"
	"strings"

	"k8s.io/apimachinery/pkg/runtime/schema"
	openapiclient "k8s.io/client-go/openapi"
	explainv2 "k8s.io/kubectl/pkg/explain/v2"
)

type Explainer struct {
	gvr           schema.GroupVersionResource
	openAPIClient openapiclient.Client
}

func (e Explainer) Explain(w io.Writer, path string) error {
	if path == "" {
		return fmt.Errorf("empty path is not allowed for explain: %s", path)
	}
	fields := strings.Split(path, ".")
	if len(fields) > 0 {
		// Skip resource name
		fields = fields[1:]
	}

	return explainv2.PrintModelDescription(
		fields,
		w,
		e.openAPIClient,
		e.gvr,
		false,
		"plaintext",
	)
}
