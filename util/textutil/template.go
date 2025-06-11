package textutil

import (
	"bytes"
	"context"
	"text/template"

	"github.com/groundhogchain/chuckd/lib/must"
)

type M map[string]any

func Apply(ctx context.Context, templateStr string, values map[string]interface{}) string {

	tmpl, err := template.New("").Parse(templateStr)
	must.NoError(ctx, err)

	var buf bytes.Buffer
	err = tmpl.Execute(&buf, values)
	must.NoError(ctx, err)

	return buf.String()
}
