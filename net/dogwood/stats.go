package dogwood

import "github.com/groundhogchain/chuckd/lib/form"

type Stats struct {
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
		NumSendStreams:                 x.streamSender.NumStreams(),
		NumReceiveStreams:              x.streamReceiver.NumStreams(),
		NumSendTransmissionsSending:    x.xSender.NumSending(),
		NumSendTransmissionsWaiting:    x.xSender.NumWaiting(),
		NumReceiveTransmissionsPartial: x.xReceiver.NumPartial(),
	}
}
