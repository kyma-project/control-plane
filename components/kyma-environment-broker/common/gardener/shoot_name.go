package gardener

import (
	"fmt"
	"strings"
	"unicode"

	"github.com/google/uuid"
)

// CreateShootName generates random shoot name in pattern "[a-z0-0]{7}" or "c-[a-z0-0]{7}"
func CreateShootName() string {
	id := uuid.New()

	name := strings.ReplaceAll(id.String(), "-", "")
	name = fmt.Sprintf("%.7s", name)
	name = startWithLetter(name)
	name = strings.ToLower(name)
	return name
}

// startWithLetter returns given string but starting with letter
func startWithLetter(str string) string {
	if len(str) == 0 {
		return "c"
	} else if !unicode.IsLetter(rune(str[0])) {
		return fmt.Sprintf("c-%.9s", str)
	}
	return str
}
