package collection

// Returns symbol <count> times, with <separator> between each one.
// if symbol = ?, separator = , and count = 5
// this returns: "?,?,?,?,?"
func RepeatSymbol(count int, symbol, separator string) string {
	result := ""
	for i := 0; i < count; i++ {
		if i != 0 {
			result += separator
		}

		result += symbol
	}

	return result
}
