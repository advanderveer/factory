package engine

import (
	"context"

	"github.com/advanderveer/factory/model"
	"github.com/pkg/errors"
)

//Evict will release all claims for a node and resubmit to schedule queue
//@TODO this needs a distributed lock to prevent multiple client from evicting the same claims ands
func (e *Engine) Evict(ctx context.Context, nodeID string) error {
	e.logs.Printf("[INFO] Evicting node '%s'", nodeID)

	claims, err := model.NodeClaims(ctx, e.db, nodeID)
	if err != nil {
		return errors.Wrap(err, "failed to find node claims")
	}

	e.logs.Printf("[INFO] Found %d claims for eviction", len(claims))
	for _, claim := range claims {
		err := e.release(ctx, claim)
		if err != nil {
			return errors.Wrapf(err, "failed to release claim '%s'", claim.ClaimPK)
		}
	}

	return nil
}
