package files

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
)

// FolderSpace — файлын сангийн дээд түвшний нэг хавтас (teachers, books г.м).
type FolderSpace struct {
	Name  string `json:"name"`
	Bytes int64  `json:"bytes"`
	Files int    `json:"files"`
}

// Space — файлын сан (STORAGE_DIR)-гийн эзлэх хэмжээ ба тухайн дискний багтаамж (хянагчийн самбарт).
type Space struct {
	Root      string        `json:"root"`
	Bytes     int64         `json:"bytes"`
	Files     int           `json:"files"`
	Folders   []FolderSpace `json:"folders"`
	DiskTotal uint64        `json:"disk_total"`
	DiskFree  uint64        `json:"disk_free"`
}

// Space нь файлын санг бүхэлд нь гүйж нийт хэмжээ, файлын тоо, дээд түвшний хавтас бүрийн задаргааг гаргана.
func (s *Store) Space() (*Space, error) {
	root, err := filepath.Abs(s.root)
	if err != nil {
		root = s.root
	}
	out := &Space{Root: root, Folders: []FolderSpace{}}
	byTop := map[string]*FolderSpace{}
	err = filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			if errors.Is(err, fs.ErrNotExist) || errors.Is(err, fs.ErrPermission) {
				return nil
			}
			return err
		}
		if d.IsDir() {
			return nil
		}
		fi, err := d.Info()
		if err != nil {
			return nil
		}
		rel, _ := filepath.Rel(root, p)
		top := "(үндсэн)" // үндсэн хавтсанд шууд байгаа файл
		if i := indexSep(rel); i >= 0 {
			top = rel[:i]
		}
		f := byTop[top]
		if f == nil {
			f = &FolderSpace{Name: top}
			byTop[top] = f
		}
		f.Bytes += fi.Size()
		f.Files++
		out.Bytes += fi.Size()
		out.Files++
		return nil
	})
	for _, f := range byTop {
		out.Folders = append(out.Folders, *f)
	}
	sort.Slice(out.Folders, func(i, j int) bool { return out.Folders[i].Bytes > out.Folders[j].Bytes })
	out.DiskTotal, out.DiskFree = diskSpace(root)
	if _, statErr := os.Stat(root); statErr != nil && err == nil {
		err = statErr
	}
	return out, err
}

func indexSep(p string) int {
	for i := 0; i < len(p); i++ {
		if os.IsPathSeparator(p[i]) {
			return i
		}
	}
	return -1
}
