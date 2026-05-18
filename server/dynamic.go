package itab

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"

	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/net/ghttp"
)

// mu protects k.collections for concurrent reads during dynamic CRUD
// and writes during runtime collection creation.
var mu sync.RWMutex

// loadDynamicCollections reads _collections + _fields from the DB and
// appends the resulting Collection structs to k.collections.
func (k *Kernel) loadDynamicCollections(ctx context.Context) error {
	colRows, err := k.db.Model(BuiltinMetaCollections).Ctx(ctx).OrderAsc("sort").OrderAsc("id").All()
	if err != nil {
		return fmt.Errorf("load _collections: %w", err)
	}
	if colRows.IsEmpty() {
		return nil
	}

	fieldRows, err := k.db.Model(BuiltinMetaFields).Ctx(ctx).OrderAsc("sort").OrderAsc("id").All()
	if err != nil {
		return fmt.Errorf("load _fields: %w", err)
	}

	fieldsByCol := map[string][]Field{}
	for _, row := range fieldRows.List() {
		colName, _ := row["collection_name"].(string)
		f := fieldFromRow(row)
		fieldsByCol[colName] = append(fieldsByCol[colName], f)
	}

	for _, row := range colRows.List() {
		name, _ := row["name"].(string)
		display, _ := row["display"].(string)
		if _, exists := k.collectionByName(name); exists {
			g.Log().Warningf(ctx, "[itab] skip dynamic collection %q: name conflicts with existing", name)
			continue
		}
		fields := fieldsByCol[name]
		if len(fields) == 0 {
			continue
		}
		c := Collection{
			Name:    name,
			Display: display,
			Fields:  fields,
			Source:   SourceDynamic,
		}
		if err := c.Validate(); err != nil {
			g.Log().Warningf(ctx, "[itab] skip invalid dynamic collection %q: %v", name, err)
			continue
		}
		k.collections = append(k.collections, c)
	}
	return nil
}

func fieldFromRow(row map[string]any) Field {
	getString := func(key string) string {
		v, _ := row[key].(string)
		return v
	}
	getInt := func(key string) int {
		switch v := row[key].(type) {
		case int:
			return v
		case int64:
			return int(v)
		case float64:
			return int(v)
		}
		return 0
	}
	getBool := func(key string) bool {
		switch v := row[key].(type) {
		case bool:
			return v
		case int:
			return v != 0
		case int64:
			return v != 0
		case float64:
			return v != 0
		}
		return false
	}

	f := Field{
		Name:     getString("name"),
		Type:     FieldType(getString("type")),
		Required: getBool("required"),
		MaxLen:   getInt("max_len"),
		Target:   getString("target"),
		Through:  getString("through"),
	}
	if dv := getString("default_value"); dv != "" {
		var parsed any
		if json.Unmarshal([]byte(dv), &parsed) == nil {
			f.Default = parsed
		}
	}
	return f
}

// dynamicCRUD returns a handler that resolves the collection name from the URL
// parameter and delegates to the appropriate CRUD handler with ACL.
func (k *Kernel) dynamicCRUD(action string) ghttp.HandlerFunc {
	return func(r *ghttp.Request) {
		colName := r.GetRouter("_col").String()
		mu.RLock()
		c, ok := k.collectionByName(colName)
		mu.RUnlock()
		if !ok {
			writeErr(r, http.StatusNotFound, fmt.Sprintf("collection %q not found", colName), nil)
			return
		}
		var inner ghttp.HandlerFunc
		switch action {
		case ActionList:
			inner = k.handleList(c)
		case ActionGet:
			inner = k.handleGet(c)
		case ActionCreate:
			inner = k.handleCreate(c)
		case ActionUpdate:
			inner = k.handleUpdate(c)
		case ActionDelete:
			inner = k.handleDelete(c)
		default:
			writeErr(r, http.StatusBadRequest, "unknown action", nil)
			return
		}
		k.aclWrap(c, action, inner)(r)
	}
}

// ---------- Collection management API ----------

