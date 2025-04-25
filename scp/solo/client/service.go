package client

import (
	"context"

	"github.com/groundhogchain/chuckd/lib/libnet"
	"github.com/groundhogchain/chuckd/scp/api"
	"github.com/groundhogchain/chuckd/util/protoutil"
	"github.com/groundhogchain/chuckd/util/rpcutil"
	proto "github.com/groundhogchain/protocol-go"
)

type SoloClientService struct {
	net     libnet.Network
	self    proto.ValidatorPrivateInfo
	handler api.SCPHandler
}

func NewSoloClientService(
	ctx context.Context,
	n libnet.Network,
	self proto.ValidatorPrivateInfo,
	handler api.SCPHandler,
) *SoloClientService {
	return &SoloClientService{
		net:     n,
		self:    self,
		handler: handler,
	}
}

func (svc *SoloClientService) Start(ctx context.Context, address string) {
	rpcutil.Start(ctx, svc.net, address, proto.SoloClient1_Server{Srv: svc})
}

func (svc *SoloClientService) Soloclient_null(context.Context) error {
	return nil
}

func (svc *SoloClientService) Soloclient_validate(ctx context.Context, signed proto.SoloSignedValidate) (proto.SoloValidationLevel, error) {

	// check signatures
	if !protoutil.VerifyXdr(signed.Signer.Public_key, &signed.Validate, signed.Signature) {
		return proto.SoloInvalidValue, nil
	}

	return svc.handler.ValidateValue(signed.Validate.Slot_index, signed.Validate.Value, true), nil
}

func (svc *SoloClientService) Soloclient_externalized(ctx context.Context, signed proto.SoloSignedExternalized) error {

	// check signatures
	if !protoutil.VerifyXdr(signed.Signer.Public_key, &signed.Externalized, signed.Signature) {
		return nil
	}

	svc.handler.OnExternalizedValue(signed.Externalized.Slot_index, signed.Externalized.Value)
	return nil
}
