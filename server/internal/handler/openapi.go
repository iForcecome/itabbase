package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/gogf/gf/v2/net/ghttp"

	"ksogit.kingsoft.net/wpsee/itabbase/server/internal/model"
)

// OpenAPIOptions carries context that Env doesn't hold but the spec needs.
type OpenAPIOptions struct {
	Title        string
	Version      string
	APIPrefix    string
	BuiltinAuth  bool
	SSOEnabled   bool
	CustomRoutes []model.Route
}

// HandleOpenAPISpec returns a handler that generates an OpenAPI 3.0 JSON spec
// from the registered collections and custom routes at request time.
func (e *Env) HandleOpenAPISpec(opts OpenAPIOptions) ghttp.HandlerFunc {
	if opts.Title == "" {
		opts.Title = "ITabBase API"
	}
	if opts.Version == "" {
		opts.Version = "1.0.0"
	}
	return func(r *ghttp.Request) {
		spec := e.buildSpec(opts)
		b, err := json.MarshalIndent(spec, "", "  ")
		if err != nil {
			r.Response.Status = http.StatusInternalServerError
			return
		}
		r.Response.Header().Set("Content-Type", "application/json")
		r.Response.Write(b)
	}
}

// ---- internal spec types ----

type oaSpec struct {
	OpenAPI    string            `json:"openapi"`
	Info       oaInfo            `json:"info"`
	Paths      map[string]oaPath `json:"paths"`
	Components oaComponents      `json:"components"`
	Tags       []oaTag           `json:"tags,omitempty"`
}

type oaInfo struct {
	Title   string `json:"title"`
	Version string `json:"version"`
}

type oaTag struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
}

type oaPath struct {
	Get    *oaOp `json:"get,omitempty"`
	Post   *oaOp `json:"post,omitempty"`
	Patch  *oaOp `json:"patch,omitempty"`
	Delete *oaOp `json:"delete,omitempty"`
}

type oaOp struct {
	Tags        []string              `json:"tags,omitempty"`
	Summary     string                `json:"summary"`
	OperationID string                `json:"operationId"`
	Parameters  []oaParam             `json:"parameters,omitempty"`
	RequestBody *oaRequestBody        `json:"requestBody,omitempty"`
	Responses   map[string]oaResponse `json:"responses"`
	Security    []map[string][]string `json:"security,omitempty"`
}

type oaParam struct {
	Name        string   `json:"name"`
	In          string   `json:"in"`
	Required    bool     `json:"required,omitempty"`
	Description string   `json:"description,omitempty"`
	Schema      oaSchema `json:"schema"`
}

type oaRequestBody struct {
	Required bool                   `json:"required"`
	Content  map[string]oaMediaType `json:"content"`
}

type oaMediaType struct {
	Schema oaSchema `json:"schema"`
}

type oaResponse struct {
	Description string                 `json:"description"`
	Content     map[string]oaMediaType `json:"content,omitempty"`
}

type oaSchema struct {
	Type        string              `json:"type,omitempty"`
	Format      string              `json:"format,omitempty"`
	Ref         string              `json:"$ref,omitempty"`
	Properties  map[string]oaSchema `json:"properties,omitempty"`
	Required    []string            `json:"required,omitempty"`
	Items       *oaSchema           `json:"items,omitempty"`
	Description string              `json:"description,omitempty"`
}

type oaComponents struct {
	Schemas         map[string]oaSchema         `json:"schemas"`
	SecuritySchemes map[string]oaSecurityScheme `json:"securitySchemes"`
}

type oaSecurityScheme struct {
	Type string `json:"type"`
	In   string `json:"in,omitempty"`
	Name string `json:"name,omitempty"`
}

// ---- builder ----

