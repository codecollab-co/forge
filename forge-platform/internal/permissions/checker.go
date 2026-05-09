// Package permissions is the PermissionChecker deep module from ADR-0007.
//
// Pure logic over (Actor, Resource, Action). No I/O. The whole permission
// policy lives here so it can be reasoned about and table-tested without
// spinning up a database or HTTP server.
package permissions

type Action int

const (
	ActionRead Action = iota
	ActionPush
)

type Actor struct {
	UserID      string // empty when IsAnonymous
	IsAnonymous bool
}

type Repo struct {
	OwnerID    string
	Visibility string // "public" or "private"
}

func Allow(actor Actor, repo Repo, action Action) bool {
	switch action {
	case ActionRead:
		if repo.Visibility == "public" {
			return true
		}
		if actor.IsAnonymous {
			return false
		}
		return actor.UserID == repo.OwnerID
	case ActionPush:
		if actor.IsAnonymous {
			return false
		}
		return actor.UserID == repo.OwnerID
	default:
		return false
	}
}
