package architecture

import "path/filepath"

func buildDirFiles(fileFuncs, fileTypes map[string][]string) map[string][]string {
	dirFiles := map[string][]string{}
	for f := range fileFuncs {
		dirFiles[filepath.Dir(f)] = append(dirFiles[filepath.Dir(f)], f)
	}
	for f := range fileTypes {
		dir := filepath.Dir(f)
		for _, existing := range dirFiles[dir] {
			if existing == f {
				goto next
			}
		}
		dirFiles[dir] = append(dirFiles[dir], f)
	next:
	}
	return dirFiles
}
