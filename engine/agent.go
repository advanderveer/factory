package engine

import (
	"context"
	"time"

	"github.com/advanderveer/factory/model"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/request"
	"github.com/pkg/errors"
)

var (
	//MaxAgentShutdownTime determines how long the agent gets for a shutdown
	MaxAgentShutdownTime = time.Second * 5

	//AgentHeartbeatInterval determines how often the agent reports home
	AgentHeartbeatInterval = time.Second * 10
)

//HandleNodeMessage will start handling node messages
func (e *Engine) HandleNodeMessage(ctx context.Context, nodePK model.NodePK, doneCh chan<- struct{}) {
	e.logs.Printf("[INFO] Start handling messages for node '%s'", nodePK)
	defer e.logs.Printf("[INFO] Stopped handling messages for node '%s'", nodePK)
	defer close(doneCh)

	for {
		nextMsg, err := NextNodeMessage(ctx, e.q, nodePK)
		if err != nil {
			if aerr, ok := errors.Cause(err).(awserr.Error); ok && aerr.Code() == request.CanceledErrorCode {
				e.logs.Printf("[INFO] Mext node message receive was cancelled")
				return
			}

			e.logs.Printf("[ERROR] Failed to receive next node message: %v", err)
			return
		}

		if nextMsg == "" {
			continue
		}

		e.logs.Printf("[INFO] Received next node message: '%s'", nextMsg)
		//messages are signals that are immediately removed from the queue, it is a replacement for simple short polling on an api endpoint.
	}
}

func (e *Engine) shutdownAgent(node *model.Node, doneCh chan struct{}) error {
	ctx := context.Background()
	ctx, cancel := context.WithTimeout(ctx, MaxAgentShutdownTime)
	defer cancel()

	e.logs.Printf("[INFO] Deregister node '%s'", node.NodePK)
	err := model.DeregisterNode(ctx, e.db, node.NodePK)
	if err != nil {
		return errors.Wrap(err, "failed to deregister node")
	}

	e.logs.Printf("[DEBUG] Deleting queue for node '%s'", node.NodePK)
	err = DeleteNodeQueue(ctx, e.q, node.NodePK)
	if err != nil {
		return errors.Wrap(err, "failed to delete node queue")
	}

	e.logs.Printf("[INFO] Waiting for handling routine to exit")
	select {
	case <-doneCh:
	case <-ctx.Done():
		return errors.Wrap(err, "handling routine didn't exit in time")
	}

	return nil
}

//Agent will start the node agent
func (e *Engine) Agent(ctx context.Context, poolID string) (err error) {
	e.logs.Printf("[INFO] Starting node agent for pool '%s'", poolID)
	defer e.logs.Printf("[INFO] Exited node agent")

	node, err := model.RegisterNode(ctx, e.db, poolID)
	if err != nil {
		return errors.Wrap(err, "failed to register node")
	}

	e.logs.Printf("[DEBUG] Creating queue for node '%s'", node.NodePK)
	err = CreateNodeQueue(ctx, e.q, node.NodePK)
	if err != nil {
		return errors.Wrap(err, "failed to create node queue")
	}

	doneCh := make(chan struct{})
	go e.HandleNodeMessage(ctx, node.NodePK, doneCh)

	ticker := time.NewTicker(AgentHeartbeatInterval)
	for {
		select {
		case <-ctx.Done():
			return e.shutdownAgent(node, doneCh)
		case <-ticker.C:
			e.logs.Printf("[DEBUG] Exector tick")
			//@TODO send node ttl
		}
	}
}
