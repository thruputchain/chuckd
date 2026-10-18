package dogwood

import (
	"github.com/thruputchain/chuckd/runtime"
	"github.com/thruputchain/lib-go/libtext"
)

type Stats struct {
	ResourceUsage string
	//
	CallerStats
	StreamReceiverStats
	StreamSenderStats
	TransmissionReceiverStats
	TransmissionSenderStats
	NavigatorStats
	//
	NumSendStreams              int
	NumSendTransmissionsSending int
	NumSendTransmissionsWaiting int
	//
	NumReceiveTransmissionsPartial int
}

func (x Stats) String() string {
	return libtext.Display(x)
}

func (x *DogwoodService) Stats() Stats {
	return Stats{
		ResourceUsage: runtime.ResourceUsage(),
		//
		CallerStats:               x.caller.Stats(),
		StreamReceiverStats:       x.streamReceiver.Stats(),
		StreamSenderStats:         x.streamSender.Stats(),
		TransmissionReceiverStats: x.xReceiver.Stats(),
		TransmissionSenderStats:   x.xSender.Stats(),
		NavigatorStats:            x.nav.Stats(),
		//
		NumSendStreams:                 x.streamSender.NumStreams(),
		NumReceiveTransmissionsPartial: x.xReceiver.NumPartial(),
	}
}
