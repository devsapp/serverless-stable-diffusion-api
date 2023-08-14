package config

import "time"

const (
	// model status
	REGISTERING = "registering"
	LOADING     = "loading"
	LOADED      = "loading"
	UNLOADED    = "unloaded"

	// task status
	INPROGRESS = "inprogress"
	FAILED     = "failed"
	QUEUE      = "queue"

	HTTPTIMEOUT = 500 * time.Microsecond

	// message
	INTERNALERROR = "an internal error"
	BADREQUEST    = "bad request body"
	NOTFOUND      = "not found"
)
