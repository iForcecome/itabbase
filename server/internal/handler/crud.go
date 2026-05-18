package handler

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/gogf/gf/v2/database/gdb"
	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/net/ghttp"

	"ksogit.kingsoft.net/wpsee/itabbase/server/internal/model"
)

const (
	maxPageSize     = 100
	defaultPageSize = 20
)

func (e *Env) HandleList(c model.Collection) ghttp.HandlerFunc {
	return func(r *ghttp.Request) {
		ctx := r.Context()
		buildModel := func() *gdb.Model {
			m := e.DB.Model(c.Name).Ctx(ctx)
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
			if err := e.resolveIncludes(r.Context(), c, list, includes); err != nil {
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

func (e *Env) HandleGet(c model.Collection) ghttp.HandlerFunc {
	return func(r *ghttp.Request) {
		id := r.Get("id").String()
		rec, err := e.DB.Model(c.Name).Ctx(r.Context()).Where("id", id).One()
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
			if err := e.resolveIncludes(r.Context(), c, []map[string]any{row}, includes); err != nil {
				writeErr(r, http.StatusInternalServerError, "include resolution failed", err)
				return
			}
		}
		r.Response.WriteJsonExit(g.Map{"data": row})
	}
}

func (e *Env) HandleCreate(c model.Collection) ghttp.HandlerFunc {
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
		txErr := e.DB.Transaction(ctx, func(txCtx context.Context, tx gdb.TX) error {
			txCtx = model.WithTxCtx(txCtx, tx)
			rec := model.NewRecord(data)
			if h := c.Hooks.BeforeCreate; h != nil {
				if err := h(txCtx, rec); err != nil {
					return model.UserErr(http.StatusBadRequest, "before-create rejected: "+err.Error())
				}
			}
			if err := e.validateFKs(txCtx, tx, c, rec.Map()); err != nil {
				return model.UserErr(http.StatusBadRequest, err.Error())
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
			if hookErr := c.Hooks.AfterCreate(ctx, model.NewRecord(loadedMap)); hookErr != nil {
				g.Log().Warningf(ctx, "after-create hook %s: %v", c.Name, hookErr)
			}
		}
		r.Response.Status = http.StatusCreated
		r.Response.WriteJsonExit(g.Map{"data": loadedMap})
	}
}

func (e *Env) HandleUpdate(c model.Collection) ghttp.HandlerFunc {
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
		txErr := e.DB.Transaction(ctx, func(txCtx context.Context, tx gdb.TX) error {
			txCtx = model.WithTxCtx(txCtx, tx)
			data["id"] = id
			rec := model.NewRecord(data)
			if h := c.Hooks.BeforeUpdate; h != nil {
				if err := h(txCtx, rec); err != nil {
					return model.UserErr(http.StatusBadRequest, "before-update rejected: "+err.Error())
				}
			}
			patch := map[string]any{}
			for k2, v := range rec.Map() {
				if k2 == "id" {
					continue
				}
				patch[k2] = v
			}
			if err := e.validateFKs(txCtx, tx, c, patch); err != nil {
				return model.UserErr(http.StatusBadRequest, err.Error())
			}
			res, err := tx.Model(c.Name).Ctx(txCtx).Where("id", id).Update(patch)
			if err != nil {
				return err
			}
			affected, _ := res.RowsAffected()
			if affected == 0 {
				return model.UserErr(http.StatusNotFound, "not found")
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
			if hookErr := c.Hooks.AfterUpdate(ctx, model.NewRecord(loadedMap)); hookErr != nil {
				g.Log().Warningf(ctx, "after-update hook %s: %v", c.Name, hookErr)
			}
		}
		r.Response.WriteJsonExit(g.Map{"data": loadedMap})
	}
}

func (e *Env) HandleDelete(c model.Collection) ghttp.HandlerFunc {
	return func(r *ghttp.Request) {
		ctx := r.Context()
		id := r.Get("id").String()
		var snapshot map[string]any
		txErr := e.DB.Transaction(ctx, func(txCtx context.Context, tx gdb.TX) error {
			txCtx = model.WithTxCtx(txCtx, tx)
			loaded, err := tx.Model(c.Name).Ctx(txCtx).Where("id", id).One()
			if err != nil {
				return err
			}
			if loaded.IsEmpty() {
				return model.UserErr(http.StatusNotFound, "not found")
			}
			snapshot = loaded.Map()
			if h := c.Hooks.BeforeDelete; h != nil {
				if err := h(txCtx, model.NewRecord(snapshot)); err != nil {
					return model.UserErr(http.StatusBadRequest, "before-delete rejected: "+err.Error())
				}
			}
			res, err := tx.Model(c.Name).Ctx(txCtx).Where("id", id).Delete()
			if err != nil {
				return err
			}
			affected, _ := res.RowsAffected()
			if affected == 0 {
				return model.UserErr(http.StatusNotFound, "not found")
			}
			return nil
		})
		if !writeOpError(r, txErr, "delete failed") {
			return
		}
		if c.Hooks.AfterDelete != nil && snapshot != nil {
			if hookErr := c.Hooks.AfterDelete(ctx, model.NewRecord(snapshot)); hookErr != nil {
				g.Log().Warningf(ctx, "after-delete hook %s: %v", c.Name, hookErr)
			}
		}
		r.Response.WriteJsonExit(g.Map{"data": g.Map{"id": id}})
	}
}

func (e *Env) validateFKs(ctx context.Context, tx gdb.TX, c model.Collection, data map[string]any) error {
	for _, f := range c.Fields {
		if f.Type != model.TBelongsTo {
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
