package moonlighter

import (
	"fmt"
	"syscall"
	"unsafe"
)

var (
	user32   = syscall.NewLazyDLL("user32.dll")
	kernel32 = syscall.NewLazyDLL("kernel32.dll")
	shell32  = syscall.NewLazyDLL("shell32.dll")
)

// ReadSignals reads the owner's idle time, one full-screen game signal and
// the host's free RAM on Windows through the standard library's syscall.
// Any failed call returns an error so nothing is lent.
func ReadSignals() Signals {
	input := struct{ Size, Time uint32 }{Size: 8}
	if r, _, err := user32.NewProc("GetLastInputInfo").Call(uintptr(unsafe.Pointer(&input))); r == 0 {
		return Signals{Err: fmt.Errorf("GetLastInputInfo: %v", err)}
	}
	tick, _, _ := user32.NewProc("GetTickCount").Call()
	var state uint32
	if hr, _, _ := shell32.NewProc("SHQueryUserNotificationState").Call(uintptr(unsafe.Pointer(&state))); hr != 0 {
		return Signals{Err: fmt.Errorf("SHQueryUserNotificationState: HRESULT 0x%x", hr)}
	}
	mem := struct {
		Length, MemoryLoad           uint32
		TotalPhys, AvailPhys         uint64
		TotalPageFile, AvailPageFile uint64
		TotalVirtual, AvailVirtual   uint64
		AvailExtendedVirtual         uint64
	}{Length: 64}
	if r, _, err := kernel32.NewProc("GlobalMemoryStatusEx").Call(uintptr(unsafe.Pointer(&mem))); r == 0 {
		return Signals{Err: fmt.Errorf("GlobalMemoryStatusEx: %v", err)}
	}
	return Signals{Idle: IdleSince(uint32(tick), input.Time), OwnerInput: FullScreenGame(state), FreeRAM: mem.AvailPhys}
}
