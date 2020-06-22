package v1

import (
	"strings"
	"time"
)

// File represents a system file.
type File struct {
	Path         string
	Name         string
	Size         int64
	Extension    string
	ContentType  string
	LastModified time.Time
	Directory    bool
}

// FilePathToParentPath given a path, returns the parent path, assuming a '/' delimiter
// Result does not have a trailing slash.
// -> a/b/c/d would return a/b/c
// -> a/b/c/d/ would return a/b/c
// If path is empty string, it is returned.
// If path is '/' (root) it is returned as is.
// If there is no '/', '/' is returned.
func FilePathToParentPath(path string) string {
	separator := "/"
	if path == "" || path == separator {
		return path
	}

	if strings.HasSuffix(path, "/") {
		path = path[0 : len(path)-1]
	}

	lastIndexOfForwardSlash := strings.LastIndex(path, separator)
	if lastIndexOfForwardSlash <= 0 {
		return separator
	}

	return path[0:lastIndexOfForwardSlash]
}

// FilePathToExtension returns the file's extension if it uses a dot "." to denote it.
// otherwise it returns the text following the last dot in the path.
func FilePathToExtension(path string) string {
	dotIndex := strings.LastIndex(path, ".")

	if dotIndex == -1 {
		return ""
	}

	if dotIndex == (len(path) - 1) {
		return ""
	}

	return path[dotIndex+1:]
}

// FilePathToName returns the name of the file, assuming that "/" denote directories and that the
// file name is after the last "/"
func FilePathToName(path string) string {
	if strings.HasSuffix(path, "/") {
		path = path[:len(path)-1]
	}

	lastSlashIndex := strings.LastIndex(path, "/")
	if lastSlashIndex < 0 {
		return path
	}

	return path[lastSlashIndex+1:]
}
