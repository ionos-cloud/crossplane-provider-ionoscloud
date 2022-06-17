package utils

// PointerString returns a pointer to the given string. Useful to get pointers to string literals or constants.
func PointerString(in string) *string {
	return &in
}
