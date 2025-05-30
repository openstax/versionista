package main

import (
	"strings"
)

func removeDuplicates(input []string) []string {
	seen := make(map[string]struct{})
	result := []string{}

	for _, str := range input {
		str := strings.ToUpper(str)
		if _, ok := seen[str]; !ok {
			seen[str] = struct{}{}
			result = append(result, str)
		}
	}

	return result
}

func mdQuote(text string) string {
	escapedChars := []string{"|"}
	for _, c := range escapedChars {
		text = strings.ReplaceAll(text, c, "\\"+c)
	}
	return text
}
