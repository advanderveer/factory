package engine

import (
	"log"

	"github.com/advanderveer/factory/model"
)

//Engine controls the factory
type Engine struct {
	logs *log.Logger
	db   model.DB
	q    Q
}

//New creates a new Engine
func New(logs *log.Logger, db model.DB, q Q) *Engine {
	return &Engine{
		logs: logs,
		db:   db,
		q:    q,
	}
}
