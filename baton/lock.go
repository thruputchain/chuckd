package baton

func (b *Baton) lock() {
	b.db.lk.Lock()
	b.catchup.lk.Lock()
	b.consensus.lk.Lock()
	b.nominator.lk.Lock()
}

func (b *Baton) unlock() {
	b.nominator.lk.Unlock()
	b.consensus.lk.Unlock()
	b.catchup.lk.Unlock()
	b.db.lk.Unlock()
}
