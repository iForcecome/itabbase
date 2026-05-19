package handler

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/net/ghttp"

	"ksogit.kingsoft.net/wpsee/itabbase/server/internal/model"
	"ksogit.kingsoft.net/wpsee/itabbase/server/internal/schema"
)

// DynamicCRUD returns a handler that resolves the collection name from the URL
// parameter and delegates to the appropriate CRUD handler with ACL.
func (e *Env) DynamicCRUD(action string) ghttp.HandlerFunc {
	return func(r *ghttp.Request) {
		colName := r.GetRouter("_col").String()
		e.Mu.RLock()
		c, ok := e.FindCollection(colName)
		e.Mu.RUnlock()
		if !ok {
			writeErr(r, http.StatusNotFound, fmt.Sprintf("collection %q not found", colName), nil)
			return
		}
		var inner ghttp.HandlerFunc
		switch action {
		case model.ActionList:
			inner = e.HandleList(c)
		case model.ActionGet:
			inner = e.HandleGet(c)
		case model.ActionCreate:
			inner = e.HandleCreate(c)
		case model.ActionUpdate:
			inner = e.HandleUpdate(c)
		case model.ActionDelete:
			inner = e.HandleDelete(c)
		default:
			writeErr(r, http.StatusBadRequest, "unknown action", nil)
			return
		}
		e.ACLWrap(c, action, inner)(r)
	}
}

// HandleCreateCollection handles POST /meta/collections.
func (e *Env) HandleCreateCollection(r *ghttp.Request) {
	ctx := r.Context()
	body, err := readJSONBody(r)
	if err != nil {
		writeErr(r, http.StatusBadRequest, "invalid json body", err)
		return
	}
	name, _ := body["name"].(string)
	display, _ := body["display"].(string)
	ownerField, _ := body["owner_field"].(string)
	titleField, _ := body["title_field"].(string)
	fieldsRaw, _ := body["fields"].([]any)

	if name == "" {
		writeErr(r, http.StatusBadRequest, "name is required", nil)
		return
	}
	if !model.IdentRe.MatchString(name) {
		writeErr(r, http.StatusBadRequest, fmt.Sprintf("name %q must match %s", name, model.IdentRe.String()), nil)
		return
	}
	tableName := "t_" + name
	for _, rp := range e.ReservedPaths {
		if name == rp {
			writeErr(r, http.StatusConflict, fmt.Sprintf("name %q is reserved", name), nil)
			return
		}
	}
	e.Mu.RLock()
	_, exists := e.FindCollection(name)
	e.Mu.RUnlock()
	if exists {
		writeErr(r, http.StatusConflict, fmt.Sprintf("collection %q already exists", name), nil)
		return
	}

	fields := schema.ParseFieldsFromBody(fieldsRaw)
	if len(fields) == 0 {
		writeErr(r, http.StatusBadRequest, "at least one field is required", nil)
		return
	}

	c := model.Collection{
		Name:       name,
		TableName:  tableName,
		Display:    display,
		Fields:     fields,
		OwnerField: ownerField,
		TitleField: titleField,
		Source:     model.SourceDynamic,
	}
	if err := c.Validate(); err != nil {
		writeErr(r, http.StatusBadRequest, err.Error(), nil)
		return
	}

	metaRow := g.Map{
		"name":       name,
		"table_name": tableName,
		"display":    display,
	}
	if ownerField != "" {
		metaRow["owner_field"] = ownerField
	}
	if titleField != "" {
		metaRow["title_field"] = titleField
	}
	if _, err := e.DB.Model(model.BuiltinMetaCollections).Ctx(ctx).Insert(metaRow); err != nil {
		writeErr(r, http.StatusInternalServerError, "failed to save collection metadata", err)
		return
	}
	for i, f := range fields {
		fm := g.Map{
			"collection_name": name,
			"name":            f.Name,
			"type":            string(f.Type),
			"required":        f.Required,
			"max_len":         f.MaxLen,
			"target":          f.Target,
			"through":         f.Through,
			"sort":            i,
		}
		if f.Default != nil {
			if dv, err := json.Marshal(f.Default); err == nil {
				fm["default_value"] = string(dv)
			}
		}
		if _, err := e.DB.Model(model.BuiltinMetaFields).Ctx(ctx).Insert(fm); err != nil {
			writeErr(r, http.StatusInternalServerError, "failed to save field metadata", err)
			return
		}
	}

	dialect := schema.Dialect(e.DB)
	stmt := schema.BuildCreateTable(dialect, c)
	if _, err := e.DB.Exec(ctx, stmt); err != nil {
		e.DB.Model(model.BuiltinMetaFields).Ctx(ctx).Where("collection_name", name).Delete()
		e.DB.Model(model.BuiltinMetaCollections).Ctx(ctx).Where("name", name).Delete()
		writeErr(r, http.StatusInternalServerError, "failed to create table", err)
		return
	}

	e.Mu.Lock()
	*e.Collections = append(*e.Collections, c)
	e.Mu.Unlock()

	r.Response.Status = http.StatusCreated
	r.Response.WriteJsonExit(g.Map{"data": CollectionMeta(c)})
}

