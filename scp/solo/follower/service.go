package follower

import (
	"context"

	"github.com/groundhogchain/chuckd/lib/liblog"
	"github.com/groundhogchain/chuckd/lib/libnet"
	"github.com/groundhogchain/chuckd/scp/scpapi"
	"github.com/groundhogchain/chuckd/scp/solo/dial"
	"github.com/groundhogchain/chuckd/util/protoutil"
	"github.com/groundhogchain/chuckd/util/rpcutil"
	"github.com/groundhogchain/chuckd/util/xdrutil"
	proto "github.com/groundhogchain/protocol-go"
)

type SoloFollowerService struct {
	net     libnet.Network
	leader  proto.ValidatorPublicInfo
	self    proto.ValidatorPrivateInfo
	handler scpapi.SCPHandler
	m       xdrutil.XdrMarshaler
}

func NewSoloFollowerService(
	ctx context.Context,
	n libnet.Network,
	leader proto.ValidatorPublicInfo,
	self proto.ValidatorPrivateInfo,
	handler scpapi.SCPHandler,
) *SoloFollowerService {
	return &SoloFollowerService{
		net:     n,
		leader:  leader,
		self:    self,
		handler: handler,
		m:       xdrutil.NewXdrMarshaler(),
	}
}

func (svc *SoloFollowerService) Start(ctx context.Context, address string) {
	rpcutil.StartSerial(ctx, svc.net, address, proto.SoloFollower1_Server{Srv: svc})
}

func (svc *SoloFollowerService) Solofollower_null(context.Context) error {
	return nil
}

func (svc *SoloFollowerService) Solofollower_validate(ctx context.Context, signed proto.SoloSignedValidate) (proto.SoloValidationLevel, error) {

	// log.Infof("solo follower received validation request %v", form.SprintJSON(signed.Validate))

	// check signatures
	if !protoutil.VerifyXdr(svc.m, signed.Signer.Public_key, &signed.Validate, signed.Signature) {
		return proto.SoloInvalidValue, nil
	}

	return scpapi.ToSoloValidationLevel(
		svc.handler.ValidateNominationValue(ctx, proto.SCPSlotIndex(signed.Validate.Slot_index), signed.Validate.Value)), nil
}

func (svc *SoloFollowerService) Solofollower_externalized(ctx context.Context, signed proto.SoloSignedExternalized) error {

	// check signatures
	if !protoutil.VerifyXdr(svc.m, signed.Signer.Public_key, &signed.Externalized, signed.Signature) {
		return nil
	}

	svc.handler.ExternalizedValue(ctx, proto.SCPSlotIndex(signed.Externalized.Slot_index), signed.Externalized.Value)
	return nil
}

// SCPController interface

func (svc *SoloFollowerService) Nominate(ctx context.Context, slot scpapi.Slot, value scpapi.Value, prev scpapi.Value) bool {

	c, err := dial.DialSoloLeaderClient(ctx, svc.net, svc.leader.Address)
	if err != nil {
		liblog.Errorf("solo leader not reachable for nomination: %s", err)
		return false
	}
	defer c.Close()

	nomination := proto.SoloNomination{
		Slot_index: proto.SoloSlotIndex(slot),
		Value:      value,
	}
	signed := proto.SoloSignedNomination{
		Signer:     svc.self.Public_info,
		Signature:  protoutil.SignXdr(svc.m, svc.self.Secret_key, &nomination),
		Nomination: nomination,
	}

	err = c.Sololeader_nominate(ctx, signed)
	if err != nil {
		liblog.Errorf("solo nomination error: %s", err)
		return false
	}

	return true
}

func (svc *SoloFollowerService) UpdateQuorumSet(proto.SCPQuorumSet) {
	panic("not used")
}

func (svc *SoloFollowerService) GetQuorumSet() proto.SCPQuorumSet {
	panic("not used")
}

func (svc *SoloFollowerService) NodeID() scpapi.NodeID {
	panic("not used")
}

func (svc *SoloFollowerService) PurgeSlots(maxSlotIndex scpapi.Slot, slotToKeep scpapi.Slot) {
	panic("not used")
}

func (svc *SoloFollowerService) IsValidator() bool {
	panic("not used")
}

func (svc *SoloFollowerService) IsSlotFullyValidated() bool {
	panic("not used")
}
