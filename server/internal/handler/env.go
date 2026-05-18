package handler

import (
	"sync"

	"github.com/gogf/gf/v2/database/gdb"

	"ksogit.kingsoft.net/wpsee/itabbase/server/internal/model"
)

// Env holds shared dependencies for all HTTP handlers.
type Env struct {
	DB            gdb.DB
	Auth          model.AuthAdapter
	ACLDisabled   bool
	Mu            *sync.RWMutex
	Collections   *[]model.Collection
	ReservedPaths []string
}

// FindCollection returns the registered collection with the given name.
func (e *Env) FindCollection(name string) (model.Collection, bool) {
	for _, c := range *e.Collections {
		if c.Name == name {
			return c, true
		}
	}
	return model.Collection{}, false
}

// AllCollections returns a copy of the current collection slice.
func (e *Env) AllCollections() []model.Collection {
	return *e.Collections
}
