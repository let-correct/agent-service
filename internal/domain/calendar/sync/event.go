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

type Appointment struct {
	Email       string    `json:"email"`
	EventID     string    `json:"eventId"`
	CalendarID  string    `json:"calendarId"`
	Summary     string    `json:"summary"`
	Description string    `json:"description"`
	Start       time.Time `json:"start"`
	End         time.Time `json:"end"`
	Attendees   []string  `json:"attendees"`
	Status      string    `json:"status"`
}

type Event struct {
	DetailType DetailType    `json:"detailType"`
	Metadata   EventMetadata `json:"metadata"`
	Payload    Appointment   `json:"payload"`
}
