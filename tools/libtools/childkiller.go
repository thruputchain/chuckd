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
	defer x.lk.Unlock()
	for pgid := range x.pgid {
		fmt.Fprintf(os.Stderr, "killing pid %d\n", pgid)
		syscall.Kill(pgid, syscall.SIGTERM)
		delete(x.pgid, pgid)
	}
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
