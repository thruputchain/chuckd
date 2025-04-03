package broadcast

func (s *BroadcastService) findTree(h MulticastTreeHash) []MulticastTreeNode {
	s.lk.Lock()
	defer s.lk.Unlock()
	return s.trees[h]
}
