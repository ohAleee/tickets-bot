package tickets

import (
	"testing"

	"github.com/TicketsBot-cloud/gdl/objects/channel"
	discordpermission "github.com/TicketsBot-cloud/gdl/permission"
)

func TestApplyLock(t *testing.T) {
	view := discordpermission.BuildPermissions(discordpermission.ViewChannel)
	overwrites := []channel.PermissionOverwrite{
		{Id: 1, Type: channel.PermissionTypeMember, Allow: view | sendMessages}, // opener
		{Id: 2, Type: channel.PermissionTypeMember, Allow: view | sendMessages}, // staff
		{Id: 3, Type: channel.PermissionTypeRole, Allow: view | sendMessages},   // role, id collides with a target
	}

	targets := map[uint64]bool{1: true, 3: true}

	locked := applyLock(overwrites, targets, true)
	if locked[0].Allow&sendMessages != 0 || locked[0].Deny&sendMessages == 0 {
		t.Errorf("opener should be denied SendMessages, got allow=%d deny=%d", locked[0].Allow, locked[0].Deny)
	}
	if locked[0].Allow&view == 0 {
		t.Error("opener should keep ViewChannel")
	}
	if locked[1].Allow&sendMessages == 0 || locked[1].Deny != 0 {
		t.Error("non-target member overwrite should be untouched")
	}
	if locked[2].Allow&sendMessages == 0 || locked[2].Deny != 0 {
		t.Error("role overwrite should be untouched")
	}
	if overwrites[0].Deny != 0 {
		t.Error("applyLock must not mutate its input")
	}

	unlocked := applyLock(locked, targets, false)
	if unlocked[0].Allow&sendMessages == 0 || unlocked[0].Deny&sendMessages != 0 {
		t.Errorf("opener should be re-allowed SendMessages, got allow=%d deny=%d", unlocked[0].Allow, unlocked[0].Deny)
	}
}
