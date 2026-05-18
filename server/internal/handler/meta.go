package handler

import (
	"net/http"

	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/net/ghttp"

	"ksogit.kingsoft.net/wpsee/itabbase/server/internal/model"
)

// HandleWhoami returns the current authenticated user with full profile.
func (e *Env) HandleWhoami(r *ghttp.Request) {
	if u, ok := model.UserFromCtx(r.Context()); ok {
		roles, _, _ := model.RolesFromCtx(r.Context())
		data := g.Map{"id": u.ID, "display_name": u.Name, "roles": roles}

		if u.LocalID > 0 {
			row, err := e.DB.Model(model.BuiltinUsers).Ctx(r.Context()).
				Where("id", u.LocalID).One()
			if err == nil && !row.IsEmpty() {
				directFields := []string{
					"external_id", "avatar", "email", "phone",
					"gender", "employee_id", "title", "department",
					"department_path", "company_id",
				}
				for _, f := range directFields {
					if v := row[f].String(); v != "" {
						data[f] = v
					}
				}
				if v := row["login_name"].String(); v != "" {
					data["user_name"] = v
				}
			}
		}

		r.Response.WriteJsonExit(g.Map{"data": data})
		return
	}
	if e.ACLDisabled {
		r.Response.WriteJsonExit(g.Map{
			"data": g.Map{"id": "anonymous", "display_name": "Anonymous (auth disabled)", "roles": []string{"admin"}},
		})
		return
	}
	writeErr(r, http.StatusUnauthorized, "unauthenticated", nil)
}

// HandleMetaCollections describes every registered collection.
func (e *Env) HandleMetaCollections(r *ghttp.Request) {
	out := make([]g.Map, 0, len(*e.Collections))
	for _, c := range *e.Collections {
		out = append(out, CollectionMeta(c))
	}
	r.Response.WriteJsonExit(g.Map{"data": out})
}

// CollectionMeta converts a Collection to its JSON meta representation.
func CollectionMeta(c model.Collection) g.Map {
	fields := make([]g.Map, 0, len(c.Fields))
	for _, f := range c.Fields {
		fm := g.Map{
			"name": f.Name,
			"type": string(f.Type),
		}
		if f.Required {
			fm["required"] = true
		}
		if f.Default != nil {
			fm["default"] = f.Default
		}
		if f.MaxLen > 0 {
			fm["max_len"] = f.MaxLen
		}
		if f.Target != "" {
			fm["target"] = f.Target
		}
		if f.Through != "" {
			fm["through"] = f.Through
		}
		fields = append(fields, fm)
	}
	source := c.Source
	if source == "" {
		source = model.SourceCode
	}
	out := g.Map{
		"name":     c.Name,
		"display":  c.Display,
		"fields":   fields,
		"source":   source,
		"internal": c.Internal,
	}
	if c.ACL != nil {
		out["acl"] = c.ACL
	}
	return out
}
