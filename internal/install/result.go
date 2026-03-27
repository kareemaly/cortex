package install

// ItemStatus represents the outcome of a setup item.
type ItemStatus int

const (
	// StatusCreated indicates the item was newly created.
	StatusCreated ItemStatus = iota
	// StatusExists indicates the item already existed.
	StatusExists
	// StatusSkipped indicates the item was skipped.
	StatusSkipped
)

// String returns a human-readable status.
func (s ItemStatus) String() string {
	switch s {
	case StatusCreated:
		return "created"
	case StatusExists:
		return "exists"
	case StatusSkipped:
		return "skipped"
	default:
		return "unknown"
	}
}

// SetupItem represents a single setup operation result.
type SetupItem struct {
	Path   string
	Status ItemStatus
	Error  error
}
