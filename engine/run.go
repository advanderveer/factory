package engine

import (
	"context"

	"github.com/advanderveer/factory/model"
	"github.com/cenkalti/backoff"
	"github.com/pkg/errors"
)

var (
	//MaxClaimRetries determines how often a query + claim is retried
	MaxClaimRetries = uint64(10)

	//MaxClaimCandidates is the max number of candidates that will be considered
	MaxClaimCandidates = int64(10)
)

//Run will submit a task for execution on a node
func (e *Engine) Run(ctx context.Context, poolID string, size int64) error {

	var claimed *model.Node
	operation := func() error {
		e.logs.Printf("[DEBUG] quering nodes with at capacity >= %d", size)
		nodes, err := model.NodesWithEnoughCapacity(ctx, e.db, poolID, size, MaxClaimCandidates)
		if err != nil {
			return errors.Wrap(err, "failed to find nodes with enough capacity")
		}

		e.logs.Printf("[DEBUG] found %d nodes with enough capacity", len(nodes))
		for _, node := range nodes {
			err = model.ClaimNodeCapacity(ctx, e.db, node.NodePK, size)
			if err != nil {
				if errors.Cause(err) == model.ErrNodeCapacityUnfit {
					continue
				}

				return errors.Wrap(err, "failed to claim node capacity")
			}

			e.logs.Printf("[INFO] successfully claimed %d capacity on node %v", size, node.NodePK)
			claimed = node

			return nil //no need to consider other nodes, we succeeded
		}

		return errors.New("no nodes with enough capacity")
	}

	b := backoff.NewExponentialBackOff()
	err := backoff.Retry(operation, backoff.WithContext(
		backoff.WithMaxTries(b, MaxClaimRetries), ctx))
	if err != nil || claimed == nil {
		return errors.Wrap(err, "failed to claim node capacity")
	}

	claim, err := model.CreateClaim(ctx, e.db, poolID, claimed.NodeID, size)
	if err != nil {
		return errors.Wrap(err, "failed to create claim")
	}

	_ = claim
	//@TODO at this point we have claimed capacity on a node
	//@TODO create a claim object that expires, the node is expected to receive messages in real-time. If cannot handle the messages in time, or crashes while handling them, or decides not to handle them. The claim should expire and return back to (priority) scheduling.
	msg := "hello, world"

	e.logs.Printf("[DEBUG] Dispatching message '%s' to node '%s'", msg, claimed.NodePK)
	err = SendNodeMessage(ctx, e.q, claimed.NodePK, msg)
	if err != nil {
		return errors.Wrapf(err, "failed to send node message '%s'", msg)
	}

	return nil
}
