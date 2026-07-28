//go:build windows

package credentials_test

import (
	"testing"
	"unsafe"

	"golang.org/x/sys/windows"
)

func assertOwnerOnlyPermissions(t *testing.T, path string) {
	t.Helper()
	descriptor, err := windows.GetNamedSecurityInfo(
		path,
		windows.SE_FILE_OBJECT,
		windows.DACL_SECURITY_INFORMATION|windows.OWNER_SECURITY_INFORMATION,
	)
	if err != nil {
		t.Fatal(err)
	}
	control, _, err := descriptor.Control()
	if err != nil {
		t.Fatal(err)
	}
	if control&windows.SE_DACL_PROTECTED == 0 {
		t.Fatal("vault DACL is not protected from inherited permissions")
	}
	dacl, _, err := descriptor.DACL()
	if err != nil {
		t.Fatal(err)
	}
	if dacl == nil || dacl.AceCount != 1 {
		t.Fatalf("vault DACL entry count = %d", daclEntryCount(dacl))
	}
	var ace *windows.ACCESS_ALLOWED_ACE
	if err := windows.GetAce(dacl, 0, &ace); err != nil {
		t.Fatal(err)
	}
	owner, _, err := descriptor.Owner()
	if err != nil {
		t.Fatal(err)
	}
	allowed := (*windows.SID)(unsafe.Pointer(&ace.SidStart))
	if owner == nil || !allowed.Equals(owner) {
		t.Fatal("vault DACL grants access to an identity other than its owner")
	}
	if ace.Header.AceFlags&windows.INHERITED_ACE != 0 {
		t.Fatal("vault DACL retained an inherited ACE")
	}
}

func daclEntryCount(dacl *windows.ACL) uint16 {
	if dacl == nil {
		return 0
	}
	return dacl.AceCount
}
