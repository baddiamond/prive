package platform

import (
	"os"
	"syscall"
	"unsafe"
)

// localeNameMaxLength mirrors LOCALE_NAME_MAX_LENGTH from the Windows SDK.
const localeNameMaxLength = 85

// systemLocale returns the user's default locale name (e.g. "es-ES") from the Windows API.
func systemLocale() string {
	proc := syscall.NewLazyDLL("kernel32.dll").NewProc("GetUserDefaultLocaleName")
	if proc.Find() == nil {
		buf := make([]uint16, localeNameMaxLength)
		n, _, _ := proc.Call(uintptr(unsafe.Pointer(&buf[0])), uintptr(len(buf)))
		if n > 0 {
			return syscall.UTF16ToString(buf)
		}
	}
	return localeFromEnv(os.Getenv)
}
