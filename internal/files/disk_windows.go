//go:build windows

package files

import "golang.org/x/sys/windows"

// diskSpace — тухайн замын дискний нийт ба сул (хэрэглэгчид боломжтой) багтаамж.
func diskSpace(path string) (total, free uint64) {
	p, err := windows.UTF16PtrFromString(path)
	if err != nil {
		return 0, 0
	}
	var avail, tot, totFree uint64
	if windows.GetDiskFreeSpaceEx(p, &avail, &tot, &totFree) != nil {
		return 0, 0
	}
	return tot, avail
}
