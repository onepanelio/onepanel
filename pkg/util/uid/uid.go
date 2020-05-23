package uid

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
)

func GenerateUID(input string, max int) (string, error) {
	re, _ := regexp.Compile(`[^a-zA-Z0-9-]{1,}`)
	cleanUp := strings.ToLower(re.ReplaceAllString(input, `-`))
	if len(cleanUp) > max {
		return "", errors.New(fmt.Sprintf("Length of string exceeds %d", max))
	}
	return strings.ToLower(re.ReplaceAllString(input, `-`)), nil
}
