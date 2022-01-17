package logging

import (
	"os"
	"time"

	"github.com/rs/zerolog"
)

var file, _ = os.Create("./log.txt")

var RootLogger zerolog.Logger = zerolog.New(
	zerolog.NewConsoleWriter(
		func(w *zerolog.ConsoleWriter) { w.Out = file },
		func(w *zerolog.ConsoleWriter) {
			zerolog.TimeFieldFormat = time.RFC3339Nano
			w.TimeFormat = "15:04:05.000"
		})).Level(zerolog.ErrorLevel).
	With().Timestamp().Logger()
