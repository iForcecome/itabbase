package itab

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/gogf/gf/v2/database/gdb"
	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/net/ghttp"
)

const (
	maxPageSize     = 100
	defaultPageSize = 20
)

func (k *Kernel) handleList(c Collection) ghttp.HandlerFunc {
	return func(r *ghttp.Request) {
		ctx := r.Context()
		buildModel := func() *gdb.Model {
			m := k.db.Model(c.Name).Ctx(ctx)
			for key, vals := range r.URL.Query() {
				if !strings.HasPrefix(key, "filter[") || !strings.HasSuffix(key, "]") {
					continue
				}
				if len(vals) == 0 {
					continue
				}
				field := strings.TrimSuffix(strings.TrimPrefix(key, "filter["), "]")
				if !c.HasField(field) {
					continue
				}
				m = m.Where(field, vals[0])
			}
			return m
		}

		applySort := func(m *gdb.Model) *gdb.Model {
			sort := r.GetQuery("sort").String()
			if sort == "" {
				return m.OrderDesc("id")
			}
			for _, s := range strings.Split(sort, ",") {
				s = strings.TrimSpace(s)
				if s == "" {
					continue
				}
				desc := strings.HasPrefix(s, "-")
				field := strings.TrimPrefix(s, "-")
				if !c.HasField(field) {
					continue
				}
				if desc {
					m = m.OrderDesc(field)
				} else {
					m = m.OrderAsc(field)
				}
			}
			return m
		}

		page := r.GetQuery("page").Int()
		if page < 1 {
			page = 1
		}
		size := r.GetQuery("size").Int()
		if size < 1 {
			size = defaultPageSize
		}
		if size > maxPageSize {
			size = maxPageSize
		}

		total, err := buildModel().Count()
		if err != nil {
			writeErr(r, http.StatusInternalServerError, "count failed", err)
			return
		}

		rows, err := applySort(buildModel()).Page(page, size).All()
		if err != nil {
			writeErr(r, http.StatusInternalServerError, "query failed", err)
			return
		}

		list := rows.List()
		if includes := parseIncludes(r); len(includes) > 0 {
			if err := k.resolveIncludes(r.Context(), c, list, includes); err != nil {
				writeErr(r, http.StatusInternalServerError, "include resolution failed", err)
				return
			}
		}

		r.Response.WriteJsonExit(g.Map{
			"data":  list,
			"total": total,
			"page":  page,
			"size":  size,
		})
	}
}

func (k *Kernel) handleGet(c Collection) ghttp.HandlerFunc {
	return func(r *ghttp.Request) {
		id := r.Get("id").String()
		rec, err := k.db.Model(c.Name).Ctx(r.Context()).Where("id", id).One()
		if err != nil {
			writeErr(r, http.StatusInternalServerError, "query failed", err)
			return
		}
		if rec.IsEmpty() {
			writeErr(r, http.StatusNotFound, "not found", nil)
			return
		}
		row := rec.Map()
		if includes := parseIncludes(r); len(includes) > 0 {
			if err := k.resolveIncludes(r.Context(), c, []map[string]any{row}, includes); err != nil {
				writeErr(r, http.StatusInternalServerError, "include resolution failed", err)
				return
			}
		}
		r.Response.WriteJsonExit(g.Map{"data": row})
	}
}

