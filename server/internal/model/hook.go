package model

import "context"

// HookFunc is a lifecycle callback on CRUD operations.
type HookFunc func(ctx context.Context, rec *Record) error

// Hooks are optional lifecycle callbacks.
type Hooks struct {
	BeforeCreate HookFunc
	AfterCreate  HookFunc
	BeforeUpdate HookFunc
	AfterUpdate  HookFunc
	BeforeDelete HookFunc
	AfterDelete  HookFunc
}

// Record wraps a row's data map for hook functions.
type Record struct {
	data map[string]any
}

func NewRecord(data map[string]any) *Record {
	if data == nil {
		data = map[string]any{}
	}
	return &Record{data: data}
}

func (r *Record) Get(name string) any { return r.data[name] }

func (r *Record) Set(name string, v any) { r.data[name] = v }

func (r *Record) Has(name string) bool {
	_, ok := r.data[name]
	return ok
}

func (r *Record) Map() map[string]any { return r.data }
