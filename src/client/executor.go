package client

import (
	"context"
)

type Executor struct {
	requestLoop *Dispatcher[QAPIResult]
	instances   map[string]map[string]struct{}
	instanceCh  chan func()
}

func NewExecutor() *Executor {
	executor := &Executor{
		requestLoop: NewDispatcher[QAPIResult](0),
		instances:   make(map[string]map[string]struct{}),
		instanceCh:  make(chan func()),
	}

	go func() {
		for fn := range executor.instanceCh {
			fn()
		}
	}()

	return executor
}

func (e *Executor) Enqueue(instance string, requestId string) <-chan QAPIResult {
	ch := e.requestLoop.Enqueue(requestId)
	e.instanceCh <- func() {
		if e.instances[instance] == nil {
			e.instances[instance] = make(map[string]struct{})
		}
		e.instances[instance][requestId] = struct{}{}
	}
	return ch
}

func (e *Executor) Complete(requestId string, result QAPIResult) {
	e.requestLoop.Post(Data[QAPIResult]{
		Id:      requestId,
		Payload: result,
	})
	e.instanceCh <- func() {
		for instance, reqSet := range e.instances {
			if _, exists := reqSet[requestId]; exists {
				delete(reqSet, requestId)
				if len(reqSet) == 0 {
					delete(e.instances, instance)
				}
				return
			}
		}
	}
}

func (e *Executor) Cancel(instance string) {
	e.instanceCh <- func() {
		if reqSet, exists := e.instances[instance]; exists {
			for reqId := range reqSet {
				e.requestLoop.Post(Data[QAPIResult]{
					Id:      reqId,
					Payload: QAPIResult{},
				})
			}
			delete(e.instances, instance)
		}
	}
}

func (e *Executor) Run(ctx context.Context) context.CancelFunc {
	cancel, _ := e.requestLoop.Run(ctx)
	return cancel
}
