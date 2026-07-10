package coordinator

import (
	"context"
	"testing"
	"time"

	"github.com/groundhogchain/chuckd/sub/baton/batonkit"
)

func TestNotifyReceiveStopReleasesBlockedNotifications(t *testing.T) {
	const notificationCount = 32

	notify := NewNotifyReceive()
	started := make(chan struct{}, notificationCount)
	done := make(chan struct{}, notificationCount)
	for range notificationCount {
		go func() {
			started <- struct{}{}
			notify.Committed(context.Background(), nil)
			done <- struct{}{}
		}()
	}
	for range notificationCount {
		<-started
	}

	select {
	case <-done:
		t.Fatal("notification returned before it was stopped")
	default:
	}

	notify.Stop()
	deadline := time.After(time.Second)
	for range notificationCount {
		select {
		case <-done:
		case <-deadline:
			t.Fatal("blocked notifications were not released by Stop")
		}
	}
}

func TestNotifyReceiveNotificationsReturnAfterStop(t *testing.T) {
	notify := NewNotifyReceive()
	notify.Stop()
	notify.Stop()

	done := make(chan struct{})
	go func() {
		notify.Committed(context.Background(), nil)
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("notification blocked after Stop")
	}
}

func TestNotifyReceiveStopReleasesBlockedReceiver(t *testing.T) {
	notify := NewNotifyReceive()
	result := make(chan *batonkit.CommittableBlock)
	go func() {
		result <- notify.ReceiveCommitted(context.Background())
	}()

	notify.Stop()
	select {
	case cb := <-result:
		if cb != nil {
			t.Fatal("stopped receiver returned a commit")
		}
	case <-time.After(time.Second):
		t.Fatal("blocked receiver was not released by Stop")
	}
}
