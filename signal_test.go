package dxui

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/dxui-org/dxui/internal/platform"
)

type fakeSignalNotifier struct {
	channel chan<- os.Signal
	signals []os.Signal
	stops   int
}

func (notifier *fakeSignalNotifier) Notify(channel chan<- os.Signal, signals ...os.Signal) {
	notifier.channel = channel
	notifier.signals = append(notifier.signals, signals...)
}

func (notifier *fakeSignalNotifier) Stop(channel chan<- os.Signal) {
	if channel != notifier.channel {
		panic("stopped an unregistered signal channel")
	}
	notifier.stops++
}

type interruptTestEvents struct {
	wake chan struct{}
}

func (*interruptTestEvents) Wait(context.Context, *time.Time) (platform.Event, error) {
	return platform.Event{}, errors.New("unused")
}

func (*interruptTestEvents) Poll() (platform.Event, bool, error) {
	return platform.Event{}, false, nil
}

func (events *interruptTestEvents) Wake() error {
	events.wake <- struct{}{}
	return nil
}

func TestInterruptClosesRunningAppAndStopsListener(t *testing.T) {
	events := &interruptTestEvents{wake: make(chan struct{}, 1)}
	app := NewApp(AppOptions{})
	app.running = true
	app.events = events
	notifier := &fakeSignalNotifier{}
	handler := startInterruptHandler(app, notifier)

	if len(notifier.signals) != 1 || notifier.signals[0] != os.Interrupt {
		t.Fatalf("registered signals = %v, want os.Interrupt", notifier.signals)
	}
	notifier.channel <- os.Interrupt
	<-handler.done
	if !app.isClosed() {
		t.Fatal("interrupt did not close app")
	}
	select {
	case <-events.wake:
	default:
		t.Fatal("interrupt did not wake event loop")
	}
	handler.Stop()
	if notifier.stops != 1 {
		t.Fatalf("stop calls = %d, want 1", notifier.stops)
	}
}

func TestStoppingInterruptListenerDoesNotCloseApp(t *testing.T) {
	app := NewApp(AppOptions{})
	app.running = true
	notifier := &fakeSignalNotifier{}
	handler := startInterruptHandler(app, notifier)

	handler.Stop()
	handler.Stop()
	if app.isClosed() {
		t.Fatal("stopping listener closed app")
	}
	if notifier.stops != 1 {
		t.Fatalf("stop calls = %d, want 1", notifier.stops)
	}
}
