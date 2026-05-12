package itab

import (
	"embed"
	"io/fs"
	"mime"
	"net/http"
	"path"
	"strings"

	"github.com/gogf/gf/v2/net/ghttp"
)

//go:embed all:web
var adminFS embed.FS

// distRoot is the directory inside adminFS that holds Vite's build output.
const distRoot = "web"

// serveAdminSPA serves the embedded admin SPA.
//
// Behavior:
//   - <prefix>/admin (no trailing slash) → 301 redirect to <prefix>/admin/.
//     Required so the browser resolves the index.html's relative asset URLs
//     (`./assets/foo.js`) against `/admin/` and not the parent directory.
//   - <prefix>/admin/<asset> → serve from embedded FS with proper Content-Type.
//   - any other path (e.g. SPA history routing) → fall back to index.html.
func (k *Kernel) serveAdminSPA(r *ghttp.Request) {
	// Use RequestURI (raw HTTP request line) instead of URL.Path because
	// GoFrame normalizes the latter and strips trailing slashes, making
	// /admin and /admin/ indistinguishable to the handler.
	rawPath := r.RequestURI
	if i := strings.IndexByte(rawPath, '?'); i >= 0 {
		rawPath = rawPath[:i]
	}
	if strings.HasSuffix(rawPath, "/admin") {
		r.Response.Header().Set("Location", rawPath+"/")
		r.Response.WriteHeader(http.StatusMovedPermanently)
		return
	}

	full := r.URL.Path
	idx := strings.LastIndex(full, "/admin/")
	rel := ""
	if idx >= 0 {
		rel = full[idx+len("/admin/"):]
	}
	if rel == "" || strings.HasSuffix(rel, "/") {
		rel = path.Join(rel, "index.html")
	}

	data, err := fs.ReadFile(adminFS, distRoot+"/"+rel)
	if err != nil {
		fallback, ferr := fs.ReadFile(adminFS, distRoot+"/index.html")
		if ferr != nil {
			r.Response.Status = http.StatusInternalServerError
			r.Response.Write("admin SPA not built (run `pnpm build` in itabbase/admin)")
			return
		}
		r.Response.Header().Set("Content-Type", "text/html; charset=utf-8")
		r.Response.Write(fallback)
		return
	}
	if ct := mime.TypeByExtension(path.Ext(rel)); ct != "" {
		r.Response.Header().Set("Content-Type", ct)
	}
	if strings.HasPrefix(rel, "assets/") {
		r.Response.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
	}
	r.Response.Write(data)
}
