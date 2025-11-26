package libtools

import (
	"fmt"
	"os"
	"os/signal"
	"sync"
	"syscall"
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
	pgids := make([]int, 0, len(x.pgid))
	for pgid := range x.pgid {
		pgids = append(pgids, pgid)
	}
	x.lk.Unlock()
	
	// First, send SIGTERM to all process groups
	for _, pgid := range pgids {
		if pgid < 0 {
			// Negative PID means process group ID
			fmt.Fprintf(os.Stderr, "killing process group %d\n", -pgid)
		} else {
			fmt.Fprintf(os.Stderr, "killing pid %d\n", pgid)
		}
		syscall.Kill(pgid, syscall.SIGTERM)
	}
	
	// Give processes a moment to clean up, then send SIGKILL if needed
	// Note: We don't wait or check if they're still alive - the remote processes
	// should receive SIGHUP when SSH disconnects and clean themselves up
	x.lk.Lock()
	for pgid := range x.pgid {
		delete(x.pgid, pgid)
	}
	x.lk.Unlock()
}

var ck = &childKiller{pgid: map[int]bool{}}

func init() {
	sigc := make(chan os.Signal, 1)
	signal.Notify(sigc, syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP)
	go func() {
		s := <-sigc
		ck.kill()
		os.Exit(128 + int(s.(syscall.Signal))) // propagate a conventional code
	}()
}