func (k *Kernel) handleCreate(c Collection) ghttp.HandlerFunc {
	return func(r *ghttp.Request) {
		ctx := r.Context()
		body, err := readJSONBody(r)
		if err != nil {
			writeErr(r, http.StatusBadRequest, "invalid json body", err)
			return
		}
		data, err := pickFields(c, body, true)
		if err != nil {
			writeErr(r, http.StatusBadRequest, err.Error(), nil)
			return
		}
		var loadedMap map[string]any
		txErr := k.db.Transaction(ctx, func(txCtx context.Context, tx gdb.TX) error {
			txCtx = WithTxCtx(txCtx, tx)
			rec := NewRecord(data)
			if h := c.Hooks.BeforeCreate; h != nil {
				if err := h(txCtx, rec); err != nil {
					return userErr(http.StatusBadRequest, "before-create rejected: "+err.Error())
				}
			}
			if err := k.validateFKs(txCtx, tx, c, rec.Map()); err != nil {
				return userErr(http.StatusBadRequest, err.Error())
			}
			result, err := tx.Model(c.Name).Ctx(txCtx).Insert(rec.Map())
			if err != nil {
				return err
			}
			id, _ := result.LastInsertId()
			loaded, err := tx.Model(c.Name).Ctx(txCtx).Where("id", id).One()
			if err != nil {
				return err
			}
			if loaded.IsEmpty() {
				return errors.New("insert succeeded but reload returned no row")
			}
			loadedMap = loaded.Map()
			return nil
		})
		if !writeOpError(r, txErr, "create failed") {
			return
		}
		if c.Hooks.AfterCreate != nil {
			if hookErr := c.Hooks.AfterCreate(ctx, NewRecord(loadedMap)); hookErr != nil {
				g.Log().Warningf(ctx, "after-create hook %s: %v", c.Name, hookErr)
			}
		}
		r.Response.Status = http.StatusCreated
		r.Response.WriteJsonExit(g.Map{"data": loadedMap})
	}
}

func (k *Kernel) handleUpdate(c Collection) ghttp.HandlerFunc {
	return func(r *ghttp.Request) {
		ctx := r.Context()
		id := r.Get("id").String()
		body, err := readJSONBody(r)
		if err != nil {
			writeErr(r, http.StatusBadRequest, "invalid json body", err)
			return
		}
		data, err := pickFields(c, body, false)
		if err != nil {
			writeErr(r, http.StatusBadRequest, err.Error(), nil)
			return
		}
		if len(data) == 0 {
			writeErr(r, http.StatusBadRequest, "no valid fields to update", nil)
			return
		}
		var loadedMap map[string]any
		txErr := k.db.Transaction(ctx, func(txCtx context.Context, tx gdb.TX) error {
			txCtx = WithTxCtx(txCtx, tx)
			data["id"] = id
			rec := NewRecord(data)
			if h := c.Hooks.BeforeUpdate; h != nil {
				if err := h(txCtx, rec); err != nil {
					return userErr(http.StatusBadRequest, "before-update rejected: "+err.Error())
				}
			}
			patch := map[string]any{}
			for k2, v := range rec.Map() {
				if k2 == "id" {
					continue
				}
				patch[k2] = v
			}
			if err := k.validateFKs(txCtx, tx, c, patch); err != nil {
				return userErr(http.StatusBadRequest, err.Error())
			}
			res, err := tx.Model(c.Name).Ctx(txCtx).Where("id", id).Update(patch)
			if err != nil {
				return err
			}
			affected, _ := res.RowsAffected()
			if affected == 0 {
				return userErr(http.StatusNotFound, "not found")
			}
			loaded, err := tx.Model(c.Name).Ctx(txCtx).Where("id", id).One()
			if err != nil {
				return err
			}
			loadedMap = loaded.Map()
			return nil
		})
		if !writeOpError(r, txErr, "update failed") {
			return
		}
		if c.Hooks.AfterUpdate != nil {
			if hookErr := c.Hooks.AfterUpdate(ctx, NewRecord(loadedMap)); hookErr != nil {
				g.Log().Warningf(ctx, "after-update hook %s: %v", c.Name, hookErr)
			}
		}
		r.Response.WriteJsonExit(g.Map{"data": loadedMap})
	}
}

