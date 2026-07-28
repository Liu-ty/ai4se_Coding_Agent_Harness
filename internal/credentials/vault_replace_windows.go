//go:build windows

package credentials

import (
	"errors"
	"unsafe"

	"golang.org/x/sys/windows"
)

const replaceFileWriteThrough = 0x1

var replaceFileW = windows.NewLazySystemDLL("kernel32.dll").NewProc("ReplaceFileW")

func replaceVaultFile(replacement, target string) error {
	replacementPath, err := windows.UTF16PtrFromString(replacement)
	if err != nil {
		return err
	}
	targetPath, err := windows.UTF16PtrFromString(target)
	if err != nil {
		return err
	}
	callReplace := func() error {
		result, _, callErr := replaceFileW.Call(
			uintptr(unsafe.Pointer(targetPath)),
			uintptr(unsafe.Pointer(replacementPath)),
			0,
			replaceFileWriteThrough,
			0,
			0,
		)
		if result == 0 {
			return callErr
		}
		return nil
	}
	if err := callReplace(); err == nil {
		return nil
	} else if !errors.Is(err, windows.ERROR_FILE_NOT_FOUND) &&
		!errors.Is(err, windows.ERROR_PATH_NOT_FOUND) {
		return err
	}
	if err := windows.MoveFileEx(replacementPath, targetPath, windows.MOVEFILE_WRITE_THROUGH); err != nil {
		if errors.Is(err, windows.ERROR_ALREADY_EXISTS) || errors.Is(err, windows.ERROR_FILE_EXISTS) {
			return callReplace()
		}
		return err
	}
	return nil
}
