package supervisor

// State is the controller or generation lifecycle.
type State string

const (
	StateIdle     State = "idle"
	StateBuilding State = "building"
	StateStarting State = "starting"
	StateReady    State = "ready"
	StateStopping State = "stopping"
	StateFailed   State = "failed"
	StateClosed   State = "closed"
)

func (s State) String() string {
	return string(s)
}
