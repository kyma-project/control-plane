package edp

import "errors"

// FakeClient implements the edp client interface but does not process data nor call real external system
type FakeClient struct {
	data map[string]interface{}
}

// NewFakeClient creates edp fake client
func NewFakeClient() *FakeClient {
	return &FakeClient{
		data: make(map[string]interface{}),
	}
}

func (f *FakeClient) CreateDataTenant(data DataTenantPayload) error {
	err := checkDataTenantPayload(data)
	if err != nil {
		return err
	}
	f.data["DataTenant"] = struct {
		Name        string
		Environment string
		Secret      string
	}{
		Name:        data.Name,
		Environment: data.Environment,
		Secret:      data.Secret,
	}
	return nil
}

func (f *FakeClient) CreateMetadataTenant(name, env string, data MetadataTenantPayload) error {
	err := checkMetadataTenantPayload(data)
	if err != nil {
		return err
	}
	f.data["MetadataTenant"] = struct {
		Key   string
		Value string
	}{
		Key:   data.Key,
		Value: data.Value,
	}
	return nil
}

func (f *FakeClient) DeleteDataTenant(name, env string) error {
	_, found := f.data["DataTenant"]
	if !found {
		return errors.New("datatenant does not exist")
	}
	delete(f.data, "DataTenant")
	return nil
}

func (f *FakeClient) DeleteMetadataTenant(name, env, key string) error {
	_, found := f.data["MetadataTenant"]
	if !found {
		return errors.New("metadatatenant does not exist")
	}
	delete(f.data, "MetadataTenant")
	return nil
}

func checkDataTenantPayload(data DataTenantPayload) error {
	if data.Name == "" || data.Environment == "" || data.Secret == "" {
		return errors.New("one of the fields in DataTenantPayload is missing")
	}
	return nil
}

func checkMetadataTenantPayload(data MetadataTenantPayload) error {
	if data.Key == "" || data.Value == "" {
		return errors.New("one of the fields in MetadataTenantPayload is missing")
	}
	return nil
}
