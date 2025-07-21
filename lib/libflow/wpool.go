package libflow

type WorkerPool interface {
	Submit(func())
	Stop()
}
