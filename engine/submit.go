package engine

import (
	"context"
	"encoding/json"

	"github.com/pkg/errors"
)

//Submit will submit a task for execution on a node
func (e *Engine) Submit(ctx context.Context, poolID string, size int64) error {
	data := ScheduleMsg{
		Size:   size,
		PoolID: poolID,
	}

	msg, err := json.Marshal(data)
	if err != nil {
		return errors.Wrap(err, "failed to marshal schedule message")
	}

	if err := SendScheduleMessage(ctx, e.q, string(msg)); err != nil {
		return errors.Wrap(err, "failed to send schedule message")
	}

	return nil
}
