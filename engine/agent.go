package engine

import (
	"context"
	"time"

	"github.com/advanderveer/factory/model"
	"github.com/pkg/errors"
)

//RunAgent will start the node agent
func (e *Engine) RunAgent(ctx context.Context, poolID string) (err error) {
	e.logs.Printf("[INFO] Starting node agent for pool '%s'", poolID)
	defer e.logs.Printf("[INFO] Exited node agent")

	node, err := model.RegisterNode(e.db, poolID)
	if err != nil {
		return errors.Wrap(err, "failed to register node")
	}

	ticker := time.NewTicker(time.Second)
	for {
		select {
		case <-ctx.Done():
			e.logs.Printf("[INFO] Deregister node '%s'", node.NodePK)
			err := model.DeregisterNode(e.db, node.NodePK)
			if err != nil {
				return errors.Wrap(err, "failed to deregister node")
			}

			return nil
		case <-ticker.C:
			e.logs.Printf("[DEBUG] Exector tick")
			//tick
		}
	}
}
