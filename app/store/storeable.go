package store

import "apns_feedback_service/app/feedback"

type Storeable interface {
	Store(resp feedback.FeedbackResponse) error
	Name() string
	Connection() error
	Disconnection() error
}
