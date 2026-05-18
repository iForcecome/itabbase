package handler

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/net/ghttp"

	"ksogit.kingsoft.net/wpsee/itabbase/server/internal/model"
)

func writeErr(r *ghttp.Request, status int, msg string, cause error) {
	r.Response.Status = status
	payload := g.Map{"error": msg}
	if cause != nil {
		payload["detail"] = cause.Error()
	}
	r.Response.WriteJsonExit(payload)
}

// writeOpError writes the appropriate HTTP response if err is non-nil.
// Returns true when no error (caller should continue), false otherwise.
func writeOpError(r *ghttp.Request, err error, fallbackMsg string) bool {
	if err == nil {
		return true
	}
	var oe *model.OpError
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
func pickFields(c model.Collection, body map[string]any, isCreate bool) (map[string]any, error) {
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
