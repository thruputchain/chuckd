package protoutil

import (
	"bytes"
	"encoding/base64"
)

func EncodeBase64(v []byte) string {
	var w bytes.Buffer
	enc := base64.NewEncoder(base64.StdEncoding, &w)
	enc.Write(v)
	enc.Close()
	return w.String()
}
