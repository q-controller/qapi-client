package monitor

import (
	"context"
	"time"

	"github.com/q-controller/qapi-client/src/client"
)

type ExecuteResult struct {
	resultCh <-chan client.QAPIResult
	instance string
}

func (r *ExecuteResult) Get(ctx context.Context, timeout time.Duration) (*client.QAPIResult, bool) {
	newCtx := ctx
	if timeout > 0 {
		var cancel context.CancelFunc
		newCtx, cancel = context.WithTimeout(ctx, timeout)
		defer cancel()
	}

	select {
	case rv, ok := <-r.resultCh:
		if !ok {
			return nil, false
		}
		return &rv, true
	case <-newCtx.Done():
		return nil, false
	}
}

func (r *ExecuteResult) Instance() string {
	return r.instance
}

func NewExecuteResult(resultCh <-chan client.QAPIResult, instance string) *ExecuteResult {
	return &ExecuteResult{
		resultCh: resultCh,
		instance: instance,
	}
}
