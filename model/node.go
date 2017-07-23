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
	//NodeTableName sets the name of the node table
	NodeTableName = "factory-nodes"

	//NodeCapIdxName sets the name of capacity index
	NodeCapIdxName = "cap_idx"

	//NodeTTLIdxName sets the name of capacity index
	NodeTTLIdxName = "ttl_idx"

	//NodeScatterPartitions determines the spread of gsi indexes
	NodeScatterPartitions = int64(10)

	//ErrNodeExists is thrown when a node was expected not to exist
	ErrNodeExists = errors.New("node already exists")

	//ErrNodeNotExists is thrown when a node was expected to exist
	ErrNodeNotExists = errors.New("node does not exist")

	//ErrNodeCapacityUnfit means the node capacity is too low or it unregistered
	ErrNodeCapacityUnfit = errors.New("node capacity low or node no longer exist")

	//ErrNodeReturnUnfit means the node capacity is too low or it unregistered
	ErrNodeReturnUnfit = errors.New("node capacity high or node no longer exist")
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
	PoolID    string `dynamodbav:"pool"`
	TTL       int64  `dynamodbav:"ttl"`
	Cap       int64  `dynamodbav:"cap"`
	Max       int64  `dynamodbav:"max"`
	Partition int64  `dynamodbav:"part"`
}

//RegisterNode will add a node and set the ttl
func RegisterNode(ctx context.Context, db DB, poolID string, ttl time.Time) (*Node, error) {
	uuid, err := uuid.GenerateUUID()
	if err != nil {
		return nil, errors.Wrap(err, "failed to generate node id")
	}

	node := &Node{
		NodePK: NodePK{
			NodeID: uuid,
		},
		PoolID:    poolID,
		Cap:       10,
		Max:       10,
		TTL:       ttl.Unix(),
		Partition: rand.Int63n(NodeScatterPartitions),
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

//ReturnNodeCapacity returns capacity backe to the node
func ReturnNodeCapacity(ctx context.Context, db DB, pk NodePK, size int64) (err error) {
	upd := dynamo.NewUpdate(NodeTableName, pk)
	upd.SetUpdateExpression("SET cap = cap + :size")
	upd.SetConditionExpression("attribute_exists(id) AND cap < #max")
	upd.SetConditionError(ErrNodeReturnUnfit)
	upd.AddExpressionName("#max", "max")
	upd.AddExpressionValue(":size", size)
	if err = upd.ExecuteWithContext(ctx, db); err != nil {
		return errors.Wrap(err, "failed to update node")
	}

	return nil
}

//IncrementNodeTTL will lenghten the ttl of the node
func IncrementNodeTTL(ctx context.Context, db DB, pk NodePK, t time.Duration) (err error) {
	upd := dynamo.NewUpdate(NodeTableName, pk)
	upd.SetUpdateExpression("SET #ttl = :ttl")
	upd.SetConditionExpression("attribute_exists(id)")
	upd.AddExpressionName("#ttl", "ttl")
	upd.AddExpressionValue(":ttl", time.Now().Add(t).Unix())
	upd.SetConditionError(ErrNodeNotExists)
	if err = upd.ExecuteWithContext(ctx, db); err != nil {
		return errors.Wrap(err, "failed to update node")
	}

	return nil
}

//ExpiredNodes queries the ttl index for expired claims
func ExpiredNodes(ctx context.Context, db DB, limit int64) (nodes []*Node, err error) {
	for i := int64(0); i < NodeScatterPartitions; i++ {
		q := dynamo.NewQuery(NodeTableName, "part = :part AND #ttl BETWEEN :minttl AND :maxttl")
		q.SetIndexName(NodeTTLIdxName)
		q.SetLimit(limit)
		q.AddExpressionValue(":part", i)
		q.AddExpressionName("#ttl", "ttl")
		q.AddExpressionValue(":minttl", 1)
		q.AddExpressionValue(":maxttl", time.Now().Unix())

		partNodes := []*Node{}
		if _, err := q.ExecuteWithContext(ctx, db, &partNodes); err != nil {
			return nil, errors.Wrapf(err, "failed to query partition %d", i)
		}

		nodes = append(nodes, partNodes...)
	}

	return nodes, nil
}
