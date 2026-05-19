package model

import "github.com/gogf/gf/v2/net/ghttp"

// Route declares a non-CRUD endpoint registered under the kernel's apiPrefix.
type Route struct {
	Method  string
	Path    string
	Handler ghttp.HandlerFunc
	ACL     RouteACL

	Collection string // if set, the route is bound to this collection for owner-scope checks
	IDParam    string // URL param name that holds the record ID (used with Collection)
}

// RouteACL controls access to a custom route.
type RouteACL struct {
	Roles     []string
	Anonymous bool
}

// RequireAuthed permits any authenticated user (any role).
func RequireAuthed() RouteACL { return RouteACL{} }

// RequireRole permits only authenticated users that hold any of the listed roles.
func RequireRole(roles ...string) RouteACL { return RouteACL{Roles: roles} }

// AllowAnonymous permits unauthenticated requests.
func AllowAnonymous() RouteACL { return RouteACL{Anonymous: true} }
