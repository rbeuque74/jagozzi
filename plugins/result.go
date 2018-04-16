package plugins

import (
	"github.com/syncbak-git/nsca"
)

// StatusEnum corresponds to all status for a Consumer message
type StatusEnum int16

const (
	// STATE_OK represents a healthy service
	STATE_OK StatusEnum = nsca.STATE_OK
	// STATE_WARNING represents a service that requires attention
	STATE_WARNING StatusEnum = nsca.STATE_WARNING
	// STATE_CRITICAL represents a service that needs immediate fix
	STATE_CRITICAL StatusEnum = nsca.STATE_CRITICAL
	// STATE_UNKNOWN represents a service in an unsure status
	STATE_UNKNOWN StatusEnum = nsca.STATE_UNKNOWN
)

// Result is the structure that represents a checker result
type Result struct {
	// Status indicates if check was successful or not
	Status StatusEnum
	// Message is the additional message that can be given to Consumer server
	Message string
	// Checker is the checker that returns this result
	Checker Checker
}