// HandleUpdateCollection handles PATCH /meta/collections/:name.
func (e *Env) HandleUpdateCollection(r *ghttp.Request) {
	ctx := r.Context()
	name := r.GetRouter("name").String()

	e.Mu.RLock()
	c, ok := e.FindCollection(name)
	e.Mu.RUnlock()
	if !ok || c.Source != model.SourceDynamic {
		writeErr(r, http.StatusNotFound, "dynamic collection not found", nil)
		return
	}

	body, err := readJSONBody(r)
	if err != nil {
		writeErr(r, http.StatusBadRequest, "invalid json body", err)
		return
	}

	patch := g.Map{}
	if v, ok := body["display"]; ok {
		patch["display"] = v
	}
	if v, ok := body["icon"]; ok {
		patch["icon"] = v
	}
	if v, ok := body["sort"]; ok {
		patch["sort"] = v
	}
	if v, ok := body["title_field"]; ok {
		patch["title_field"] = v
	}
	if v, ok := body["owner_field"]; ok {
		patch["owner_field"] = v
	}
	if len(patch) == 0 {
		writeErr(r, http.StatusBadRequest, "nothing to update", nil)
		return
	}

	if _, err := e.DB.Model(model.BuiltinMetaCollections).Ctx(ctx).Where("name", name).Update(patch); err != nil {
		writeErr(r, http.StatusInternalServerError, "update failed", err)
		return
	}

	e.Mu.Lock()
	for i := range *e.Collections {
		if (*e.Collections)[i].Name == name {
			if d, ok := patch["display"].(string); ok {
				(*e.Collections)[i].Display = d
			}
			if v, ok := patch["title_field"].(string); ok {
				(*e.Collections)[i].TitleField = v
			}
			if v, ok := patch["owner_field"].(string); ok {
				(*e.Collections)[i].OwnerField = v
			}
			c = (*e.Collections)[i]
			break
		}
	}
	e.Mu.Unlock()

	r.Response.WriteJsonExit(g.Map{"data": CollectionMeta(c)})
}

// HandleDeleteCollection handles DELETE /meta/collections/:name.
func (e *Env) HandleDeleteCollection(r *ghttp.Request) {
	ctx := r.Context()
	name := r.GetRouter("name").String()

	e.Mu.RLock()
	c, ok := e.FindCollection(name)
	e.Mu.RUnlock()
	if !ok {
		writeErr(r, http.StatusNotFound, "collection not found", nil)
		return
	}
	if c.Source != model.SourceDynamic {
		writeErr(r, http.StatusForbidden, "cannot delete non-dynamic collection", nil)
		return
	}

	dialect := schema.Dialect(e.DB)
	dropSQL := fmt.Sprintf("DROP TABLE IF EXISTS %s", schema.QuoteIdent(dialect, c.DBTable()))
	if _, err := e.DB.Exec(ctx, dropSQL); err != nil {
		writeErr(r, http.StatusInternalServerError, "failed to drop table", err)
		return
	}

	e.DB.Model(model.BuiltinMetaFields).Ctx(ctx).Where("collection_name", name).Delete()
	e.DB.Model(model.BuiltinMetaCollections).Ctx(ctx).Where("name", name).Delete()

	e.Mu.Lock()
	cols := *e.Collections
	for i := range cols {
		if cols[i].Name == name {
			*e.Collections = append(cols[:i], cols[i+1:]...)
			break
		}
	}
	e.Mu.Unlock()

	r.Response.WriteJsonExit(g.Map{"data": g.Map{"name": name}})
}

// HandleAddField handles POST /meta/collections/:name/fields.
func (e *Env) HandleAddField(r *ghttp.Request) {
	ctx := r.Context()
	colName := r.GetRouter("name").String()

	e.Mu.RLock()
	c, ok := e.FindCollection(colName)
	e.Mu.RUnlock()
	if !ok || c.Source != model.SourceDynamic {
		writeErr(r, http.StatusNotFound, "dynamic collection not found", nil)
		return
	}

	body, err := readJSONBody(r)
	if err != nil {
		writeErr(r, http.StatusBadRequest, "invalid json body", err)
		return
	}

	f := schema.ParseOneField(body)
	if f.Name == "" || f.Type == "" {
		writeErr(r, http.StatusBadRequest, "name and type are required", nil)
		return
	}
	if !model.IdentRe.MatchString(f.Name) {
		writeErr(r, http.StatusBadRequest, fmt.Sprintf("field name %q invalid", f.Name), nil)
		return
	}
	if !model.KnownType(f.Type) {
		writeErr(r, http.StatusBadRequest, fmt.Sprintf("unknown field type %q", f.Type), nil)
		return
	}
	if c.HasField(f.Name) {
		writeErr(r, http.StatusConflict, fmt.Sprintf("field %q already exists", f.Name), nil)
		return
	}

	display, _ := body["display"].(string)
	fm := g.Map{
		"collection_name": colName,
		"name":            f.Name,
		"type":            string(f.Type),
		"display":         display,
		"required":        f.Required,
		"max_len":         f.MaxLen,
		"target":          f.Target,
		"through":         f.Through,
		"sort":            len(c.Fields),
	}
	if f.Default != nil {
		if dv, err := json.Marshal(f.Default); err == nil {
			fm["default_value"] = string(dv)
		}
	}

	if _, err := e.DB.Model(model.BuiltinMetaFields).Ctx(ctx).Insert(fm); err != nil {
		writeErr(r, http.StatusInternalServerError, "failed to save field metadata", err)
		return
	}

	if !f.IsVirtual() {
		dialect := schema.Dialect(e.DB)
		stmt := schema.BuildAddColumn(dialect, c.DBTable(), f)
		if _, err := e.DB.Exec(ctx, stmt); err != nil {
			e.DB.Model(model.BuiltinMetaFields).Ctx(ctx).
				Where("collection_name", colName).
				Where("name", f.Name).
				Delete()
			writeErr(r, http.StatusInternalServerError, "failed to add column", err)
			return
		}
	}

	e.Mu.Lock()
	for i := range *e.Collections {
		if (*e.Collections)[i].Name == colName {
			(*e.Collections)[i].Fields = append((*e.Collections)[i].Fields, f)
			c = (*e.Collections)[i]
			break
		}
	}
	e.Mu.Unlock()

	r.Response.Status = http.StatusCreated
	r.Response.WriteJsonExit(g.Map{"data": CollectionMeta(c)})
}

