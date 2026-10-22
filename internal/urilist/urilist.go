// Package urilist reads the text/uri-list format (RFC 2483) that file
// managers use for files dragged to another app.
package urilist

import (
	"net/url"
	"strings"
)

// Paths returns the local paths of the file:// URIs in list, in order.
// Comment lines and URIs for other schemes or remote hosts are skipped.
func Paths(list string) []string {
	var paths []string
	for _, line := range strings.Split(list, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		u, err := url.Parse(line)
		if err != nil || u.Scheme != "file" || (u.Host != "" && u.Host != "localhost") || u.Path == "" {
			continue
		}
		paths = append(paths, u.Path)
	}
	return paths
}
