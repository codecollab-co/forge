package permissions

import "testing"

func TestAllow(t *testing.T) {
	owner := Actor{UserID: "u-owner"}
	stranger := Actor{UserID: "u-other"}
	anon := Actor{IsAnonymous: true}

	pub := Repo{OwnerID: "u-owner", Visibility: "public"}
	priv := Repo{OwnerID: "u-owner", Visibility: "private"}

	cases := []struct {
		name   string
		actor  Actor
		repo   Repo
		action Action
		want   bool
	}{
		// Reads
		{"anon reads public", anon, pub, ActionRead, true},
		{"anon reads private", anon, priv, ActionRead, false},
		{"stranger reads public", stranger, pub, ActionRead, true},
		{"stranger reads private", stranger, priv, ActionRead, false},
		{"owner reads public", owner, pub, ActionRead, true},
		{"owner reads private", owner, priv, ActionRead, true},
		// Pushes
		{"anon pushes public", anon, pub, ActionPush, false},
		{"anon pushes private", anon, priv, ActionPush, false},
		{"stranger pushes public", stranger, pub, ActionPush, false},
		{"stranger pushes private", stranger, priv, ActionPush, false},
		{"owner pushes public", owner, pub, ActionPush, true},
		{"owner pushes private", owner, priv, ActionPush, true},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := Allow(c.actor, c.repo, c.action); got != c.want {
				t.Fatalf("Allow(%+v, %+v, %d) = %v, want %v", c.actor, c.repo, c.action, got, c.want)
			}
		})
	}
}

func TestAllow_UnknownAction(t *testing.T) {
	if Allow(Actor{UserID: "u"}, Repo{OwnerID: "u", Visibility: "public"}, Action(99)) {
		t.Fatal("unknown action should deny")
	}
}
