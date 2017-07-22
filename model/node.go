package model

import (
	"fmt"

	dynamo "github.com/advanderveer/go-dynamo"
	uuid "github.com/hashicorp/go-uuid"
	"github.com/pkg/errors"
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

var (
	//NodeTableName sets the name of the node table
	NodeTableName = "factory-nodes"
)

//RegisterNode will add a node and set the ttl
func RegisterNode(db DB, poolID string) (*Node, error) {
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
	if err = put.Execute(db); err != nil {
		return nil, errors.Wrap(err, "failed to put node item")
	}

	return node, nil
}

//DeregisterNode will remove a node
func DeregisterNode(db DB, pk NodePK) (err error) {
	del := dynamo.NewDelete(NodeTableName, pk)
	if err = del.Execute(db); err != nil {
		return errors.Wrap(err, "failed to delete node item")
	}

	return nil
}
