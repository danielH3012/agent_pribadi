package tools

import (
	"fmt"
	"io/fs"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

// GlobToRegex - Convert glob pattern to regex
func GlobToRegex(pattern string) *regexp.Regexp {
	// 1. Escape special regex chars (except glob chars)
	regex := ""
	i := 0

	for i < len(pattern) {
		// 2. Handle ** (recursive)
		if i+1 < len(pattern) && pattern[i:i+2] == "**" {
			regex += ".*" // Match anything including /
			i += 2
			if i < len(pattern) && pattern[i] == '/' {
				regex += "/"
				i++
			}
			continue
		}

		// 3. Handle * (non-recursive)
		if pattern[i] == '*' {
			regex += "[^/]*" // Match anything except /
			i++
			continue
		}

		// 4. Handle ? (single char)
		if pattern[i] == '?' {
			regex += "[^/]" // Single char except /
			i++
			continue
		}

		// 5. Handle [ ] (character class)
		if pattern[i] == '[' {
			j := i
			for j < len(pattern) && pattern[j] != ']' {
				j++
			}
			if j < len(pattern) {
				// Copy [...] as-is (regex syntax)
				regex += pattern[i : j+1]
				i = j + 1
				continue
			}
		}

		// 6. Escape regex special chars
		if strings.ContainsAny(string(pattern[i]), ".+^$(){}|\\") {
			regex += "\\" + string(pattern[i])
		} else {
			regex += string(pattern[i])
		}
		i++
	}

	// 7. Anchor with ^ and $
	regex = "^" + regex + "$"

	// 8. Compile regex
	compiled, _ := regexp.Compile(regex)
	return compiled
}

// Glob - Find files matching pattern
func Glob(pattern string) ([]string, error) {
	// Ask user permission
	if err := AskPermission("glob", fmt.Sprintf("Search files matching glob pattern: %s (recursive: %v)", pattern, strings.Contains(pattern, "**"))); err != nil {
		return nil, err
	}

	regex := GlobToRegex(pattern)
	isRecursive := strings.Contains(pattern, "**")

	var results []string

	// Walk directory
	filepath.WalkDir(".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() {
			// Jika non-recursive dan bukan root, skip
			if !isRecursive && path != "." {
				return filepath.SkipDir
			}
			return nil
		}

		// Get relative path
		relPath := strings.TrimPrefix(path, "."+string(filepath.Separator))

		// Match
		if regex.MatchString(relPath) {
			results = append(results, relPath)
		}

		return nil
	})

	// Sort
	sort.Strings(results)
	return results, nil
}
