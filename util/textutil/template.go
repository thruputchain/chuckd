package textutil

import (
	"bytes"
	"context"
	"text/template"

	"github.com/groundhogchain/lib-go/must"
)

type M map[string]any

func Apply(ctx context.Context, tmplStr string, values map[string]interface{}) string {

	tmpl, err := template.New("").Parse(tmplStr)
	must.NoError(ctx, err)

	var buf bytes.Buffer
	err = tmpl.Execute(&buf, values)
	must.NoError(ctx, err)

	return buf.String()
}
