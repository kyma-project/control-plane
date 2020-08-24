package templates

import (
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
)

const renderTemplate = `Shoot: {{ .ShootName }}
Gardener Project: {{ .ProjectName }}
Gardener Secret: {{ .GardenerSecretName }}
Region: {{ .Region }}`

func TestRenderTemplates(t *testing.T) {

	expectedRender := `Shoot: my-shoot
Gardener Project: my-project
Gardener Secret: my-secret
Region: eu-west`

	values := Values{
		ShootName:          "my-shoot",
		ProjectName:        "my-project",
		GardenerSecretName: "my-secret",
		Region:             "eu-west",
	}

	result, err := RenderTemplate(renderTemplate, values)
	require.NoError(t, err)

	assert.Equal(t, expectedRender, string(result))
}
