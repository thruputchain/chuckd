package baton

type EventNominating struct {
	Nominating any
	Tip        Tip
	NumTx      int
}
