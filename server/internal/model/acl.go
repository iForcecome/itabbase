package model

const (
	ActionList   = "list"
	ActionGet    = "get"
	ActionCreate = "create"
	ActionUpdate = "update"
	ActionDelete = "delete"
	ActionAll    = "*"
)

// ACL maps a role name to the list of actions it is permitted to perform on a collection.
// Action "*" grants every action.
type ACL map[string][]string

func (a ACL) Allows(roles []string, action string) bool {
	for _, role := range roles {
		actions, ok := a[role]
		if !ok {
			continue
		}
		for _, ac := range actions {
			if ac == ActionAll || ac == action {
				return true
			}
		}
	}
	return false
}

// DecideAccess applies the ACL policy.
//
// Default policy when collection.ACL is nil:
//   - authed -> allow
//   - anonymous -> deny
//
// Explicit ACL overrides the default; only roles listed there get access.
func DecideAccess(c Collection, action string, roles []string, authed bool) bool {
	if c.ACL == nil {
		return authed
	}
	return c.ACL.Allows(roles, action)
}
