package sync

import (
	"os"
)

// GetTopLevelFiles returns the file information at the top level of the directory, similar to ioutil.ReadDir but does not sort
// since we don't need that
// If the call fails, returns and empty slice of FileInfo
func GetTopLevelFiles(dirName string) ([]os.FileInfo, error) {
	items, err := readElements(dirName)
	if err != nil {
		return make([]os.FileInfo, 0), err
	}

	files := items[:0]
	for _, item := range items {
		if !item.IsDir() && item.Mode()&os.ModeSymlink == 0 {
			files = append(files, item)
		}
	}
	return files, nil
}

// GetTopLevelFolders returns the file information at the top level of the directory, similar to ioutil.ReadDir but does not sort
// since we don't need that
// If the call fails, returns and empty slice of FileInfo
func GetTopLevelFolders(dirName string) ([]os.FileInfo, error) {
	items, err := readElements(dirName)
	if err != nil {
		return make([]os.FileInfo, 0), err
	}

	dirs := items[:0]
	for _, item := range items {
		if item.IsDir() && item.Mode()&os.ModeSymlink == 0 {
			// in the future we need to evaluate the symlinks using
			// filepath.EvalSymlinks()
			dirs = append(dirs, item)
		}
	}
	return dirs, nil
}

// returns FileInfo objects for filtering, only returns a slice on success
func readElements(dirName string) ([]os.FileInfo, error) {
	f, err := os.Open(dirName)
	defer f.Close()
	if err != nil {
		return nil, err
	}
	items, err := f.Readdir(-1)
	if err != nil {
		return nil, err
	}
	return items, nil
}
