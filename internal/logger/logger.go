package logger

import (
	"os"
	"time"

	"github.com/rs/zerolog"
)

var log zerolog.Logger

func Init(level, environment string) {
	lvl, err := zerolog.ParseLevel(level)
	if err != nil {
		lvl = zerolog.InfoLevel
	}
	zerolog.SetGlobalLevel(lvl)

	zerolog.TimeFieldFormat = time.RFC3339

	if environment == "development" {
		writer := zerolog.ConsoleWriter{
			Out:        os.Stderr,
			TimeFormat: "2006-01-02 15:04:05",
		}
		log = zerolog.New(writer).With().Timestamp().Logger()
	} else {
		log = zerolog.New(os.Stderr).With().Timestamp().Logger()
	}
}

func Info() *zerolog.Event  { return log.Info() }
func Warn() *zerolog.Event  { return log.Warn() }
func Error() *zerolog.Event { return log.Error() }
func Fatal() *zerolog.Event { return log.Fatal() }
func Debug() *zerolog.Event { return log.Debug() }

func With() zerolog.Context  { return log.With() }
func Logger() zerolog.Logger { return log }
