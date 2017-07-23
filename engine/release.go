package engine

import (
	"context"

	"github.com/advanderveer/factory/model"
	"github.com/pkg/errors"
)

func (e *Engine) release(ctx context.Context, claim *model.Claim) error {
	if rerr := model.ReturnNodeCapacity(ctx, e.db, model.NodePK{NodeID: claim.NodeID}, claim.Size); rerr != nil {
		e.logs.Printf("[WARN] failed to return node capacity: %v", rerr)
	}

	if serr := e.Submit(ctx, claim.PoolID, claim.Size); serr != nil {
		e.logs.Printf("[WARN] failed to re-submit claim as task: %v", serr)
	}

	err := model.DeleteClaim(ctx, e.db, claim.ClaimPK)
	if err != nil {
		return errors.Wrapf(err, "failed to delete claim '%s'", claim.ClaimPK)
	}

	return nil
}

func (e *Engine) deleteNode(ctx context.Context, pk model.NodePK) error {
	e.logs.Printf("[INFO] Deregister node '%s'", pk)
	err := model.DeregisterNode(ctx, e.db, pk)
	if err != nil {
		return errors.Wrap(err, "failed to deregister node")
	}

	e.logs.Printf("[DEBUG] Deleting queue for node '%s'", pk)
	err = DeleteNodeQueue(ctx, e.q, pk)
	if err != nil {
		return errors.Wrap(err, "failed to delete node queue")
	}

	return nil
}
