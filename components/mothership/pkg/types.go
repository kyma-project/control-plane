package mothership

type State string

const (
	StateOK        State = "ok"
	StateErr       State = "err"
	StateSuspended State = "suspended"
	AllState       State = "all"
)

type Reconciliation struct {
	ID string `json:"id"`
}
