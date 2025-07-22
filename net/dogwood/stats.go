package dogwood

import (
	"github.com/groundhogchain/chuckd/lib/form"
	"github.com/groundhogchain/chuckd/runtime"
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
	NumReceiveStreams              int
	NumReceiveTransmissionsPartial int
}

func (x Stats) String() string {
	return form.Display(x)
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
		NumReceiveStreams:              x.streamReceiver.NumStreams(),
		NumSendTransmissionsSending:    x.xSender.NumSending(),
		NumSendTransmissionsWaiting:    x.xSender.NumWaiting(),
		NumReceiveTransmissionsPartial: x.xReceiver.NumPartial(),
	}
}
