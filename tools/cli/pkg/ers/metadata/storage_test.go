package metadata

import (
	"github.com/kyma-project/control-plane/tools/cli/pkg/ers"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestSaveGet(t *testing.T) {
	// given
	m := ers.MigrationMetadata{
		Id:           "1234",
		KymaMigrated: true,
		KymaSkipped:  true,
	}
	svc := Storage{}

	// when
	err := svc.Save(m)
	require.NoError(t, err)

	// then
	stored, err := svc.Get(m.Id)
	require.NoError(t, err)
	assert.Equal(t, m, stored)
}
