package baton

func (b *Baton) lock() {
	b.db.lk.Lock()
	b.catchup.lk.Lock()
	b.consensus.lk.Lock()
	b.nominate.lk.Lock()
}

func (b *Baton) unlock() {
	b.nominate.lk.Unlock()
	b.consensus.lk.Unlock()
	b.catchup.lk.Unlock()
	b.db.lk.Unlock()
}