func (e *Env) buildSpec(opts OpenAPIOptions) oaSpec {
	paths := map[string]oaPath{}
	schemas := map[string]oaSchema{}
	var tags []oaTag

	// Fixed schema: generic error response.
	schemas["Error"] = oaSchema{
		Type: "object",
		Properties: map[string]oaSchema{
			"error":  {Type: "string"},
			"detail": {Type: "string"},
		},
	}

	errResp := oaResponse{
		Description: "Error",
		Content: map[string]oaMediaType{
			"application/json": {Schema: oaSchema{Ref: "#/components/schemas/Error"}},
		},
	}

	// --- auth endpoints ---
	authTag := "auth"
	tags = append(tags, oaTag{Name: authTag, Description: "认证"})
	if opts.BuiltinAuth {
		p := opts.APIPrefix + "/auth/local/login"
		paths[p] = oaPath{
			Post: &oaOp{
				Tags:        []string{authTag},
				Summary:     "账号密码登录",
				OperationID: "authLocalLogin",
				RequestBody: &oaRequestBody{
					Required: true,
					Content: map[string]oaMediaType{
						"application/json": {Schema: oaSchema{
							Type: "object",
							Properties: map[string]oaSchema{
								"username": {Type: "string"},
								"password": {Type: "string", Format: "password"},
							},
							Required: []string{"username", "password"},
						}},
					},
				},
				Responses: map[string]oaResponse{
					"200": {Description: "登录成功"},
					"401": errResp,
				},
				Security: []map[string][]string{},
			},
		}
		p2 := opts.APIPrefix + "/auth/logout"
		paths[p2] = oaPath{
			Post: &oaOp{
				Tags:        []string{authTag},
				Summary:     "登出",
				OperationID: "authLogout",
				Responses:   map[string]oaResponse{"200": {Description: "登出成功"}},
			},
		}
	}
	if opts.SSOEnabled {
		p := opts.APIPrefix + "/auth/login"
		paths[p] = oaPath{
			Get: &oaOp{
				Tags:        []string{authTag},
				Summary:     "SSO 跳转登录",
				OperationID: "authSSOLogin",
				Responses:   map[string]oaResponse{"302": {Description: "重定向至 SSO 提供商"}},
				Security:    []map[string][]string{},
			},
		}
		p2 := opts.APIPrefix + "/auth/callback"
		paths[p2] = oaPath{
			Get: &oaOp{
				Tags:        []string{authTag},
				Summary:     "SSO 回调",
				OperationID: "authSSOCallback",
				Responses:   map[string]oaResponse{"302": {Description: "重定向至管理后台"}},
				Security:    []map[string][]string{},
			},
		}
	}

	// --- meta endpoints ---
	metaTag := "meta"
	tags = append(tags, oaTag{Name: metaTag, Description: "元信息"})
	{
		p := opts.APIPrefix + "/meta/whoami"
		paths[p] = oaPath{
			Get: &oaOp{
				Tags:        []string{metaTag},
				Summary:     "当前登录用户信息",
				OperationID: "metaWhoami",
				Responses:   map[string]oaResponse{"200": {Description: "用户信息"}, "401": errResp},
				Security:    cookieSecurity(e.ACLDisabled),
			},
		}
		p2 := opts.APIPrefix + "/meta/collections"
		paths[p2] = oaPath{
			Get: &oaOp{
				Tags:        []string{metaTag},
				Summary:     "已注册 Collection 列表",
				OperationID: "metaListCollections",
				Responses:   map[string]oaResponse{"200": {Description: "Collection 列表"}, "401": errResp},
				Security:    cookieSecurity(e.ACLDisabled),
			},
			Post: &oaOp{
				Tags:        []string{metaTag},
				Summary:     "动态创建 Collection (admin)",
				OperationID: "metaCreateCollection",
				Responses:   map[string]oaResponse{"200": {Description: "创建成功"}, "403": errResp},
				Security:    cookieSecurity(e.ACLDisabled),
			},
		}
		p3 := opts.APIPrefix + "/meta/collections/{name}"
		paths[p3] = oaPath{
			Patch: &oaOp{
				Tags:        []string{metaTag},
				Summary:     "更新 Collection (admin)",
				OperationID: "metaUpdateCollection",
				Parameters:  []oaParam{pathParam("name", "Collection 名称")},
				Responses:   map[string]oaResponse{"200": {Description: "更新成功"}, "403": errResp},
				Security:    cookieSecurity(e.ACLDisabled),
			},
			Delete: &oaOp{
				Tags:        []string{metaTag},
				Summary:     "删除 Collection (admin)",
				OperationID: "metaDeleteCollection",
				Parameters:  []oaParam{pathParam("name", "Collection 名称")},
				Responses:   map[string]oaResponse{"200": {Description: "删除成功"}, "403": errResp},
				Security:    cookieSecurity(e.ACLDisabled),
			},
		}
		p4 := opts.APIPrefix + "/meta/collections/{name}/fields"
		paths[p4] = oaPath{
			Post: &oaOp{
				Tags:        []string{metaTag},
				Summary:     "添加字段 (admin)",
				OperationID: "metaAddField",
				Parameters:  []oaParam{pathParam("name", "Collection 名称")},
				Responses:   map[string]oaResponse{"200": {Description: "添加成功"}, "403": errResp},
				Security:    cookieSecurity(e.ACLDisabled),
			},
		}
		p5 := opts.APIPrefix + "/meta/collections/{name}/fields/{fieldName}"
		paths[p5] = oaPath{
			Patch: &oaOp{
				Tags:        []string{metaTag},
				Summary:     "更新字段 (admin)",
				OperationID: "metaUpdateField",
				Parameters:  []oaParam{pathParam("name", "Collection 名称"), pathParam("fieldName", "字段名")},
				Responses:   map[string]oaResponse{"200": {Description: "更新成功"}, "403": errResp},
				Security:    cookieSecurity(e.ACLDisabled),
			},
			Delete: &oaOp{
				Tags:        []string{metaTag},
				Summary:     "删除字段 (admin)",
				OperationID: "metaDeleteField",
				Parameters:  []oaParam{pathParam("name", "Collection 名称"), pathParam("fieldName", "字段名")},
				Responses:   map[string]oaResponse{"200": {Description: "删除成功"}, "403": errResp},
				Security:    cookieSecurity(e.ACLDisabled),
			},
		}
	}

	// --- collection CRUD endpoints ---
	for _, c := range *e.Collections {
		if c.Internal {
			continue
		}
		schemaName := collectionSchemaName(c.Name)
		schemas[schemaName] = buildCollectionSchema(c)
		schemas[schemaName+"Input"] = buildInputSchema(c)
		tags = append(tags, oaTag{Name: c.Name, Description: displayName(c)})

		listPath := fmt.Sprintf("%s/%s", opts.APIPrefix, c.Name)
		itemPath := fmt.Sprintf("%s/%s/{id}", opts.APIPrefix, c.Name)

		listSecurity := collectionSecurity(c, model.ActionList, e.ACLDisabled)
		createSecurity := collectionSecurity(c, model.ActionCreate, e.ACLDisabled)
		getSecurity := collectionSecurity(c, model.ActionGet, e.ACLDisabled)
		updateSecurity := collectionSecurity(c, model.ActionUpdate, e.ACLDisabled)
		deleteSecurity := collectionSecurity(c, model.ActionDelete, e.ACLDisabled)

		paths[listPath] = oaPath{
			Get: &oaOp{
				Tags:        []string{c.Name},
				Summary:     fmt.Sprintf("获取 %s 列表", displayName(c)),
				OperationID: c.Name + "List",
				Parameters:  listQueryParams(c),
				Responses: map[string]oaResponse{
					"200": listResponse(schemaName),
					"401": errResp,
				},
				Security: listSecurity,
			},
			Post: &oaOp{
				Tags:        []string{c.Name},
				Summary:     fmt.Sprintf("创建 %s", displayName(c)),
				OperationID: c.Name + "Create",
				RequestBody: inputRequestBody(schemaName),
				Responses: map[string]oaResponse{
					"201": singleResponse(schemaName),
					"400": errResp,
					"401": errResp,
				},
				Security: createSecurity,
			},
		}
		paths[itemPath] = oaPath{
			Get: &oaOp{
				Tags:        []string{c.Name},
				Summary:     fmt.Sprintf("获取单条 %s", displayName(c)),
				OperationID: c.Name + "Get",
				Parameters: []oaParam{
					idParam(),
					{Name: "include", In: "query", Schema: oaSchema{Type: "string"}, Description: "逗号分隔的关联字段名"},
				},
				Responses: map[string]oaResponse{
					"200": singleResponse(schemaName),
					"404": errResp,
				},
				Security: getSecurity,
			},
			Patch: &oaOp{
				Tags:        []string{c.Name},
				Summary:     fmt.Sprintf("更新 %s", displayName(c)),
				OperationID: c.Name + "Update",
				Parameters:  []oaParam{idParam()},
				RequestBody: inputRequestBody(schemaName),
				Responses: map[string]oaResponse{
					"200": singleResponse(schemaName),
					"400": errResp,
					"404": errResp,
				},
				Security: updateSecurity,
			},
			Delete: &oaOp{
				Tags:        []string{c.Name},
				Summary:     fmt.Sprintf("删除 %s", displayName(c)),
				OperationID: c.Name + "Delete",
				Parameters:  []oaParam{idParam()},
				Responses: map[string]oaResponse{
					"200": {Description: "删除成功"},
					"404": errResp,
				},
				Security: deleteSecurity,
			},
		}
	}

	// --- custom routes ---
	if len(opts.CustomRoutes) > 0 {
		customTag := "custom"
		tags = append(tags, oaTag{Name: customTag, Description: "业务自定义接口"})
		for _, route := range opts.CustomRoutes {
			p := opts.APIPrefix + route.Path
			// Convert :param to {param} for OpenAPI path syntax.
			p = colonParamsToOpenAPI(p)

			params := extractPathParams(p)
			sec := routeSecurity(route.ACL, e.ACLDisabled)
			op := &oaOp{
				Tags:        []string{customTag},
				Summary:     fmt.Sprintf("%s %s", route.Method, route.Path),
				OperationID: routeOperationID(route),
				Parameters:  params,
				Responses: map[string]oaResponse{
					"200": {Description: "成功"},
					"400": errResp,
					"401": errResp,
				},
				Security: sec,
			}

			existing := paths[p]
			switch strings.ToUpper(route.Method) {
			case "GET":
				existing.Get = op
			case "POST":
				existing.Post = op
			case "PATCH":
				existing.Patch = op
			case "DELETE":
				existing.Delete = op
			}
			paths[p] = existing
		}
	}

	securitySchemes := map[string]oaSecurityScheme{}
	if !e.ACLDisabled {
		securitySchemes["cookieAuth"] = oaSecurityScheme{
			Type: "apiKey",
			In:   "cookie",
			Name: model.DefaultSessionCookieName,
		}
	}

	return oaSpec{
		OpenAPI: "3.0.3",
		Info:    oaInfo{Title: opts.Title, Version: opts.Version},
		Paths:   paths,
		Tags:    tags,
		Components: oaComponents{
			Schemas:         schemas,
			SecuritySchemes: securitySchemes,
		},
	}
}

