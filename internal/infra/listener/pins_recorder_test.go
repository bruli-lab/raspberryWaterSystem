package listener_test

import (
	"context"
	"sync"
	"time"

	"github.com/bruli-lab/go-core/cqs"
	"github.com/bruli-lab/go-core/event"
	"github.com/bruli/raspberryWaterSystem/internal/app"
)

type pinsRecorder struct {
	m            sync.Mutex
	pins         []app.ExecutePinsCmd
	waits        []time.Duration
	messages     []string
	logs         int
	cleanStates  []bool
	failingPins  map[string]error
	cleanChanged chan struct{}
}

func newPinsRecorder() *pinsRecorder {
	return &pinsRecorder{cleanChanged: make(chan struct{}, 1)}
}

func (r *pinsRecorder) commandHandler() cqs.CommandHandler {
	return &cqs.CommandHandlerMock{
		HandleFunc: func(_ context.Context, cmd cqs.Command) ([]event.Event, error) {
			r.m.Lock()
			defer r.m.Unlock()
			switch c := cmd.(type) {
			case app.ExecutePinsCmd:
				r.pins = append(r.pins, c)
				return nil, r.failingPins[c.Pins[0]]
			case app.PublishMessageCmd:
				r.messages = append(r.messages, c.Message)
			case app.SaveExecutionLogCmd:
				r.logs++
			case app.ChangeCleanStatusCmd:
				r.cleanStates = append(r.cleanStates, c.Clean)
				r.cleanChanged <- struct{}{}
			}
			return nil, nil
		},
	}
}

func (r *pinsRecorder) wait(_ context.Context, d time.Duration) error {
	r.m.Lock()
	defer r.m.Unlock()
	r.waits = append(r.waits, d)
	return nil
}

func (r *pinsRecorder) recordedPins() []app.ExecutePinsCmd {
	r.m.Lock()
	defer r.m.Unlock()
	return append([]app.ExecutePinsCmd(nil), r.pins...)
}

func indexOfPin(cmds []app.ExecutePinsCmd, pin string) int {
	for i, c := range cmds {
		if c.Pins[0] == pin {
			return i
		}
	}
	return -1
}
