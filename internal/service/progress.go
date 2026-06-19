package service

type ProgressEvent struct {
	GenID  string
	Step   string
	Status string
	Error  string
}

type ProgressPublisher interface {
	Publish(genID string, event ProgressEvent)
}
