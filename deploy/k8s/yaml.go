package k8s

import (
	"bytes"
	"context"

	"github.com/goccy/go-yaml"

	"github.com/groundhogchain/chuckd/lib/must"
	"k8s.io/apimachinery/pkg/runtime"
)

func SprintResources(ctx context.Context, resources []runtime.Object) string {

	var w bytes.Buffer

	for i, resource := range resources {
		yamlBytes, err := yaml.Marshal(resource)
		must.NoError(ctx, err)

		_, err = w.Write(yamlBytes)
		must.NoError(ctx, err)

		// add separator between resources
		if i < len(resources)-1 {
			_, err = w.WriteString("---\n")
			must.NoError(ctx, err)
		}
	}

	return w.String()
}
