package logging

import (
	"github.com/rs/zerolog"
	"os"
)

var RootLogger zerolog.Logger = zerolog.New(
	zerolog.NewConsoleWriter(
		func(w *zerolog.ConsoleWriter) { w.Out = os.Stderr },
		func(w *zerolog.ConsoleWriter) { w.TimeFormat = "15:04:05.000" })).Level(zerolog.InfoLevel).
	With().Timestamp().Logger()
