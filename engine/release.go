package engine

import (
	"context"

	"github.com/advanderveer/factory/model"
	"github.com/pkg/errors"
)

func (e *Engine) release(ctx context.Context, claim *model.Claim) error {
	if rerr := model.ReturnNodeCapacity(ctx, e.db, model.NodePK{NodeID: claim.NodeID}, claim.Size); rerr != nil {
		e.logs.Printf("[ERROR] failed to return node capacity: %v", rerr)
	}

	if serr := e.Submit(ctx, claim.PoolID, claim.Size); serr != nil {
		e.logs.Printf("[ERROR] failed to re-submit claim as task: %v", serr)
	}

	err := model.DeleteClaim(ctx, e.db, claim.ClaimPK)
	if err != nil {
		return errors.Wrapf(err, "failed to delete claim '%s'", claim.ClaimPK)
	}

	return nil
}