func (k *Kernel) handleCreateCollection(r *ghttp.Request) {
	ctx := r.Context()
	body, err := readJSONBody(r)
	if err != nil {
		writeErr(r, http.StatusBadRequest, "invalid json body", err)
		return
	}
	name, _ := body["name"].(string)
	display, _ := body["display"].(string)
	fieldsRaw, _ := body["fields"].([]any)

	if name == "" {
		writeErr(r, http.StatusBadRequest, "name is required", nil)
		return
	}
	if !identRe.MatchString(name) {
		writeErr(r, http.StatusBadRequest, fmt.Sprintf("name %q must match %s", name, identRe.String()), nil)
		return
	}
	for _, rp := range k.reservedPaths {
		if name == rp {
			writeErr(r, http.StatusConflict, fmt.Sprintf("name %q is reserved", name), nil)
			return
		}
	}
	mu.RLock()
	_, exists := k.collectionByName(name)
	mu.RUnlock()
	if exists {
		writeErr(r, http.StatusConflict, fmt.Sprintf("collection %q already exists", name), nil)
		return
	}

	fields := parseFieldsFromBody(fieldsRaw)
	if len(fields) == 0 {
		writeErr(r, http.StatusBadRequest, "at least one field is required", nil)
		return
	}

	c := Collection{
		Name:    name,
		Display: display,
		Fields:  fields,
		Source:  SourceDynamic,
	}
	if err := c.Validate(); err != nil {
		writeErr(r, http.StatusBadRequest, err.Error(), nil)
		return
	}

	if _, err := k.db.Model(BuiltinMetaCollections).Ctx(ctx).Insert(g.Map{
		"name":    name,
		"display": display,
	}); err != nil {
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
		if _, err := k.db.Model(BuiltinMetaFields).Ctx(ctx).Insert(fm); err != nil {
			writeErr(r, http.StatusInternalServerError, "failed to save field metadata", err)
			return
		}
	}

	dialect := k.dialect()
	stmt := buildCreateTable(dialect, c)
	if _, err := k.db.Exec(ctx, stmt); err != nil {
		k.db.Model(BuiltinMetaFields).Ctx(ctx).Where("collection_name", name).Delete()
		k.db.Model(BuiltinMetaCollections).Ctx(ctx).Where("name", name).Delete()
		writeErr(r, http.StatusInternalServerError, "failed to create table", err)
		return
	}

	mu.Lock()
	k.collections = append(k.collections, c)
	mu.Unlock()

	r.Response.Status = http.StatusCreated
	r.Response.WriteJsonExit(g.Map{"data": collectionMeta(c)})
}

