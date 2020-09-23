package util

import (
	"fmt"
	"rand"
	"strings"
	"unicode"
)

// RemoveNotAllowedCharacters returns string containing only alphanumeric characters or hyphens
func RemoveNotAllowedCharacters(str string) string {
	for _, char := range strings.ToLower(str) {
		if !unicode.IsLetter(char) {
			str = strings.ReplaceAll(str, string(char), "")
		}
	}
	return str
}

// StartWithLetter returns given string but starting with letter
func StartWithLetter(str string) string {
	if len(str) == 0 {
		return RandomLetter()
	} else if !unicode.IsLetter(rune(str[0])) {
		return fmt.Sprintf("%s%s", RandomLetter(), str[1:])
	}
	return str
}

// RandomLetter returns randomly generated letter
func RandomLetter() string {
	letterRunes := []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")
	return string(letterRunes[rand.Intn(len(letterRunes))])
}

func NotNilOrEmpty(str *string) bool {
	return str != nil && *str != ""
}

func IsNilOrEmpty(str *string) bool {
	return !NotNilOrEmpty(str)
}
