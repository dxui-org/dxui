package dxui

import (
	"os"
	"os/signal"
	"sync"
)

type signalNotifier interface {
	Notify(chan<- os.Signal, ...os.Signal)
	Stop(chan<- os.Signal)
}

type processSignalNotifier struct{}

func (processSignalNotifier) Notify(channel chan<- os.Signal, signals ...os.Signal) {
	signal.Notify(channel, signals...)
}

func (processSignalNotifier) Stop(channel chan<- os.Signal) {
	signal.Stop(channel)
}

type interruptHandler struct {
	notifier signalNotifier
	signals  chan os.Signal
	stop     chan struct{}
	done     chan struct{}
	once     sync.Once
}

func startInterruptHandler(app *App, notifier signalNotifier) *interruptHandler {
	handler := &interruptHandler{
		notifier: notifier,
		signals:  make(chan os.Signal, 1),
		stop:     make(chan struct{}),
		done:     make(chan struct{}),
	}
	notifier.Notify(handler.signals, os.Interrupt)
	go func() {
		defer close(handler.done)
		select {
		case <-handler.stop:
			return
		case <-handler.signals:
		}
		select {
		case <-handler.stop:
			return
		default:
			app.Close()
		}
	}()
	return handler
}

func (handler *interruptHandler) Stop() {
	handler.once.Do(func() {
		handler.notifier.Stop(handler.signals)
		close(handler.stop)
	})
	<-handler.done
}