func (k *Kernel) handleDelete(c Collection) ghttp.HandlerFunc {
	return func(r *ghttp.Request) {
		ctx := r.Context()
		id := r.Get("id").String()
		var snapshot map[string]any
		txErr := k.db.Transaction(ctx, func(txCtx context.Context, tx gdb.TX) error {
			txCtx = WithTxCtx(txCtx, tx)
			loaded, err := tx.Model(c.Name).Ctx(txCtx).Where("id", id).One()
			if err != nil {
				return err
			}
			if loaded.IsEmpty() {
				return userErr(http.StatusNotFound, "not found")
			}
			snapshot = loaded.Map()
			if h := c.Hooks.BeforeDelete; h != nil {
				if err := h(txCtx, NewRecord(snapshot)); err != nil {
					return userErr(http.StatusBadRequest, "before-delete rejected: "+err.Error())
				}
			}
			res, err := tx.Model(c.Name).Ctx(txCtx).Where("id", id).Delete()
			if err != nil {
				return err
			}
			affected, _ := res.RowsAffected()
			if affected == 0 {
				return userErr(http.StatusNotFound, "not found")
			}
			return nil
		})
		if !writeOpError(r, txErr, "delete failed") {
			return
		}
		if c.Hooks.AfterDelete != nil && snapshot != nil {
			if hookErr := c.Hooks.AfterDelete(ctx, NewRecord(snapshot)); hookErr != nil {
				g.Log().Warningf(ctx, "after-delete hook %s: %v", c.Name, hookErr)
			}
		}
		r.Response.WriteJsonExit(g.Map{"data": g.Map{"id": id}})
	}
}

// validateFKs ensures every belongs_to value in data points to an existing target row.
// Runs inside the active transaction so it sees uncommitted writes from earlier in
// the same operation.
func (k *Kernel) validateFKs(ctx context.Context, tx gdb.TX, c Collection, data map[string]any) error {
	for _, f := range c.Fields {
		if f.Type != TBelongsTo {
			continue
		}
		v, ok := data[f.Name]
		if !ok || v == nil {
			continue
		}
		n, err := tx.Model(f.Target).Ctx(ctx).Where("id", v).Count()
		if err != nil {
			return fmt.Errorf("validate %s: %w", f.Name, err)
		}
		if n == 0 {
			return fmt.Errorf("%s: target %s id=%v does not exist", f.Name, f.Target, v)
		}
	}
	return nil
}

// writeOpError writes the appropriate HTTP response if err is non-nil.
// Returns true when no error (caller should continue), false otherwise.
func writeOpError(r *ghttp.Request, err error, fallbackMsg string) bool {
	if err == nil {
		return true
	}
	var oe *opError
	if errors.As(err, &oe) {
		writeErr(r, oe.Status, oe.Msg, nil)
		return false
	}
	writeErr(r, http.StatusInternalServerError, fallbackMsg, err)
	return false
}

func readJSONBody(r *ghttp.Request) (map[string]any, error) {
	raw := r.GetBody()
	if len(raw) == 0 {
		return map[string]any{}, nil
	}
	var out map[string]any
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// pickFields filters the body to only fields declared in the collection.
// When isCreate is true, missing required fields without a default raise an error.
// Virtual fields (has_many) are always skipped — they have no DB column.
func pickFields(c Collection, body map[string]any, isCreate bool) (map[string]any, error) {
	out := map[string]any{}
	for _, f := range c.Fields {
		if f.IsVirtual() {
			continue
		}
		v, present := body[f.Name]
		if !present {
			if isCreate {
				if f.Default != nil {
					out[f.Name] = f.Default
				} else if f.Required {
					return nil, fmt.Errorf("missing required field: %s", f.Name)
				}
			}
			continue
		}
		out[f.Name] = v
	}
	return out, nil
}

func writeErr(r *ghttp.Request, status int, msg string, cause error) {
	r.Response.Status = status
	payload := g.Map{"error": msg}
	if cause != nil {
		payload["detail"] = cause.Error()
	}
	r.Response.WriteJsonExit(payload)
}
