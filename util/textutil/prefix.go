package textutil

import (
	"bytes"
	"io"
)

type PrefixWriter struct {
	writer io.Writer
	prefix []byte
	buffer bytes.Buffer
}

func NewPrefixWriter(w io.Writer, prefix string) *PrefixWriter {
	return &PrefixWriter{
		writer: w,
		prefix: []byte(prefix),
	}
}

func (pw *PrefixWriter) Write(p []byte) (int, error) {
	originalLen := len(p)

	for len(p) > 0 {
		newlineIndex := bytes.IndexByte(p, '\n')

		if newlineIndex == -1 {
			pw.buffer.Write(p)
			break
		}

		_, err := pw.buffer.Write(p[:newlineIndex+1])
		if err != nil {
			return 0, err
		}
		p = p[newlineIndex+1:]

		if err = pw.Flush(); err != nil {
			return 0, err
		}
	}

	return originalLen, nil
}

func (pw *PrefixWriter) Flush() error {
	if pw.buffer.Len() > 0 {
		q := make([]byte, len(pw.prefix)+pw.buffer.Len())
		copy(q[:len(pw.prefix)], pw.prefix)
		copy(q[len(pw.prefix):], pw.buffer.Bytes())
		// one write ensures lines will be written synchronized relative to other writers to pw.writer
		pw.writer.Write(q)
		pw.buffer.Reset()
	}
	return nil
}
