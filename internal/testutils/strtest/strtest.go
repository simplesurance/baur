// Package strtest provides test utilties to operate with strings
package strtest

// InSlice returns true if the slice contains the passed string
func InSlice(slice []string, wanted string) bool {
	for _, elem := range slice {
		if elem == wanted {
			return true
		}
	}

	return false
}