// HandleUpdateField handles PATCH /meta/collections/:name/fields/:fieldName.
func (e *Env) HandleUpdateField(r *ghttp.Request) {
	ctx := r.Context()
	colName := r.GetRouter("name").String()
	fieldName := r.GetRouter("fieldName").String()

	e.Mu.RLock()
	c, ok := e.FindCollection(colName)
	e.Mu.RUnlock()
	if !ok || c.Source != model.SourceDynamic {
		writeErr(r, http.StatusNotFound, "dynamic collection not found", nil)
		return
	}

	body, err := readJSONBody(r)
	if err != nil {
		writeErr(r, http.StatusBadRequest, "invalid json body", err)
		return
	}

	patch := g.Map{}
	if v, ok := body["display"]; ok {
		patch["display"] = v
	}
	if v, ok := body["required"]; ok {
		patch["required"] = v
	}
	if v, ok := body["sort"]; ok {
		patch["sort"] = v
	}
	if len(patch) == 0 {
		writeErr(r, http.StatusBadRequest, "nothing to update", nil)
		return
	}

	res, err := e.DB.Model(model.BuiltinMetaFields).Ctx(ctx).
		Where("collection_name", colName).
		Where("name", fieldName).
		Update(patch)
	if err != nil {
		writeErr(r, http.StatusInternalServerError, "update failed", err)
		return
	}
	affected, _ := res.RowsAffected()
	if affected == 0 {
		writeErr(r, http.StatusNotFound, "field not found", nil)
		return
	}

	if req, ok := body["required"].(bool); ok {
		e.Mu.Lock()
		for i := range *e.Collections {
			if (*e.Collections)[i].Name == colName {
				for j := range (*e.Collections)[i].Fields {
					if (*e.Collections)[i].Fields[j].Name == fieldName {
						(*e.Collections)[i].Fields[j].Required = req
					}
				}
				c = (*e.Collections)[i]
				break
			}
		}
		e.Mu.Unlock()
	}

	r.Response.WriteJsonExit(g.Map{"data": CollectionMeta(c)})
}

// HandleDeleteField handles DELETE /meta/collections/:name/fields/:fieldName.
func (e *Env) HandleDeleteField(r *ghttp.Request) {
	ctx := r.Context()
	colName := r.GetRouter("name").String()
	fieldName := r.GetRouter("fieldName").String()

	e.Mu.RLock()
	c, ok := e.FindCollection(colName)
	e.Mu.RUnlock()
	if !ok || c.Source != model.SourceDynamic {
		writeErr(r, http.StatusNotFound, "dynamic collection not found", nil)
		return
	}

	if fieldName == "id" {
		writeErr(r, http.StatusForbidden, "cannot delete id field", nil)
		return
	}

	res, err := e.DB.Model(model.BuiltinMetaFields).Ctx(ctx).
		Where("collection_name", colName).
		Where("name", fieldName).
		Delete()
	if err != nil {
		writeErr(r, http.StatusInternalServerError, "delete failed", err)
		return
	}
	affected, _ := res.RowsAffected()
	if affected == 0 {
		writeErr(r, http.StatusNotFound, "field not found", nil)
		return
	}

	e.Mu.Lock()
	for i := range *e.Collections {
		if (*e.Collections)[i].Name == colName {
			fields := (*e.Collections)[i].Fields
			for j := range fields {
				if fields[j].Name == fieldName {
					(*e.Collections)[i].Fields = append(fields[:j], fields[j+1:]...)
					break
				}
			}
			c = (*e.Collections)[i]
			break
		}
	}
	e.Mu.Unlock()

	r.Response.WriteJsonExit(g.Map{"data": CollectionMeta(c)})
}
