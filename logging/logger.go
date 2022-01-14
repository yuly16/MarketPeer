package logging

import (
	"os"

	"github.com/rs/zerolog"
)

var RootLogger zerolog.Logger = zerolog.New(
	zerolog.NewConsoleWriter(
		func(w *zerolog.ConsoleWriter) { w.Out = os.Stderr },
		func(w *zerolog.ConsoleWriter) { w.TimeFormat = "15:04:05.000" })).Level(zerolog.ErrorLevel).
	With().Timestamp().Logger()