// ---- schema builders ----

func buildCollectionSchema(c model.Collection) oaSchema {
	props := map[string]oaSchema{
		"id":         {Type: "integer", Format: "int64"},
		"created_at": {Type: "string", Format: "date-time"},
		"updated_at": {Type: "string", Format: "date-time"},
	}
	for _, f := range c.Fields {
		if f.Type == model.THasMany {
			props[f.Name] = oaSchema{Type: "array", Items: &oaSchema{Type: "object"}}
			continue
		}
		props[f.Name] = fieldSchema(f)
	}
	return oaSchema{Type: "object", Properties: props}
}

func buildInputSchema(c model.Collection) oaSchema {
	props := map[string]oaSchema{}
	var required []string
	for _, f := range c.Fields {
		if f.IsVirtual() {
			continue
		}
		props[f.Name] = fieldSchema(f)
		if f.Required && f.Default == nil {
			required = append(required, f.Name)
		}
	}
	s := oaSchema{Type: "object", Properties: props}
	if len(required) > 0 {
		s.Required = required
	}
	return s
}

func fieldSchema(f model.Field) oaSchema {
	switch f.Type {
	case model.TInt, model.TBelongsTo:
		return oaSchema{Type: "integer", Format: "int64"}
	case model.TFloat:
		return oaSchema{Type: "number", Format: "float"}
	case model.TBool:
		return oaSchema{Type: "boolean"}
	case model.TDateTime:
		return oaSchema{Type: "string", Format: "date-time"}
	default:
		return oaSchema{Type: "string"}
	}
}

