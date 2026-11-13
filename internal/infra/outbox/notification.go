package outbox

import "time"

type Notification struct {
	ID        string
	Message   string
	CreatedAt time.Time
}
