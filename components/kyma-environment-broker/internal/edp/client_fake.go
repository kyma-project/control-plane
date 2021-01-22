package edp

// FakeClient implements the edp client interface but does not process data nor call real external system
type FakeClient struct{}

// NewFakeClient creates edp fake client
func NewFakeClient() *FakeClient {
	return &FakeClient {}
}

func (f *FakeClient) CreateDataTenant(data DataTenantPayload) error {
	return nil
}

func (f *FakeClient) CreateMetadataTenant(name, env string, data MetadataTenantPayload) error {
	return nil
}

func (f *FakeClient) DeleteDataTenant(name, env string) error {
	return nil
}

func (f *FakeClient) DeleteMetadataTenant(name, env, key string) error {
	return nil
}
