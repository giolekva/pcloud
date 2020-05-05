package events

type EventState string

// TODO(giolekva): add FAILED
const (
	EventStateNew        EventState = "NEW"
	EventStateProcessing EventState = "PROCESSING"
	EventStateDone       EventState = "DONE"
)

type Event struct {
	Id     string
	State  EventState
	NodeId string
}

type EventStore interface {
	GetEventsInState(state EventState) ([]Event, error)
	MarkEventDone(event Event) error
}
