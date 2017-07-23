package engine

import (
	"context"
	"encoding/json"
	"time"

	"github.com/advanderveer/factory/model"
	"github.com/cenkalti/backoff"
	"github.com/pkg/errors"
)

var (
	//MaxClaimRetries determines how often a query + claim is retried
	MaxClaimRetries = uint64(10)

	//MaxClaimCandidates is the max number of candidates that will be considered
	MaxClaimCandidates = int64(10)

	//ClaimHeartbeatTimeout determines how often the node has to call in
	ClaimHeartbeatTimeout = time.Second * 30
)

//Schedule will place a task on a node
func (e *Engine) Schedule(ctx context.Context, poolID string, size int64) error {

	var claimed *model.Node
	operation := func() error {
		e.logs.Printf("[DEBUG] quering nodes with at least capacity >= %d", size)
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

	ttl := time.Now().Add(ClaimHeartbeatTimeout)
	claim, err := model.CreateClaim(ctx, e.db, poolID, claimed.NodeID, size, ttl)
	if err != nil {
		return errors.Wrap(err, "failed to create claim")
	}

	msg := RunMsg{
		Size: claim.Size,
	}

	data, err := json.Marshal(msg)
	if err != nil {
		return errors.Wrap(err, "failed to marshal claim for messaging")
	}

	msgs := string(data)
	e.logs.Printf("[DEBUG] Dispatching message '%s' to node '%s'", msgs, claimed.NodePK)
	err = SendNodeMessage(ctx, e.q, claimed.NodePK, msgs)
	if err != nil {
		return errors.Wrapf(err, "failed to send node message '%s'", msgs)
	}

	return nil
}
