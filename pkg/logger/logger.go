package logger

import (
	"os"
	"time"

	"github.com/rs/zerolog"
)

// New creates and configures a zerolog.Logger. In development mode the output
// is pretty-printed; in production it emits compact JSON.
func New(env string) zerolog.Logger {
	zerolog.TimeFieldFormat = time.RFC3339Nano

	if env == "development" {
		return zerolog.New(zerolog.ConsoleWriter{Out: os.Stdout, TimeFormat: time.RFC3339}).
			With().
			Timestamp().
			Caller().
			Logger()
	}

	return zerolog.New(os.Stdout).
		With().
		Timestamp().
		Logger()
}