func (k *Kernel) handleUpdateCollection(r *ghttp.Request) {
	ctx := r.Context()
	name := r.GetRouter("name").String()

	mu.RLock()
	c, ok := k.collectionByName(name)
	mu.RUnlock()
	if !ok || c.Source != SourceDynamic {
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
	if len(patch) == 0 {
		writeErr(r, http.StatusBadRequest, "nothing to update", nil)
		return
	}

	if _, err := k.db.Model(BuiltinMetaCollections).Ctx(ctx).Where("name", name).Update(patch); err != nil {
		writeErr(r, http.StatusInternalServerError, "update failed", err)
		return
	}

	mu.Lock()
	for i := range k.collections {
		if k.collections[i].Name == name {
			if d, ok := patch["display"].(string); ok {
				k.collections[i].Display = d
			}
			c = k.collections[i]
			break
		}
	}
	mu.Unlock()

	r.Response.WriteJsonExit(g.Map{"data": collectionMeta(c)})
}

func (k *Kernel) handleDeleteCollection(r *ghttp.Request) {
	ctx := r.Context()
	name := r.GetRouter("name").String()

	mu.RLock()
	c, ok := k.collectionByName(name)
	mu.RUnlock()
	if !ok {
		writeErr(r, http.StatusNotFound, "collection not found", nil)
		return
	}
	if c.Source != SourceDynamic {
		writeErr(r, http.StatusForbidden, "cannot delete non-dynamic collection", nil)
		return
	}

	dialect := k.dialect()
	dropSQL := fmt.Sprintf("DROP TABLE IF EXISTS %s", quoteIdent(dialect, name))
	if _, err := k.db.Exec(ctx, dropSQL); err != nil {
		writeErr(r, http.StatusInternalServerError, "failed to drop table", err)
		return
	}

	k.db.Model(BuiltinMetaFields).Ctx(ctx).Where("collection_name", name).Delete()
	k.db.Model(BuiltinMetaCollections).Ctx(ctx).Where("name", name).Delete()

	mu.Lock()
	for i := range k.collections {
		if k.collections[i].Name == name {
			k.collections = append(k.collections[:i], k.collections[i+1:]...)
			break
		}
	}
	mu.Unlock()

	r.Response.WriteJsonExit(g.Map{"data": g.Map{"name": name}})
}

func (k *Kernel) handleAddField(r *ghttp.Request) {
	ctx := r.Context()
	colName := r.GetRouter("name").String()

	mu.RLock()
	c, ok := k.collectionByName(colName)
	mu.RUnlock()
	if !ok || c.Source != SourceDynamic {
		writeErr(r, http.StatusNotFound, "dynamic collection not found", nil)
		return
	}

	body, err := readJSONBody(r)
	if err != nil {
		writeErr(r, http.StatusBadRequest, "invalid json body", err)
		return
	}

	f := parseOneField(body)
	if f.Name == "" || f.Type == "" {
		writeErr(r, http.StatusBadRequest, "name and type are required", nil)
		return
	}
	if !identRe.MatchString(f.Name) {
		writeErr(r, http.StatusBadRequest, fmt.Sprintf("field name %q invalid", f.Name), nil)
		return
	}
	if !knownType(f.Type) {
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

	if _, err := k.db.Model(BuiltinMetaFields).Ctx(ctx).Insert(fm); err != nil {
		writeErr(r, http.StatusInternalServerError, "failed to save field metadata", err)
		return
	}

	if !f.IsVirtual() {
		dialect := k.dialect()
		stmt := buildAddColumn(dialect, colName, f)
		if _, err := k.db.Exec(ctx, stmt); err != nil {
			k.db.Model(BuiltinMetaFields).Ctx(ctx).
				Where("collection_name", colName).
				Where("name", f.Name).
				Delete()
			writeErr(r, http.StatusInternalServerError, "failed to add column", err)
			return
		}
	}

	mu.Lock()
	for i := range k.collections {
		if k.collections[i].Name == colName {
			k.collections[i].Fields = append(k.collections[i].Fields, f)
			c = k.collections[i]
			break
		}
	}
	mu.Unlock()

	r.Response.Status = http.StatusCreated
	r.Response.WriteJsonExit(g.Map{"data": collectionMeta(c)})
}

func (k *Kernel) handleUpdateField(r *ghttp.Request) {
	ctx := r.Context()
	colName := r.GetRouter("name").String()
	fieldName := r.GetRouter("fieldName").String()

	mu.RLock()
	c, ok := k.collectionByName(colName)
	mu.RUnlock()
	if !ok || c.Source != SourceDynamic {
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

	res, err := k.db.Model(BuiltinMetaFields).Ctx(ctx).
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
		mu.Lock()
		for i := range k.collections {
			if k.collections[i].Name == colName {
				for j := range k.collections[i].Fields {
					if k.collections[i].Fields[j].Name == fieldName {
						k.collections[i].Fields[j].Required = req
					}
				}
				c = k.collections[i]
				break
			}
		}
		mu.Unlock()
	}

	r.Response.WriteJsonExit(g.Map{"data": collectionMeta(c)})
}

func (k *Kernel) handleDeleteField(r *ghttp.Request) {
	ctx := r.Context()
	colName := r.GetRouter("name").String()
	fieldName := r.GetRouter("fieldName").String()

	mu.RLock()
	c, ok := k.collectionByName(colName)
	mu.RUnlock()
	if !ok || c.Source != SourceDynamic {
		writeErr(r, http.StatusNotFound, "dynamic collection not found", nil)
		return
	}

	if fieldName == "id" {
		writeErr(r, http.StatusForbidden, "cannot delete id field", nil)
		return
	}

	res, err := k.db.Model(BuiltinMetaFields).Ctx(ctx).
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

	mu.Lock()
	for i := range k.collections {
		if k.collections[i].Name == colName {
			fields := k.collections[i].Fields
			for j := range fields {
				if fields[j].Name == fieldName {
					k.collections[i].Fields = append(fields[:j], fields[j+1:]...)
					break
				}
			}
			c = k.collections[i]
			break
		}
	}
	mu.Unlock()

	r.Response.WriteJsonExit(g.Map{"data": collectionMeta(c)})
}

// ---------- helpers ----------

func parseFieldsFromBody(raw []any) []Field {
	fields := make([]Field, 0, len(raw))
	for _, item := range raw {
		m, ok := item.(map[string]any)
		if !ok {
			continue
		}
		f := parseOneField(m)
		if f.Name != "" && f.Type != "" {
			fields = append(fields, f)
		}
	}
	return fields
}

func parseOneField(m map[string]any) Field {
	getString := func(key string) string {
		v, _ := m[key].(string)
		return v
	}
	getInt := func(key string) int {
		switch v := m[key].(type) {
		case float64:
			return int(v)
		case int:
			return v
		case int64:
			return int(v)
		}
		return 0
	}
	getBool := func(key string) bool {
		v, _ := m[key].(bool)
		return v
	}

	f := Field{
		Name:     getString("name"),
		Type:     FieldType(getString("type")),
		Required: getBool("required"),
		MaxLen:   getInt("max_len"),
		Target:   getString("target"),
		Through:  getString("through"),
	}
	if dv := getString("default_value"); dv != "" {
		var parsed any
		if json.Unmarshal([]byte(dv), &parsed) == nil {
			f.Default = parsed
		}
	} else if dv, ok := m["default"]; ok {
		f.Default = dv
	}
	return f
}
