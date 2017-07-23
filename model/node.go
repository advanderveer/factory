package model

import (
	"context"
	"fmt"

	dynamo "github.com/advanderveer/go-dynamo"
	uuid "github.com/hashicorp/go-uuid"
	"github.com/pkg/errors"
)

var (
	//NodeTableName sets the name of the node table
	NodeTableName = "factory-nodes"

	//NodeCapIdxName sets the name of capacity index
	NodeCapIdxName = "cap_idx"

	//ErrNodeExists is thrown when a node was expected not to exist
	ErrNodeExists = errors.New("node already exists")

	//ErrNodeNotExists is thrown when a node was expected to exist
	ErrNodeNotExists = errors.New("node does not exist")

	//ErrNodeCapacityUnfit means the node capacity is too low or it unregistered
	ErrNodeCapacityUnfit = errors.New("node capacity low or node no longer exist")
)

//NodePK is the primary key
type NodePK struct {
	NodeID string `dynamodbav:"id"`
}

func (pk NodePK) String() string {
	return fmt.Sprintf("%s", pk.NodeID)
}

//Node item
type Node struct {
	NodePK
	PoolID string `dynamodbav:"pool"`
	TTL    int64  `dynamodbav:"ttl"`
	Cap    int64  `dynamodbav:"cap"`
}

//RegisterNode will add a node and set the ttl
func RegisterNode(ctx context.Context, db DB, poolID string) (*Node, error) {
	uuid, err := uuid.GenerateUUID()
	if err != nil {
		return nil, errors.Wrap(err, "failed to generate node id")
	}

	node := &Node{
		NodePK: NodePK{
			NodeID: uuid,
		},
		PoolID: poolID,
		Cap:    10,
	}

	put := dynamo.NewPut(NodeTableName, node)
	put.SetConditionExpression("attribute_not_exists(id)")
	put.SetConditionError(ErrNodeExists)
	if err = put.ExecuteWithContext(ctx, db); err != nil {
		return nil, errors.Wrap(err, "failed to put node item")
	}

	return node, nil
}

//DeregisterNode will remove a node
func DeregisterNode(ctx context.Context, db DB, pk NodePK) (err error) {
	del := dynamo.NewDelete(NodeTableName, pk)
	del.SetConditionExpression("attribute_exists(id)")
	del.SetConditionError(ErrNodeNotExists)
	if err = del.ExecuteWithContext(ctx, db); err != nil {
		return errors.Wrap(err, "failed to delete node item")
	}

	return nil
}

//NodesWithEnoughCapacity will return nodes that have enough cap
func NodesWithEnoughCapacity(ctx context.Context, db DB, poolID string, size int64, limit int64) (nodes []*Node, err error) {
	q := dynamo.NewQuery(NodeTableName, "#pool = :pool AND cap >= :size")
	q.SetIndexName(NodeCapIdxName)
	q.SetLimit(limit)
	q.AddExpressionName("#pool", "pool")
	q.AddExpressionValue(":pool", poolID)
	q.AddExpressionValue(":size", size)
	if _, err = q.ExecuteWithContext(ctx, db, &nodes); err != nil {
		return nil, errors.Wrap(err, "failed to query nodes")
	}

	return nodes, nil
}

//ClaimNodeCapacity will atomically reduce the nodes capacity
func ClaimNodeCapacity(ctx context.Context, db DB, pk NodePK, size int64) (err error) {
	upd := dynamo.NewUpdate(NodeTableName, pk)
	upd.SetUpdateExpression("SET cap = cap - :size")
	upd.SetConditionExpression("attribute_exists(id) AND cap >= :size")
	upd.SetConditionError(ErrNodeCapacityUnfit)
	upd.AddExpressionValue(":size", size)
	if err = upd.ExecuteWithContext(ctx, db); err != nil {
		return errors.Wrap(err, "failed to update node")
	}

	return nil
}
