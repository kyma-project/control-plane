package ias

//go:generate mockery --name=BundleBuilder --output=automock --outpkg=automock --case=underscore
//go:generate mockery --name=Bundle --output=automock --outpkg=automock --case=underscore
type (
	BundleBuilder interface {
		NewBundle(identifier string, inputID SPInputID) (Bundle, error)
	}

	Bundle interface {
		FetchServiceProviderData() error
		ServiceProviderName() string
		ServiceProviderType() string
		ServiceProviderExist() bool
		CreateServiceProvider() error
		DeleteServiceProvider() error
		ConfigureServiceProvider() error
		ConfigureServiceProviderType(path string) error
		GenerateSecret() (*ServiceProviderSecret, error)
	}
)

type Builder struct {
	iasClient IASCLient
	config    Config
}

func NewBundleBuilder(iasClient IASCLient, config Config) BundleBuilder {
	return &Builder{
		iasClient: iasClient,
		config:    config,
	}
}

func (b *Builder) NewBundle(identifier string, inputID SPInputID) (Bundle, error) {
	if err := inputID.isValid(); err != nil {
		return nil, err
	}
	return NewServiceProviderBundle(identifier, ServiceProviderInputs[inputID], b.iasClient, b.config), nil
}
