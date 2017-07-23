package engine

import (
	"context"
	"encoding/json"
	"time"

	"github.com/advanderveer/factory/model"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/request"
	"github.com/pkg/errors"
)

var (
	//PumpCycleInterval determines at what rate the pump makes progress
	PumpCycleInterval = time.Second * 3

	//MaxExpiredNodesPerPartition determines the max nr of claims per partition that can expire per cycle
	MaxExpiredNodesPerPartition = int64(10)

	//MaxExpiredClaimsPerPartition determines the max nr of claims per partition that can expire per cycle
	MaxExpiredClaimsPerPartition = int64(10)
)

//HandleScheduleMessages takes a message and attempts to schedule it
func (e *Engine) HandleScheduleMessages(ctx context.Context, doneCh chan<- struct{}) {
	e.logs.Printf("[INFO] Start handling scheduling messages")
	defer e.logs.Printf("[INFO] Stopped handling scheduling messages")
	defer close(doneCh)

	for {
		if err := NextScheduleMessage(ctx, e.q, func(msgs string) bool {

			e.logs.Printf("[INFO] received schedule message: %v", msgs)
			msg := ScheduleMsg{}
			rerr := json.Unmarshal([]byte(msgs), &msg)
			if rerr != nil {
				e.logs.Printf("[ERROR] failed to unmarshal schedule message: %v", rerr)
				return false
			}

			if rerr = e.Schedule(ctx, msg.PoolID, msg.Size); rerr != nil {
				e.logs.Printf("[INFO] failed to schedule request '%v': %v", msgs, rerr)
				return false
			}

			return true
		}); err != nil {
			if aerr, ok := errors.Cause(err).(awserr.Error); ok && aerr.Code() == request.CanceledErrorCode {
				e.logs.Printf("[INFO] Mext node message receive was cancelled")
				return
			}

			e.logs.Printf("[ERROR] Failed to receive next node message: %v", err)
			return
		}
	}
}

//ExpireClaims queries the database for expired claims and reschedules them
func (e *Engine) ExpireClaims(ctx context.Context) (err error) {
	expired, err := model.ExpiredClaims(ctx, e.db, MaxExpiredClaimsPerPartition)
	if err != nil {
		return errors.Wrap(err, "failed to query expired claims")
	}

	e.logs.Printf("[INFO] found %d expired claims", len(expired))
	for _, claim := range expired {
		err := e.release(ctx, claim)
		if err != nil {
			return errors.Wrapf(err, "failed to release claim '%s'", claim.ClaimPK)
		}
	}

	return nil
}

//ExpireNodes queries the database for expired nodes and removes them
func (e *Engine) ExpireNodes(ctx context.Context) (err error) {
	expired, err := model.ExpiredNodes(ctx, e.db, MaxExpiredNodesPerPartition)
	if err != nil {
		return errors.Wrap(err, "failed to query expired nodes")
	}

	e.logs.Printf("[INFO] found %d expired nodes", len(expired))
	for _, node := range expired {
		err := e.deleteNode(ctx, node.NodePK)
		if err != nil {
			return errors.Wrapf(err, "failed to delete node '%s'", node.NodePK)
		}

		err = e.Evict(ctx, node.NodeID)
		if err != nil {
			return errors.Wrapf(err, "failed to evict node '%s' claims", node.NodePK)
		}
	}

	return nil
}

func (e *Engine) shutdownPump(doneCh chan struct{}) error {
	ctx := context.Background()
	ctx, cancel := context.WithTimeout(ctx, MaxAgentShutdownTime)
	defer cancel()

	e.logs.Printf("[INFO] Waiting for schedule routine to exit")
	select {
	case <-doneCh:
	case <-ctx.Done():
		return errors.New("pump routine didn't exit in time")
	}

	return nil
}

//Pump causes the engine to progress
func (e *Engine) Pump(ctx context.Context) (err error) {
	e.logs.Printf("[INFO] Started engine pump")
	defer e.logs.Printf("[INFO] Exited engine pump")

	doneCh := make(chan struct{})
	go e.HandleScheduleMessages(ctx, doneCh)

	ticker := time.NewTicker(PumpCycleInterval)
	for {
		select {
		case <-ctx.Done():
			return e.shutdownPump(doneCh)
		case <-ticker.C:
			e.logs.Printf("[DEBUG] Started Pump cycle")

			err := e.ExpireClaims(ctx)
			if err != nil {
				return errors.Wrap(err, "failed to expire claims")
			}

			err = e.ExpireNodes(ctx)
			if err != nil {
				return errors.Wrap(err, "failed to expire nodes")
			}
		}
	}
}
