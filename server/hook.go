package itab

import "context"

// Record is a mutable view of a row's data, passed to lifecycle hooks.
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

// Hook is called at lifecycle points.
//
// Returning an error from a Before* hook aborts the operation with HTTP 400.
// After* errors are logged but the write is not rolled back (no tx in v0.2).
type Hook func(ctx context.Context, rec *Record) error

type Hooks struct {
	BeforeCreate Hook
	AfterCreate  Hook
	BeforeUpdate Hook
	AfterUpdate  Hook
	BeforeDelete Hook
	AfterDelete  Hook
}
