package v1

// Component represents an installable "component" added to the system.
// This can be something like modeldb, or some other service that complements the main system.
type Component struct {
	Name string
	URL  string
}
