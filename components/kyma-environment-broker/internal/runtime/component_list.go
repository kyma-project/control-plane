package runtime

type ComponentListData struct {
	DefaultNamespace string `yaml:"defaultNamespace" json:"defaultNamespace"`
	Prerequisites    []ComponentDefinition
	Components       []ComponentDefinition
}

type ComponentSource struct {
	URL string
}

type ComponentDefinition struct {
	Name      string
	Namespace string
	Source    *ComponentSource
}
