package logging

import (
	"os"
	"time"

	"github.com/rs/zerolog"
)

var RootLogger zerolog.Logger = zerolog.New(
	zerolog.NewConsoleWriter(
		func(w *zerolog.ConsoleWriter) { w.Out = os.Stderr },
		func(w *zerolog.ConsoleWriter) {
			zerolog.TimeFieldFormat = time.RFC3339Nano
			w.TimeFormat = "15:04:05.000"
		})).Level(zerolog.InfoLevel).
	With().Timestamp().Logger()
