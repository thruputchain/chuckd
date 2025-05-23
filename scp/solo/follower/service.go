package follower

import (
	"context"

	"github.com/groundhogchain/chuckd/lib/libnet"
	"github.com/groundhogchain/chuckd/lib/log"
	"github.com/groundhogchain/chuckd/scp/api"
	"github.com/groundhogchain/chuckd/scp/solo/dial"
	"github.com/groundhogchain/chuckd/util/protoutil"
	"github.com/groundhogchain/chuckd/util/rpcutil"
	proto "github.com/groundhogchain/protocol-go"
)

type SoloFollowerService struct {
	net     libnet.Network
	leader  proto.ValidatorPublicInfo
	self    proto.ValidatorPrivateInfo
	handler api.SCPHandler
}

func NewSoloFollowerService(
	ctx context.Context,
	n libnet.Network,
	leader proto.ValidatorPublicInfo,
	self proto.ValidatorPrivateInfo,
	handler api.SCPHandler,
) *SoloFollowerService {
	return &SoloFollowerService{
		net:     n,
		leader:  leader,
		self:    self,
		handler: handler,
	}
}

func (svc *SoloFollowerService) Start(ctx context.Context, address string) {
	rpcutil.Start(ctx, svc.net, address, proto.SoloFollower1_Server{Srv: svc})
}

func (svc *SoloFollowerService) Solofollower_null(context.Context) error {
	return nil
}

func (svc *SoloFollowerService) Solofollower_validate(ctx context.Context, signed proto.SoloSignedValidate) (proto.SoloValidationLevel, error) {

	// log.Infof("solo follower received validation request %v", form.SprintJSON(signed.Validate))

	// check signatures
	if !protoutil.VerifyXdr(signed.Signer.Public_key, &signed.Validate, signed.Signature) {
		return proto.SoloInvalidValue, nil
	}

	return svc.handler.ValidateValue(ctx, signed.Validate.Slot_index, signed.Validate.Value, true), nil
}

func (svc *SoloFollowerService) Solofollower_externalized(ctx context.Context, signed proto.SoloSignedExternalized) error {

	// check signatures
	if !protoutil.VerifyXdr(signed.Signer.Public_key, &signed.Externalized, signed.Signature) {
		return nil
	}

	svc.handler.ExternalizedValue(ctx, signed.Externalized.Slot_index, signed.Externalized.Value)
	return nil
}

// SCPController interface

func (svc *SoloFollowerService) Nominate(ctx context.Context, slot api.Slot, value api.Value, prev api.Value) bool {

	c, err := dial.DialSoloLeaderClient(ctx, svc.net, svc.leader.Address)
	if err != nil {
		log.Errorf("solo leader not reachable for nomination: %s", err)
		return false
	}

	nomination := proto.SoloNomination{
		Slot_index: slot,
		Value:      value,
	}
	signed := proto.SoloSignedNomination{
		Signer:     svc.self.Public_info,
		Signature:  protoutil.SignXdr(svc.self.Secret_key, &nomination),
		Nomination: nomination,
	}

	err = c.Sololeader_nominate(ctx, signed)
	if err != nil {
		log.Errorf("solo nomination error: %s", err)
		return false
	}

	return true
}

func (svc *SoloFollowerService) UpdateQuorumSet(proto.QuorumSet) {
	panic("not used")
}

func (svc *SoloFollowerService) GetQuorumSet() proto.QuorumSet {
	panic("not used")
}

func (svc *SoloFollowerService) NodeID() api.NodeID {
	panic("not used")
}

func (svc *SoloFollowerService) PurgeSlots(maxSlotIndex api.Slot, slotToKeep api.Slot) {
	panic("not used")
}

func (svc *SoloFollowerService) IsValidator() bool {
	panic("not used")
}

func (svc *SoloFollowerService) IsSlotFullyValidated() bool {
	panic("not used")
}
