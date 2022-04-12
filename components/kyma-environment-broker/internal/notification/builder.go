package notification

type (
	BundleBuilder interface {
		NewBundle(identifier string, notificationParams NotificationParams) (Bundle, error)
		DisabledCheck() bool
	}

	Bundle interface {
		CreateNotificationEvent() error
		UpdateNotificationEvent() error
		CancelNotificationEvent() error
	}
)

type Builder struct {
	notificationClient NotificationClient
	config             Config
}

func NewBundleBuilder(notificationClient NotificationClient, config Config) BundleBuilder {
	return &Builder{
		notificationClient: notificationClient,
		config:             config,
	}
}

func (b *Builder) NewBundle(identifier string, notificationParams NotificationParams) (Bundle, error) {
	return NewNotificationBundle(identifier, notificationParams, b.notificationClient, b.config), nil
}

func (b *Builder) DisabledCheck() bool {
	return b.config.Disabled
}
