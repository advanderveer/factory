package engine

import (
	"log"

	"github.com/advanderveer/factory/model"
	dynamo "github.com/advanderveer/go-dynamo"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbiface"
	"github.com/pkg/errors"
)

//Engine controls the factory
type Engine struct {
	logs *log.Logger
	db   dynamodbiface.DynamoDBAPI
}

//New creates a new Engine
func New(logs *log.Logger, db dynamodbiface.DynamoDBAPI) *Engine {
	return &Engine{
		logs: logs,
		db:   db,
	}
}

//Schedule will attempt to place a task
func (e *Engine) Schedule() (err error) {
	q := dynamo.NewQuery("factory-nodes", "#pool = :pool AND cap >= :size")
	q.SetIndexName("cap_idx")
	q.AddExpressionName("#pool", "pool")
	q.AddExpressionValue(":pool", "my-pool")
	q.AddExpressionValue(":size", 5)

	candidates := []*model.Node{}
	if _, err = q.Execute(e.db, candidates); err != nil {
		return errors.Wrap(err, "failed to query nodes with capacity")
	}

	return nil
}