// ---- response helpers ----

func listResponse(schemaName string) oaResponse {
	return oaResponse{
		Description: "成功",
		Content: map[string]oaMediaType{
			"application/json": {Schema: oaSchema{
				Type: "object",
				Properties: map[string]oaSchema{
					"data":  {Type: "array", Items: &oaSchema{Ref: "#/components/schemas/" + schemaName}},
					"total": {Type: "integer"},
					"page":  {Type: "integer"},
					"size":  {Type: "integer"},
				},
			}},
		},
	}
}

func singleResponse(schemaName string) oaResponse {
	return oaResponse{
		Description: "成功",
		Content: map[string]oaMediaType{
			"application/json": {Schema: oaSchema{
				Type: "object",
				Properties: map[string]oaSchema{
					"data": {Ref: "#/components/schemas/" + schemaName},
				},
			}},
		},
	}
}

func inputRequestBody(schemaName string) *oaRequestBody {
	return &oaRequestBody{
		Required: true,
		Content: map[string]oaMediaType{
			"application/json": {Schema: oaSchema{Ref: "#/components/schemas/" + schemaName + "Input"}},
		},
	}
}

// ---- param helpers ----

func idParam() oaParam {
	return oaParam{Name: "id", In: "path", Required: true, Schema: oaSchema{Type: "integer"}}
}

