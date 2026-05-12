package itab

import (
	"net/http"

	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/net/ghttp"
)

// handleWhoami returns the current authenticated user, or an anonymous shape
// when the kernel runs in WithoutAuth mode.
func (k *Kernel) handleWhoami(r *ghttp.Request) {
	if u, ok := UserFromCtx(r.Context()); ok {
		r.Response.WriteJsonExit(g.Map{
			"data": g.Map{"id": u.ID, "name": u.Name},
		})
		return
	}
	if k.aclDisabled {
		r.Response.WriteJsonExit(g.Map{
			"data": g.Map{"id": "anonymous", "name": "Anonymous (auth disabled)"},
		})
		return
	}
	writeErr(r, http.StatusUnauthorized, "unauthenticated", nil)
}

// handleMetaCollections describes every registered collection so the admin
// SPA / TS client generator can render UIs / generate types without hardcoding.
func (k *Kernel) handleMetaCollections(r *ghttp.Request) {
	out := make([]g.Map, 0, len(k.collections))
	for _, c := range k.collections {
		out = append(out, collectionMeta(c))
	}
	r.Response.WriteJsonExit(g.Map{"data": out})
}

func collectionMeta(c Collection) g.Map {
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
	out := g.Map{
		"name":    c.Name,
		"display": c.Display,
		"fields":  fields,
	}
	if c.ACL != nil {
		out["acl"] = c.ACL
	}
	return out
}
