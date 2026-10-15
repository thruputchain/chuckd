// Package configtext adapts operator configuration identifiers to base58.
// Internal XDR JSON and secret keys retain their protocol representation.
package configtext

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"

	"github.com/thruputchain/lib-go/libxdr"
	proto "github.com/thruputchain/thruput-protocol-go"
	"github.com/mr-tron/base58"
	"github.com/xdrpp/goxdr/xdr"
)

// Marshal writes the operator representation of a supported configuration.
func Marshal(value xdr.XdrType) ([]byte, error) {
	data, err := libxdr.XdrToJson(value)
	if err != nil {
		return nil, err
	}
	return convert(value, data, true)
}

// MustMarshalString is for script generators with statically known config types.
func MustMarshalString(value xdr.XdrType) string {
	data, err := Marshal(value)
	if err != nil {
		panic(err)
	}
	return string(data)
}

// Unmarshal reads base58 identifiers. Old hex configuration must be regenerated.
func Unmarshal(data []byte, value xdr.XdrType) error {
	internal, err := convert(value, data, false)
	if err != nil {
		return err
	}
	return libxdr.JsonToXdr(value, internal)
}

func convert(value xdr.XdrType, data []byte, output bool) ([]byte, error) {
	var object map[string]any
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.UseNumber()
	if err := decoder.Decode(&object); err != nil {
		return nil, fmt.Errorf("invalid configuration JSON: %w", err)
	}
	if object == nil {
		return nil, fmt.Errorf("configuration must be an object")
	}
	if err := decoder.Decode(new(any)); err != io.EOF {
		return nil, fmt.Errorf("unexpected data after configuration")
	}
	c := converter{output: output}
	var err error
	switch value.(type) {
	case *proto.Config:
		err = c.field(object, "validator_config", c.validator)
		if err == nil {
			err = c.field(object, "scp_config", func(v map[string]any) error { return c.field(v, "quorum_set", c.quorum) })
		}
	case *proto.ValidatorConfig:
		err = c.validator(object)
	case *proto.SCPQuorumSet:
		err = c.quorum(object)
	case *proto.DogwoodStandaloneConfig:
		err = c.field(object, "validator_config", c.validator)
	default:
		return nil, fmt.Errorf("unsupported operator configuration type %T", value)
	}
	if err != nil {
		return nil, err
	}
	return json.MarshalIndent(object, "", "  ")
}

type converter struct{ output bool }

func (c converter) field(object map[string]any, name string, convert func(map[string]any) error) error {
	value := object[name]
	if value == nil {
		return nil
	}
	child, ok := value.(map[string]any)
	if !ok {
		return fmt.Errorf("%s must be an object", name)
	}
	if err := convert(child); err != nil {
		return fmt.Errorf("%s: %w", name, err)
	}
	return nil
}

func (c converter) list(object map[string]any, name string, convert func(map[string]any) error) error {
	value := object[name]
	if value == nil {
		return nil
	}
	items, ok := value.([]any)
	if !ok {
		return fmt.Errorf("%s must be an array", name)
	}
	for i, item := range items {
		child, ok := item.(map[string]any)
		if !ok {
			return fmt.Errorf("%s[%d] must be an object", name, i)
		}
		if err := convert(child); err != nil {
			return fmt.Errorf("%s[%d]: %w", name, i, err)
		}
	}
	return nil
}

func (c converter) publicInfo(v map[string]any) error {
	return c.field(v, "public_key", c.publicKey)
}

func (c converter) validator(v map[string]any) error {
	if err := c.field(v, "private_info", func(private map[string]any) error {
		return c.field(private, "public_info", c.publicInfo)
	}); err != nil {
		return err
	}
	return c.list(v, "known_validators", c.publicInfo)
}

func (c converter) quorum(v map[string]any) error {
	if err := c.list(v, "validators", c.publicKey); err != nil {
		return err
	}
	return c.list(v, "inner_sets", c.quorum)
}

func (c converter) publicKey(v map[string]any) error {
	// OpenTofu uses an explicit scheme field; libxdr emits tagged union arms.
	// Both shapes retain their existing structure, with base58 public key text.
	if scheme, ok := v["scheme"].(string); ok {
		if len(v) != 2 {
			return fmt.Errorf("expected scheme and public key fields")
		}
		return c.publicKeyBody(scheme, v)
	}
	if len(v) != 1 {
		return fmt.Errorf("expected one public key scheme")
	}
	for scheme, body := range v {
		arm, ok := body.(map[string]any)
		if !ok || len(arm) != 1 {
			return fmt.Errorf("expected a public key body")
		}
		return c.publicKeyBody(scheme, arm)
	}
	return nil
}

func (c converter) publicKeyBody(scheme string, arm map[string]any) error {
	field, width := "", 0
	switch scheme {
	case "Ed25519":
		field, width = "ed25519", 32
	case "MlDsa65":
		field, width = "mldsa65", 1952
	default:
		return fmt.Errorf("unsupported public key scheme")
	}
	text, ok := arm[field].(string)
	if !ok {
		return fmt.Errorf("public key must be a string")
	}
	var decoded []byte
	var err error
	if c.output {
		decoded, err = hex.DecodeString(text)
	} else {
		if len(text) < width || len(text) > (width*138/100)+1 {
			return fmt.Errorf("expected a %d-byte base58 public key; regenerate old configuration", width)
		}
		decoded, err = base58.Decode(text)
	}
	if err != nil || len(decoded) != width {
		return fmt.Errorf("invalid %d-byte public key", width)
	}
	if c.output {
		arm[field] = base58.Encode(decoded)
	} else {
		arm[field] = hex.EncodeToString(decoded)
	}
	return nil
}
