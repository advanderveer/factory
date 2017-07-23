package model

import (
	"context"
	"fmt"
	"math/rand"
	"time"

	dynamo "github.com/advanderveer/go-dynamo"
	uuid "github.com/hashicorp/go-uuid"
	"github.com/pkg/errors"
)

var (
	//ClaimTableName sets the name of the claim table
	ClaimTableName = "factory-claims"

	//ClaimTTLIdxName sets the name of ttl index
	ClaimTTLIdxName = "ttl_idx"

	//ClaimScatterPartitions determines the spread of gsi indexes
	ClaimScatterPartitions = int64(10)

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
	PoolID    string `dynamodbav:"pool"`
	TTL       int64  `dynamodbav:"ttl"`
	Size      int64  `dynamodbav:"size"`
	Partition int64  `dynamodbav:"part"`
	NodeID    string `dynamodbav:"node"`
}

//CreateClaim will add a claim and set the ttl
func CreateClaim(ctx context.Context, db DB, poolID, nodeID string, size int64, ttl time.Time) (*Claim, error) {
	uuid, err := uuid.GenerateUUID()
	if err != nil {
		return nil, errors.Wrap(err, "failed to generate claim id")
	}

	claim := &Claim{
		ClaimPK: ClaimPK{
			ClaimID: uuid,
		},
		PoolID:    poolID,
		NodeID:    nodeID,
		Size:      size,
		TTL:       ttl.Unix(),
		Partition: rand.Int63n(ClaimScatterPartitions),
	}

	put := dynamo.NewPut(ClaimTableName, claim)
	put.SetConditionExpression("attribute_not_exists(id)")
	put.SetConditionError(ErrClaimExists)
	if err = put.ExecuteWithContext(ctx, db); err != nil {
		return nil, errors.Wrap(err, "failed to put claim item")
	}

	return claim, nil
}

//ExpiredClaims queries the ttl index for expired claims
func ExpiredClaims(ctx context.Context, db DB, limit int64) (claims []*Claim, err error) {
	for i := int64(0); i < ClaimScatterPartitions; i++ {
		q := dynamo.NewQuery(ClaimTableName, "part = :part AND #ttl BETWEEN :minttl AND :maxttl")
		q.SetIndexName(ClaimTTLIdxName)
		q.SetLimit(limit)
		q.AddExpressionValue(":part", i)
		q.AddExpressionName("#ttl", "ttl")
		q.AddExpressionValue(":minttl", 1)
		q.AddExpressionValue(":maxttl", time.Now().Unix())

		partClaims := []*Claim{}
		if _, err := q.ExecuteWithContext(ctx, db, &partClaims); err != nil {
			return nil, errors.Wrapf(err, "failed to query partition %d", i)
		}

		claims = append(claims, partClaims...)
	}

	return claims, nil
}

//DeleteClaim will delete a claim
func DeleteClaim(ctx context.Context, db DB, pk ClaimPK) (err error) {
	del := dynamo.NewDelete(ClaimTableName, pk)
	del.SetConditionExpression("attribute_exists(id)")
	del.SetConditionError(ErrClaimNotExists)
	if err = del.ExecuteWithContext(ctx, db); err != nil {
		return errors.Wrap(err, "failed to delete claim item")
	}

	return nil
}
