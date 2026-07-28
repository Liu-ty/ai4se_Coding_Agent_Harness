//go:build windows

package credentials

import (
	"testing"
	"unsafe"

	"golang.org/x/sys/windows"
)

func TestOwnerOnlyACLGrantsOnlySuppliedSID(t *testing.T) {
	owner, err := windows.StringToSid("S-1-5-18")
	if err != nil {
		t.Fatal(err)
	}
	acl, err := ownerOnlyACL(owner)
	if err != nil {
		t.Fatal(err)
	}
	if acl.AceCount != 1 {
		t.Fatalf("ACL entry count = %d, want 1", acl.AceCount)
	}
	var ace *windows.ACCESS_ALLOWED_ACE
	if err := windows.GetAce(acl, 0, &ace); err != nil {
		t.Fatal(err)
	}
	grantee := (*windows.SID)(unsafe.Pointer(&ace.SidStart))
	if !grantee.Equals(owner) {
		t.Fatalf("ACL grantee = %s, want supplied owner %s", grantee.String(), owner.String())
	}
}
