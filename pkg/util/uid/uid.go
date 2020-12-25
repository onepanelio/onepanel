package uid

import (
	"fmt"
	"regexp"
	"strings"
)

// GenerateUID converts an input string to a uid friendly version where we have all lowercase letters and dashes instead of spaces
// If the max is exceeded, an error is returned
func GenerateUID(input string, max int) (string, error) {
	re, _ := regexp.Compile(`[^a-zA-Z0-9-]{1,}`)
	cleanUp := strings.ToLower(re.ReplaceAllString(input, `-`))
	if len(cleanUp) > max {
		return "", fmt.Errorf("length of string '%s' (%d) exceeds %d", input, len(input), max)
	}
	return strings.ToLower(re.ReplaceAllString(input, `-`)), nil
}
