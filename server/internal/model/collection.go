package model

import (
	"errors"
	"fmt"
	"regexp"
)

type FieldType string

const (
	TString    FieldType = "string"
	TText      FieldType = "text"
	TInt       FieldType = "int"
	TFloat     FieldType = "float"
	TBool      FieldType = "bool"
	TDateTime  FieldType = "datetime"
	TBelongsTo FieldType = "belongs_to"
	THasMany   FieldType = "has_many"
)

type Collection struct {
	Name    string
	Display string
	Fields  []Field
	ACL     ACL
	Hooks   Hooks

	Source   string // "builtin", "code", or "dynamic"; empty treated as "code"
	Internal bool   // internal collections are hidden from the data management sidebar
}

type Field struct {
	Name     string
	Type     FieldType
	Required bool
	Default  any
	MaxLen   int
	Target   string // for TBelongsTo / THasMany: target collection name
	Through  string // for THasMany: FK column on target
}

func (f Field) IsVirtual() bool {
	return f.Type == THasMany
}

func (f Field) IsRelation() bool {
	return f.Type == TBelongsTo || f.Type == THasMany
}

var IdentRe = regexp.MustCompile(`^[a-z_][a-z0-9_]*$`)

func (c Collection) Validate() error {
	if c.Name == "" {
		return errors.New("collection.Name is required")
	}
	if !IdentRe.MatchString(c.Name) {
		return fmt.Errorf("collection.Name %q must match %s", c.Name, IdentRe.String())
	}
	if c.Name == "id" {
		return fmt.Errorf("collection.Name cannot be %q (reserved)", c.Name)
	}
	if len(c.Fields) == 0 {
		return fmt.Errorf("collection %q must declare at least one field", c.Name)
	}
	seen := map[string]struct{}{"id": {}}
	for _, f := range c.Fields {
		if f.Name == "" {
			return fmt.Errorf("collection %q has a field with empty Name", c.Name)
		}
		if !IdentRe.MatchString(f.Name) {
			return fmt.Errorf("collection %q field %q must match %s", c.Name, f.Name, IdentRe.String())
		}
		if _, dup := seen[f.Name]; dup {
			return fmt.Errorf("collection %q has duplicate field %q (or conflicts with reserved)", c.Name, f.Name)
		}
		seen[f.Name] = struct{}{}
		if !KnownType(f.Type) {
			return fmt.Errorf("collection %q field %q has unknown type %q", c.Name, f.Name, f.Type)
		}
		switch f.Type {
		case TBelongsTo:
			if f.Target == "" {
				return fmt.Errorf("collection %q field %q (belongs_to) requires Target", c.Name, f.Name)
			}
			if f.Through != "" {
				return fmt.Errorf("collection %q field %q (belongs_to) must not set Through", c.Name, f.Name)
			}
		case THasMany:
			if f.Target == "" {
				return fmt.Errorf("collection %q field %q (has_many) requires Target", c.Name, f.Name)
			}
			if f.Through == "" {
				return fmt.Errorf("collection %q field %q (has_many) requires Through (FK column on target)", c.Name, f.Name)
			}
		default:
			if f.Target != "" || f.Through != "" {
				return fmt.Errorf("collection %q field %q: Target/Through only valid for relation types", c.Name, f.Name)
			}
		}
	}
	return nil
}

func (c Collection) HasField(name string) bool {
	if name == "id" {
		return true
	}
	for _, f := range c.Fields {
		if f.Name == name {
			return true
		}
	}
	return false
}

func (c Collection) FindField(name string) (Field, bool) {
	for _, f := range c.Fields {
		if f.Name == name {
			return f, true
		}
	}
	return Field{}, false
}

func KnownType(t FieldType) bool {
	switch t {
	case TString, TText, TInt, TFloat, TBool, TDateTime, TBelongsTo, THasMany:
		return true
	}
	return false
}
