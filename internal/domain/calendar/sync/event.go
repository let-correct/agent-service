package calendarsync

import "time"

type DetailType string

const (
	DetailTypeCreated   DetailType = "AppointmentCreated"
	DetailTypeCancelled DetailType = "AppointmentCancelled"
)

type EventMetadata struct {
	Timestamp     time.Time `json:"timestamp"`
	Source        string    `json:"source"`
	SchemaVersion string    `json:"schemaVersion"`
	CorrelationID string    `json:"correlationId"`
}

type Attendee struct {
	Name  string `json:"name"`
	Email string `json:"email"`
	Phone string `json:"phone"`
}

type Appointment struct {
	AgentEmail string    `json:"agentEmail"`
	Location   string    `json:"location"`
	Start      time.Time `json:"start"`
	End        time.Time `json:"end"`
	Attendee   Attendee  `json:"attendee"`
}

type Event struct {
	DetailType DetailType    `json:"detailType"`
	Metadata   EventMetadata `json:"metadata"`
	Payload    Appointment   `json:"payload"`
}
