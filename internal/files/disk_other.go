//go:build !windows

package files

import "syscall"

// diskSpace — тухайн замын дискний нийт ба сул (хэрэглэгчид боломжтой) багтаамж.
func diskSpace(path string) (total, free uint64) {
	var st syscall.Statfs_t
	if syscall.Statfs(path, &st) != nil {
		return 0, 0
	}
	return st.Blocks * uint64(st.Bsize), st.Bavail * uint64(st.Bsize)
}
