package pkg

// ContainsString checks if a string is present in a slice of strings
func ContainsString(slice []string, str string) bool {
	for _, item := range slice {
		if item == str {
			return true
		}
	}
	return false
}
