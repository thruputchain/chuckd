package xdrutil

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"io"
	"reflect"
	"strings"

	"github.com/xdrpp/goxdr/xdr"
)

type XdrPrint struct {
	Out io.Writer
}

// Generic function that converts any compiled XDR type to a
// multi-line string.  Useful for debugging.
func XdrToString(t xdr.XdrType) string {
	out := &strings.Builder{}
	t.XdrMarshal(&XdrPrint{out}, "")
	return out.String()
}

func (xp XdrPrint) Sprintf(f string, args ...interface{}) string {
	return fmt.Sprintf(f, args...)
}

func (xp XdrPrint) printField(name string, f string, args ...interface{}) {
	if name != "" {
		fmt.Fprintf(xp.Out, "%s: ", name)
	}
	fmt.Fprintf(xp.Out, f, args...)
	fmt.Fprintf(xp.Out, "\n")
}

func (xp XdrPrint) Marshal(name string, i xdr.XdrType) {
	switch v := i.(type) {
	case fmt.Stringer:
		fmt.Fprintf(xp.Out, "%s: %s\n", name, v.String())
	case xdr.XdrPtr:
		fmt.Fprintf(xp.Out, "%s.present: %v\n", name, v.GetPresent())
		v.XdrMarshalValue(xp, fmt.Sprintf("(*%s)", name))
	case xdr.XdrVec:
		fmt.Fprintf(xp.Out, "%s.len: %d\n", name, v.GetVecLen())
		v.XdrMarshalN(xp, name, v.GetVecLen())
	case xdr.XdrAggregate:
		v.XdrRecurse(xp, name)
	case *xdrGo:
		xp.marshalGo(name, v.Value)
	default:
		fmt.Fprintf(xp.Out, "%s: %v\n", name, i)
	}
}

// XXX: check if pointer version of val is convertible to xdrtype (do same in json)

func (xp XdrPrint) marshalGo(name string, val any) {
	switch u := (val).(type) {
	case bool:
		x := xdr.XdrBool(u)
		xp.Marshal(name, &x)
	case int32:
		x := xdr.XdrInt32(u)
		xp.Marshal(name, &x)
	case uint32:
		x := xdr.XdrUint32(u)
		xp.Marshal(name, &x)
	case int64:
		x := xdr.XdrInt64(u)
		xp.Marshal(name, &x)
	case uint64:
		x := xdr.XdrUint64(u)
		xp.Marshal(name, &x)
	case float32:
		x := xdr.XdrFloat32(u)
		xp.Marshal(name, &x)
	case float64:
		x := xdr.XdrFloat64(u)
		xp.Marshal(name, &x)
	case string:
		x := xdr.XdrString{Str: &u, Bound: uint32(len(u))}
		xp.Marshal(name, x)
	case []byte:
		x := xdr.XdrVecOpaque{Bytes: &u, Bound: uint32(len(u))}
		xp.Marshal(name, &x)
	default:
		vv := reflect.ValueOf(val)
		vv = makeAddressable(vv)
		if vv.Addr().CanConvert(typeOfXdrType) {
			x := vv.Addr().Convert(typeOfXdrType)
			xp.Marshal(name, x.Interface().(xdr.XdrType))
			return
		}
		switch {
		case vv.Kind() == reflect.Array || vv.Kind() == reflect.Slice:
			switch {
			case vv.Type().Elem() == byteType: // []byte, [n]byte
				if name != "" {
					fmt.Fprintf(xp.Out, "%s: ", name)
				}
				xp.Out.Write([]byte("\""))
				enc := base64.NewEncoder(base64.StdEncoding, xp.Out)
				aa := reflect.New(vv.Type()).Elem()
				aa.Set(vv)
				enc.Write(aa.Slice(0, aa.Len()).Bytes())
				enc.Close()
				xp.Out.Write([]byte("\"\n"))
			default: // []T, [n]T
				xp.printField(name, "")
				xp.aggregateGo(name, vv)
			}
		case vv.Kind() == reflect.Struct:
			xp.printField(name, "")
			xp.aggregateGo(name, vv)
		case vv.Kind() == reflect.Interface || vv.Kind() == reflect.Pointer:
			if vv.IsNil() {
				xp.printField(name, "nil")
			} else {
				xp.marshalGo(name, vv.Elem().Interface())
			}
		default: // complex64, etc
			xp.printField(name, "%v", val)
		}
	}
}

func makeAddressable(v reflect.Value) reflect.Value {
	if v.CanAddr() {
		return v
	}
	newValue := reflect.New(v.Type())
	newValue.Elem().Set(v)
	return newValue.Elem()
}

var typeOfXdrType = reflect.TypeOf((*xdr.XdrType)(nil)).Elem()

func (xp XdrPrint) aggregateGo(name string, val reflect.Value) {
	switch {
	case val.Type().Kind() == reflect.Slice:
		for i := 0; i < val.Len(); i++ {
			vv := val.Index(i).Interface()
			xp.marshalGo(fmt.Sprintf("%s[%d]", name, i), vv)
		}
	case val.Type().Kind() == reflect.Array:
		for i := 0; i < val.Len(); i++ {
			vv := val.Index(i).Interface()
			xp.marshalGo(fmt.Sprintf("%s[%d]", name, i), vv)
		}
	case val.Type().Kind() == reflect.Struct:
		for i := 0; i < val.NumField(); i++ {
			prefix := ""
			if name != "" {
				prefix = xp.Sprintf("%s.", name)
			}
			if val.Type().Field(i).IsExported() {
				vv := val.Field(i)
				if vv.Addr().CanConvert(typeOfXdrType) {
					x := vv.Addr().Convert(typeOfXdrType)
					xp.Marshal(xp.Sprintf("%s%s", prefix, val.Type().Field(i).Name), x.Interface().(xdr.XdrType))
				} else {
					xp.marshalGo(xp.Sprintf("%s%s", prefix, val.Type().Field(i).Name), vv.Interface())
				}
			}
		}
	}
}

// Format an XDR structure as text.
func XdrToText(src xdr.XdrType) (text string, err error) {
	defer func() {
		if i := recover(); i != nil {
			text = ""
			err = i.(error)
		}
	}()
	b := &bytes.Buffer{}
	t := &XdrPrint{Out: b}
	t.Marshal("", src)
	return b.String(), nil
}

func AnyToText(src any) (text string, err error) {
	switch v := src.(type) {
	case xdr.XdrType:
		return XdrToText(v)
	default:
		return XdrToText(&xdrGo{Value: src})
	}
}
