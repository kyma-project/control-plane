package util

import (
	"fmt"
	"github.com/kyma-project/control-plane/components/provisioner/internal/uuid"
	"strings"
)

func CreateGardenerClusterName() string {
	generator := uuid.NewUUIDGenerator
	id := generator().New()

	name := strings.ReplaceAll(id, "-", "")
	name = fmt.Sprintf("%.7s", name)
	name = StartWithLetter(name)
	name = strings.ToLower(name)
	return name
}
