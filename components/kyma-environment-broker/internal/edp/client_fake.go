package edp

import (
	"errors"
	"fmt"
	"sync"
)

const (
	dataTenantMapKey     = "%s-%s"
	metadataTenantMapKey = "%s-%s-%s"
)

// FakeClient implements the edp client interface but does not process data nor call real external system
type FakeClient struct {
	mu                 sync.Mutex
	dataTenantData     map[string]DataTenantItem
	metadataTenantData map[string]MetadataItem
}

// NewFakeClient creates edp fake client
func NewFakeClient() *FakeClient {
	return &FakeClient{
		dataTenantData:     make(map[string]DataTenantItem),
		metadataTenantData: make(map[string]MetadataItem),
	}
}

func (f *FakeClient) CreateDataTenant(data DataTenantPayload) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	err := checkDataTenantPayload(data)
	if err != nil {
		return err
	}

	key := generateDataTenantMapKey(data.Name, data.Environment)

	_, found := f.dataTenantData[key]
	if found {
		return errors.New("datatenant already exist")
	}

	f.dataTenantData[key] = DataTenantItem{
		Name:        data.Name,
		Environment: data.Environment,
	}
	return nil
}

func (f *FakeClient) CreateMetadataTenant(name, env string, data MetadataTenantPayload) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	err := checkMetadataTenantPayload(data)
	if err != nil {
		return err
	}

	dataMapKey := generateDataTenantMapKey(name, env)
	metadataMapKey := generateMetadataTenantMapKey(name, env, data.Key)

	_, found := f.metadataTenantData[metadataMapKey]
	if found {
		return errors.New("metadatatenant already exist")
	}

	f.metadataTenantData[metadataMapKey] = MetadataItem{
		DataTenant: f.dataTenantData[dataMapKey],
		Key:        data.Key,
		Value:      data.Value,
	}
	return nil
}

func (f *FakeClient) DeleteDataTenant(name, env string) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	key := generateDataTenantMapKey(name, env)

	_, found := f.dataTenantData[key]
	if !found {
		return errors.New("datatenant does not exist")
	}
	delete(f.dataTenantData, key)
	return nil
}

func (f *FakeClient) DeleteMetadataTenant(name, env, key string) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	mapKey := generateMetadataTenantMapKey(name, env, key)

	_, found := f.metadataTenantData[mapKey]
	if !found {
		return errors.New("metadatatenant does not exist")
	}
	delete(f.metadataTenantData, mapKey)
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

func generateDataTenantMapKey(name, env string) string {
	return fmt.Sprintf(dataTenantMapKey, name, env)
}

func generateMetadataTenantMapKey(name, env, key string) string {
	return fmt.Sprintf(metadataTenantMapKey, name, env, key)
}

// assert methods
func (f *FakeClient) GetDataTenantItem(name, env string) (item DataTenantItem, exists bool) {
	key := generateDataTenantMapKey(name, env)
	item, exists = f.dataTenantData[key]
	return item, exists
}

func (f *FakeClient) GetMetadataItem(name, env, key string) (item MetadataItem, exists bool) {
	mapKey := generateMetadataTenantMapKey(name, env, key)
	item, exists = f.metadataTenantData[mapKey]
	return item, exists
}
