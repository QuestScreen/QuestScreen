package data

import (
	"golang.org/x/text/transform"
	"golang.org/x/text/unicode/norm"
)

func checkRemove(r rune) bool {
	return (r < 'a' || r > 'z') && (r < 'A' || r > 'Z') &&
		(r < '0' || r > '9') && r != '-' && r != '_'
}

var normalization = transform.Chain(norm.NFD, transform.RemoveFunc(checkRemove))

// generate a normalized string from the input, containing only alphanumerical
// characters, '-' and '_'
func normalize(input string) string {
	result, _, _ := transform.String(normalization, input)
	return result
}
