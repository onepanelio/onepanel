package number

import (
	"fmt"
	"strconv"
)

// Takes the input value, increments it, and formats it as a string.
func IncrementStringInt(value string) (string, error) {
	numericValue, err := strconv.Atoi(value)
	if err != nil {
		return value, err
	}

	numericValue++

	stringValue := fmt.Sprintf("%v", numericValue)

	return stringValue, nil
}
