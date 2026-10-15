package configtext_test

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/thruputchain/chuckd/util/configtext"
	"github.com/thruputchain/chuckd/util/protoutil"
	"github.com/thruputchain/lib-go/libxdr"
	proto "github.com/thruputchain/thruput-protocol-go"
)

func TestValidatorConfigPreservesIdentityAndSecret(t *testing.T) {
	identity, err := protoutil.GenerateIdentity(bytes.NewReader(make([]byte, 32)), "localhost:9111")
	if err != nil {
		t.Fatal(err)
	}
	value := proto.ValidatorConfig{Private_info: *identity, Known_validators: []proto.ValidatorPublicInfo{identity.Public_info}}
	data, err := configtext.Marshal(&value)
	if err != nil {
		t.Fatal(err)
	}
	var object map[string]any
	if err := json.Unmarshal(data, &object); err != nil {
		t.Fatal(err)
	}
	private := object["private_info"].(map[string]any)
	public := private["public_info"].(map[string]any)
	key := public["public_key"].(map[string]any)["Ed25519"].(map[string]any)["ed25519"]
	if key != "4zvwRjXUKGfvwnParsHAS3HuSVzV5cA4McphgmoCtajS" {
		t.Fatalf("unexpected zero-seed public key: %v", key)
	}
	var decoded proto.ValidatorConfig
	if err := configtext.Unmarshal(data, &decoded); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(value, decoded) {
		t.Fatal("configuration changed identity or secret bytes")
	}
	raw, err := libxdr.XdrToJson(&value)
	if err != nil {
		t.Fatal(err)
	}
	if err := configtext.Unmarshal(raw, &decoded); err == nil {
		t.Fatal("legacy hex must require regeneration")
	}
}

func TestNestedQuorumAndCoordinatorConfig(t *testing.T) {
	key := proto.PublicKey{Scheme: proto.Ed25519}
	key.Ed25519()[31] = 1
	quorum := proto.SCPQuorumSet{Threshold: 1, Validators: []proto.SCPNodeID{key}, Inner_sets: []proto.SCPQuorumSet{{Threshold: 1, Validators: []proto.SCPNodeID{key}}}}
	value := proto.Config{Scp_config: &proto.SCPConfig{Quorum_set: quorum, Max_timeout_ms: 9007199254740993}}
	data, err := configtext.Marshal(&value)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Count(string(data), strings.Repeat("1", 31)+"2") != 2 {
		t.Fatal("nested quorum keys must use base58")
	}
	var decoded proto.Config
	if err := configtext.Unmarshal(data, &decoded); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(value, decoded) {
		t.Fatal("configuration changed during round trip")
	}
}

func TestOpsPublicKeyShape(t *testing.T) {
	data := []byte(`{"validator_config":{"private_info":{"public_info":{"public_key":{"scheme":"Ed25519","ed25519":"11111111111111111111111111111112"},"address":"hog-coordinator:50000"}}},"scp_config":{"quorum_set":{"threshold":1,"validators":[{"scheme":"Ed25519","ed25519":"11111111111111111111111111111112"}],"inner_sets":[]}}}`)
	var value proto.Config
	if err := configtext.Unmarshal(data, &value); err != nil {
		t.Fatal(err)
	}
	want := proto.Ed25519PublicKey{31: 1}
	if *value.Validator_config.Private_info.Public_info.Public_key.Ed25519() != want || *value.Scp_config.Quorum_set.Validators[0].Ed25519() != want {
		t.Fatal("ops public key bytes changed")
	}
	for _, invalid := range []string{strings.Repeat("ab", 32), "0x" + strings.Repeat("ab", 32), strings.Repeat("1", 33)} {
		bad := bytes.ReplaceAll(data, []byte("11111111111111111111111111111112"), []byte(invalid))
		var value proto.Config
		if err := configtext.Unmarshal(bad, &value); err == nil {
			t.Fatal("invalid ops public key accepted")
		}
	}
}

func TestScheduledConfigurations(t *testing.T) {
	paths, err := filepath.Glob("../../scripts/schedules/host_jobs/*/*/*.sh")
	if err != nil || len(paths) == 0 {
		t.Fatalf("finding scheduled configurations: %v", err)
	}
	for _, path := range paths {
		t.Run(path, func(t *testing.T) {
			script, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			start := bytes.Index(script, []byte("\n{"))
			end := bytes.Index(script, []byte("\nEOF"))
			if start < 0 || end < start {
				t.Fatal("configuration heredoc missing")
			}
			var config proto.Config
			if err := configtext.Unmarshal(script[start:end], &config); err != nil {
				t.Fatal(err)
			}
			if config.Validator_config == nil {
				t.Fatal("validator identity missing")
			}
			identity := config.Validator_config.Private_info
			if !bytes.Equal(identity.Secret_key.Ed25519()[32:], identity.Public_info.Public_key.Ed25519()[:]) {
				t.Fatal("public key differs from secret keypair")
			}
		})
	}
}
