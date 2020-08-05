package v1

// Service represents an installable "service" added to the system.
// This can be something like modeldb, or some other service that complements the main system.
type Service struct {
	Name string
	URL  string
}
