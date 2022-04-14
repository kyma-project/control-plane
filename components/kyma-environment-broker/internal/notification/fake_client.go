package notification

import "fmt"

const (
	FakeOrchestrationID = "ASKHGK-SAKJHTJ-ALKJSHT-HUZIUOP"
	FakeInstanceID      = "WEAJKG-INSTANCE-1"
)

type Tenants struct {
	InstanceID string
	StartDate  string
	EndDate    string
	State      string
}

type MaintenanceEvent struct {
	OrchestrationID string
	EventType       string
	Tenants         []Tenants
}

type FakeClient struct {
	maintenanceEvent []*MaintenanceEvent
}

func NewFakeClient() *FakeClient {
	return &FakeClient{
		maintenanceEvent: []*MaintenanceEvent{
			{
				OrchestrationID: FakeOrchestrationID,
				EventType:       "test",
				Tenants: []Tenants{
					{
						InstanceID: "test",
						StartDate:  "test",
						EndDate:    "test",
						State:      "test",
					},
				},
			},
		},
	}
}

func (f *FakeClient) CreateEvent(request CreateEventRequest) error {
	f.maintenanceEvent = append(f.maintenanceEvent, &MaintenanceEvent{
		OrchestrationID: request.OrchestrationID,
		EventType:       request.EventType,
		Tenants: []Tenants{
			{
				InstanceID: request.Tenants[0].InstanceID,
				StartDate:  request.Tenants[0].StartDate,
			},
		},
	})

	return nil
}

func (f *FakeClient) UpdateEvent(request UpdateEventRequest) error {
	maintenanceEvent, err := f.GetMaintenanceEvent(request.OrchestrationID)
	if err != nil {
		return err
	}

	maintenanceEvent.Tenants[0].StartDate = request.Tenants[0].StartDate
	maintenanceEvent.Tenants[0].State = request.Tenants[0].State

	return nil
}

func (f *FakeClient) CancelEvent(request CancelEventRequest) error {
	maintenanceEvent, err := f.GetMaintenanceEvent(request.OrchestrationID)
	if err != nil {
		return err
	}

	maintenanceEvent.Tenants[0].State = CancelledMaintenanceState
	return nil
}

func (f *FakeClient) GetMaintenanceEvent(id string) (*MaintenanceEvent, error) {
	for _, event := range f.maintenanceEvent {
		if event.OrchestrationID == id {
			return event, nil
		}
	}

	return nil, fmt.Errorf("cannot find MaintenanceEvent with OrchestrationID: %s", id)
}