func pathParam(name, desc string) oaParam {
	return oaParam{Name: name, In: "path", Required: true, Description: desc, Schema: oaSchema{Type: "string"}}
}

func listQueryParams(c model.Collection) []oaParam {
	params := []oaParam{
		{Name: "page", In: "query", Schema: oaSchema{Type: "integer"}, Description: "页码，从 1 开始"},
		{Name: "size", In: "query", Schema: oaSchema{Type: "integer"}, Description: "每页条数，最大 100"},
		{Name: "sort", In: "query", Schema: oaSchema{Type: "string"}, Description: "排序字段，前缀 - 表示降序，如 -created_at"},
		{Name: "include", In: "query", Schema: oaSchema{Type: "string"}, Description: "逗号分隔的关联字段名，如 include=author,tags"},
	}
	for _, f := range c.Fields {
		if f.IsVirtual() || f.Type == model.TBelongsTo {
			continue
		}
		params = append(params, oaParam{
			Name:        fmt.Sprintf("filter[%s]", f.Name),
			In:          "query",
			Description: fmt.Sprintf("过滤 %s 字段", f.Name),
			Schema:      fieldSchema(f),
		})
	}
	return params
}

// ---- security helpers ----

func cookieSecurity(aclDisabled bool) []map[string][]string {
	if aclDisabled {
		return []map[string][]string{}
	}
	return []map[string][]string{{"cookieAuth": {}}}
}

func collectionSecurity(c model.Collection, action string, aclDisabled bool) []map[string][]string {
	if aclDisabled {
		return []map[string][]string{}
	}
	if c.ACL == nil {
		return []map[string][]string{{"cookieAuth": {}}}
	}
	if c.ACL.Allows([]string{model.RoleAnonymous}, action) {
		return []map[string][]string{}
	}
	return []map[string][]string{{"cookieAuth": {}}}
}

func routeSecurity(acl model.RouteACL, aclDisabled bool) []map[string][]string {
	if aclDisabled || acl.Anonymous {
		return []map[string][]string{}
	}
	return []map[string][]string{{"cookieAuth": {}}}
}

// ---- misc helpers ----

func collectionSchemaName(name string) string {
	if len(name) == 0 {
		return name
	}
	return strings.ToUpper(name[:1]) + name[1:]
}

func displayName(c model.Collection) string {
	if c.Display != "" {
		return c.Display
	}
	return c.Name
}

func colonParamsToOpenAPI(path string) string {
	parts := strings.Split(path, "/")
	for i, p := range parts {
		if strings.HasPrefix(p, ":") {
			parts[i] = "{" + p[1:] + "}"
		}
	}
	return strings.Join(parts, "/")
}

func extractPathParams(path string) []oaParam {
	var params []oaParam
	for _, seg := range strings.Split(path, "/") {
		if strings.HasPrefix(seg, "{") && strings.HasSuffix(seg, "}") {
			name := seg[1 : len(seg)-1]
			schema := oaSchema{Type: "string"}
			if name == "id" {
				schema = oaSchema{Type: "integer"}
			}
			params = append(params, oaParam{Name: name, In: "path", Required: true, Schema: schema})
		}
	}
	return params
}

func routeOperationID(r model.Route) string {
	p := strings.NewReplacer("/", "_", ":", "", "{", "", "}", "").Replace(r.Path)
	p = strings.Trim(p, "_")
	method := strings.ToLower(r.Method)
	return method + "_" + p
}
