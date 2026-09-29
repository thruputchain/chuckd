package discovery

import (
	"bytes"
	"context"
	"crypto/rand"
	"strings"
	"testing"

	"github.com/groundhogchain/chuckd/util/protoutil"
	"github.com/groundhogchain/lib-go/libxdr"
	proto "github.com/groundhogchain/protocol-go"
	"github.com/rs/zerolog"
)

type recordingQSets struct{ added int }

func (r *recordingQSets) SCPSetKnownQuorumSet(context.Context, proto.SCPQuorumSet) proto.SCPOptQuorumSet {
	r.added++
	return nil
}

func (*recordingQSets) SCPRemoveKnownQuorumSet(context.Context, proto.SCPHash) proto.SCPOptQuorumSet {
	return nil
}

func (*recordingQSets) SCPGetKnownQuorumSet(context.Context, proto.SCPHash) proto.SCPOptQuorumSet {
	return nil
}

func TestDiscoveryRejectsUnsupportedPublicKeys(t *testing.T) {
	oldLevel := zerolog.GlobalLevel()
	zerolog.SetGlobalLevel(zerolog.InfoLevel)
	t.Cleanup(func() { zerolog.SetGlobalLevel(oldLevel) })
	identity, err := protoutil.GenerateIdentity(rand.Reader, "test")
	if err != nil {
		t.Fatal(err)
	}
	m := libxdr.NewXdrMarshaler()
	bad := proto.PublicKey{Scheme: proto.CryptographicScheme(123)}
	tests := []struct {
		name   string
		mutate func(*proto.KnownQuorumSets)
	}{
		{"quorum signer", func(k *proto.KnownQuorumSets) { k.Quorum_set_snapshots[0].Signer.Public_key = bad }},
		{"quorum validator", func(k *proto.KnownQuorumSets) {
			k.Quorum_set_snapshots[0].Quorum_set_snapshot.Quorum_set.Validators[0] = bad
		}},
		{"nested quorum validator", func(k *proto.KnownQuorumSets) {
			k.Quorum_set_snapshots[0].Quorum_set_snapshot.Quorum_set.Inner_sets = []proto.SCPQuorumSet{
				{Threshold: 1, Inner_sets: []proto.SCPQuorumSet{{Threshold: 1, Validators: []proto.SCPNodeID{bad}}}},
			}
		}},
		{"validator signer", func(k *proto.KnownQuorumSets) { k.Validator_snapshots[0].Signer.Public_key = bad }},
		{"advertised validator", func(k *proto.KnownQuorumSets) { k.Validator_snapshots[0].Validator_snapshot.Validator.Public_key = bad }},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var logs bytes.Buffer
			ctx := zerolog.New(&logs).WithContext(context.Background())
			controller := &recordingQSets{}
			svc := NewDiscoveryService(ctx, nil, proto.DiscoveryConfig{}, proto.ValidatorConfig{Private_info: *identity}, proto.SCPConfig{})
			svc.scpController = controller
			qset := proto.SCPQuorumSet{Threshold: 1, Validators: []proto.SCPNodeID{identity.Public_info.Public_key}}
			known := proto.KnownQuorumSets{
				Quorum_set_snapshots: []proto.SignedQuorumSetSnapshot{protoutil.MakeSignedQuorumSetSnapshot(m, *identity, qset)},
				Validator_snapshots:  []proto.SignedValidatorInfoSnapshot{protoutil.MakeSignedValidatorInfoSnapshot(m, *identity, identity.Public_info)},
			}
			tc.mutate(&known)
			quorumCase := strings.Contains(tc.name, "quorum")
			if quorumCase {
				known.Validator_snapshots = nil
			} else {
				known.Quorum_set_snapshots = nil
			}
			if _, err := svc.Discovery_update_quorums(ctx, known); err != nil {
				t.Fatal(err)
			}
			if len(svc.knownQSets) != 0 || len(svc.hashQSet) != 0 || len(svc.knownVs) != 0 || controller.added != 0 {
				t.Fatal("unsupported key reached discovery caches or SCP controller")
			}
			if !strings.Contains(logs.String(), `"level":"info"`) || !strings.Contains(logs.String(), "unsupported public key scheme") {
				t.Fatalf("expected Info rejection log, got %s", logs.String())
			}
			// Rejection must not prevent subsequent valid updates.
			svc.updateKnown(ctx, proto.KnownQuorumSets{
				Quorum_set_snapshots: []proto.SignedQuorumSetSnapshot{protoutil.MakeSignedQuorumSetSnapshot(m, *identity, proto.SCPQuorumSet{Threshold: 1, Validators: []proto.SCPNodeID{identity.Public_info.Public_key}})},
				Validator_snapshots:  []proto.SignedValidatorInfoSnapshot{protoutil.MakeSignedValidatorInfoSnapshot(m, *identity, identity.Public_info)},
			})
			if len(svc.knownQSets) != 1 || len(svc.knownVs) != 1 || controller.added != 1 {
				t.Fatal("valid Ed25519 update was rejected")
			}
		})
	}
}
