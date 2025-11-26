package libtools

import (
	"fmt"
	"os"
	"sync"
	"syscall"

	"github.com/groundhogchain/chuckd/lib/libruntime"
)

type childKiller struct {
	lk   sync.Mutex
	pgid map[int]bool
}

func (x *childKiller) register(pgid int) {
	x.lk.Lock()
	defer x.lk.Unlock()
	x.pgid[pgid] = true
}

func (x *childKiller) unregister(pgid int) {
	x.lk.Lock()
	defer x.lk.Unlock()
	delete(x.pgid, pgid)
}

func (x *childKiller) kill() {
	x.lk.Lock()
	defer x.lk.Unlock()
	for pgid := range x.pgid {
		fmt.Fprintf(os.Stderr, "killing pgid %d\n", pgid)
		syscall.Kill(pgid, syscall.SIGTERM)
		delete(x.pgid, pgid)
	}
}

var ck = &childKiller{pgid: map[int]bool{}}

func init() {
	// Register with libruntime's signal handler instead of having our own
	// This ensures kill() runs before os.Exit()
	libruntime.RegisterInterruptHook(ck.kill)
}
