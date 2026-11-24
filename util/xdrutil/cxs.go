package xdrutil

import "encoding/binary"

// appendCxsUint32 appends n as a cxs unsigned int, in its shortest form: n
// itself up to 62, otherwise a header byte (63, 127 or 191 for one, two or
// four more bytes) and n in that many little-endian bytes.
func appendCxsUint32(b []byte, n uint32) []byte {
	switch {
	case n <= 62:
		return append(b, byte(n))
	case n <= 0xff:
		return append(b, 63, byte(n))
	case n <= 0xffff:
		return binary.LittleEndian.AppendUint16(append(b, 127), uint16(n))
	default:
		return binary.LittleEndian.AppendUint32(append(b, 191), n)
	}
}

// appendCxsUint32s appends ns as a cxs array of unsigned ints: its length,
// then each element.
func appendCxsUint32s(b []byte, ns []uint32) []byte {
	b = appendCxsUint32(b, uint32(len(ns)))
	for _, n := range ns {
		b = appendCxsUint32(b, n)
	}
	return b
}
