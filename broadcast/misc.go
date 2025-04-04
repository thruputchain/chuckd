package broadcast

func (svc *BroadcastService) generateChannelID() ChannelID {
	svc.randLk.Lock()
	defer svc.randLk.Lock()
	var chID ChannelID
	svc.rand.Read(chID[:])
	return chID
}

func (svc *BroadcastService) buildMulticastTree(chID ChannelID) (MulticastTreeNodes, []ValidatorPublicInfo, MulticastTreeHash) {
	return svc.router.BuildTree(svc.ValidatorID(), chID)
}
