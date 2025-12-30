package batonkit

type BatonLevers struct {
	PauseNominating bool
}

type BatonHooks interface {
	CommitHook(cb *CommittableBlock) BatonLevers
}
