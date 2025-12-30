package batonkit

type BatonLevers struct {
	PauseNominating bool
}

type BatonHooks interface {
	CommitHook(cb *CommittableBlock) BatonLevers
}

// util

func JustCommitHook(f CommitHookFunc) BatonHooks {
	return funcHooks{commit: f}
}

type CommitHookFunc func(*CommittableBlock) BatonLevers

type funcHooks struct {
	commit CommitHookFunc
}

func (x funcHooks) CommitHook(cb *CommittableBlock) BatonLevers {
	return x.commit(cb)
}
