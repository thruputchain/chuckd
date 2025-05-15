package follower

import (
	"context"

	"github.com/groundhogchain/chuckd/lib/libnet"
	"github.com/groundhogchain/chuckd/lib/log"
	"github.com/groundhogchain/chuckd/scp/api"
	"github.com/groundhogchain/chuckd/scp/solo/dial"
	"github.com/groundhogchain/chuckd/util/protoutil"
	"github.com/groundhogchain/chuckd/util/rpcutil"
	"github.com/groundhogchain/chuckd/util/xdrutil"
	proto "github.com/groundhogchain/protocol-go"
	"github.com/xdrpp/goxdr/xdr"
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

func (svc *SoloFollowerService) Nominate(ctx context.Context, slot int64, value xdr.XdrType, prev xdr.XdrType) bool {

	c, err := dial.DialSoloLeaderClient(ctx, svc.net, svc.leader.Address)
	if err != nil {
		log.Errorf("solo leader not reachable for nomination: %s", err)
		return false
	}

	nomination := proto.SoloNomination{
		Slot_index: slot,
		Value:      xdrutil.MarshalXdr(value),
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
