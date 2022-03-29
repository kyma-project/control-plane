package client

import (
	"fmt"
	"net/http"
	"time"

	"github.com/kyma-project/control-plane/tools/cli/pkg/ers"
)

type migrationInfo struct {
	started time.Time
}

type Fake struct {
	migrations map[string]migrationInfo
}

func NewFake() *Fake {
	return &Fake{
		migrations: make(map[string]migrationInfo),
	}
}

func (f *Fake) GetOne(id string) (*ers.Instance, error) {
	fmt.Printf(">>> GetOne(%s)\n", id)
	m, found := f.migrations[id]
	if !found {
		return &ers.Instance{
			Id:       id,
			Name:     id,
			Migrated: false,
		}, nil
	}
	inst := &ers.Instance{
		Id:       id,
		Name:     id,
		Migrated: time.Since(m.started) > time.Second*24,
	}
	fmt.Printf(">>> returning %s migrated=%v\n", inst.Id, inst.Migrated)
	return inst, nil
}

func (f *Fake) GetPaged(pageStart, pageSize int) ([]ers.Instance, error) {
	panic("GetPaged not implemented")
}

func (f *Fake) Migrate(instanceID string) error {
	fmt.Printf(">>> Migrate(%s)\n", instanceID)
	if _, found := f.migrations[instanceID]; found {
		return nil
	}
	f.migrations[instanceID] = migrationInfo{started: time.Now()}
	return nil
}

func (f *Fake) Switch(brokerID string) error {
	panic("Switch not implemented")
}

func (f *Fake) Close() {
	fmt.Println(">>> Close()")
}

func (f *Fake) GetClient() *http.Client {
	return nil
}
