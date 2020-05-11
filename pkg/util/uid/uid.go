package uid

import (
	"errors"
	"regexp"
	"strings"
)

func GenerateUID(input string) (string, error) {
	re, _ := regexp.Compile(`[^a-zA-Z0-9-]{1,}`)
	cleanUp := strings.ToLower(re.ReplaceAllString(input, `-`))
	if len(cleanUp) > 30 {
		return "", errors.New("Length of string exceeds 30, invalid for k8s.")
	}
	return strings.ToLower(re.ReplaceAllString(input, `-`)), nil
}
