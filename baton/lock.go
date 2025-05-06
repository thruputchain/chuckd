package baton

func (b *Baton) lock() {
	b.db.lk.Lock()
	b.catchup.lk.Lock()
	b.scp.lk.Lock()
	b.prop.lk.Lock()
}

func (b *Baton) unlock() {
	b.prop.lk.Unlock()
	b.scp.lk.Unlock()
	b.catchup.lk.Unlock()
	b.db.lk.Unlock()
}
