package handler

import (
	"context"
	"fmt"
	"strings"

	"github.com/gogf/gf/v2/net/ghttp"

	"ksogit.kingsoft.net/wpsee/itabbase/server/internal/model"
)

func parseIncludes(r *ghttp.Request) []string {
	raw := r.GetQuery("include").String()
	if raw == "" {
		return nil
	}
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

func (e *Env) resolveIncludes(ctx context.Context, c model.Collection, rows []map[string]any, includes []string) error {
	if len(rows) == 0 {
		return nil
	}
	roles, authed, _ := model.RolesFromCtx(ctx)
	for _, inc := range includes {
		f, ok := findIncludeField(c, inc)
		if !ok {
			continue
		}
		target, hasTarget := e.FindCollection(f.Target)
		switch f.Type {
		case model.TBelongsTo:
			if hasTarget && !model.DecideAccess(target, model.ActionGet, roles, authed) {
				placeBelongsToEmpty(f, rows)
				continue
			}
			if err := e.includeBelongsTo(ctx, f, rows); err != nil {
				return err
			}
		case model.THasMany:
			if hasTarget && !model.DecideAccess(target, model.ActionList, roles, authed) {
				placeHasManyEmpty(f, rows)
				continue
			}
			if err := e.includeHasMany(ctx, f, rows); err != nil {
				return err
			}
		}
	}
	return nil
}

func placeBelongsToEmpty(f model.Field, rows []map[string]any) {
	key := belongsToEmbedKey(f)
	for _, row := range rows {
		row[key] = nil
	}
}

func placeHasManyEmpty(f model.Field, rows []map[string]any) {
	for _, row := range rows {
		row[f.Name] = []map[string]any{}
	}
}

func findIncludeField(c model.Collection, inc string) (model.Field, bool) {
	for _, f := range c.Fields {
		switch f.Type {
		case model.TBelongsTo:
			stripped := strings.TrimSuffix(f.Name, "_id")
			if inc == stripped || inc == f.Name {
				return f, true
			}
		case model.THasMany:
			if inc == f.Name {
				return f, true
			}
		}
	}
	return model.Field{}, false
}

func belongsToEmbedKey(f model.Field) string {
	stripped := strings.TrimSuffix(f.Name, "_id")
	if stripped != f.Name {
		return stripped
	}
	return f.Name + "_obj"
}

func (e *Env) includeBelongsTo(ctx context.Context, f model.Field, rows []map[string]any) error {
	ids := uniqueAnyValues(rows, f.Name)
	if len(ids) == 0 {
		return nil
	}
	targets, err := e.DB.Model(e.resolveTable(f.Target)).Ctx(ctx).WhereIn("id", ids).All()
	if err != nil {
		return fmt.Errorf("load belongs_to %s: %w", f.Name, err)
	}
	byID := map[string]map[string]any{}
	for _, t := range targets.List() {
		byID[fmt.Sprintf("%v", t["id"])] = t
	}
	embedKey := belongsToEmbedKey(f)
	for _, row := range rows {
		v, ok := row[f.Name]
		if !ok || v == nil {
			continue
		}
		row[embedKey] = byID[fmt.Sprintf("%v", v)]
	}
	return nil
}

func (e *Env) includeHasMany(ctx context.Context, f model.Field, rows []map[string]any) error {
	parentIDs := uniqueAnyValues(rows, "id")
	if len(parentIDs) == 0 {
		return nil
	}
	related, err := e.DB.Model(e.resolveTable(f.Target)).Ctx(ctx).WhereIn(f.Through, parentIDs).All()
	if err != nil {
		return fmt.Errorf("load has_many %s: %w", f.Name, err)
	}
	byParent := map[string][]map[string]any{}
	for _, rel := range related.List() {
		key := fmt.Sprintf("%v", rel[f.Through])
		byParent[key] = append(byParent[key], rel)
	}
	for _, row := range rows {
		key := fmt.Sprintf("%v", row["id"])
		list := byParent[key]
		if list == nil {
			list = []map[string]any{}
		}
		row[f.Name] = list
	}
	return nil
}

func uniqueAnyValues(rows []map[string]any, key string) []any {
	seen := map[string]struct{}{}
	out := []any{}
	for _, row := range rows {
		v, ok := row[key]
		if !ok || v == nil {
			continue
		}
		k := fmt.Sprintf("%v", v)
		if _, dup := seen[k]; dup {
			continue
		}
		seen[k] = struct{}{}
		out = append(out, v)
	}
	return out
}
