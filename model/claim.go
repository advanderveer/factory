package model

import (
	"context"
	"fmt"

	dynamo "github.com/advanderveer/go-dynamo"
	uuid "github.com/hashicorp/go-uuid"
	"github.com/pkg/errors"
)

var (
	//ClaimTableName sets the name of the claim table
	ClaimTableName = "factory-claims"

	//ErrClaimExists is thrown when a claim was expected not to exist
	ErrClaimExists = errors.New("claim already exists")

	//ErrClaimNotExists is thrown when a claim was expected to exist
	ErrClaimNotExists = errors.New("claim does not exist")
)

//ClaimPK is the primary key
type ClaimPK struct {
	ClaimID string `dynamodbav:"id"`
}

func (pk ClaimPK) String() string {
	return fmt.Sprintf("%s", pk.ClaimID)
}

//Claim item
type Claim struct {
	ClaimPK
	PoolID string `dynamodbav:"pool"`
	TTL    int64  `dynamodbav:"ttl"`
	Size   int64  `dynamodbav:"size"`
	NodeID string `dynamodbav:"node"`
}

//CreateClaim will add a claim and set the ttl
func CreateClaim(ctx context.Context, db DB, poolID, nodeID string, size int64) (*Claim, error) {
	uuid, err := uuid.GenerateUUID()
	if err != nil {
		return nil, errors.Wrap(err, "failed to generate claim id")
	}

	claim := &Claim{
		ClaimPK: ClaimPK{
			ClaimID: uuid,
		},
		PoolID: poolID,
		NodeID: nodeID,
		Size:   size,
	}

	put := dynamo.NewPut(ClaimTableName, claim)
	put.SetConditionExpression("attribute_not_exists(id)")
	put.SetConditionError(ErrClaimExists)
	if err = put.ExecuteWithContext(ctx, db); err != nil {
		return nil, errors.Wrap(err, "failed to put claim item")
	}

	return claim, nil
}
