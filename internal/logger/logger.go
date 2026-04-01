package logger

import (
	"os"

	"github.com/rs/zerolog"
)

var log zerolog.Logger

func Init(level, environment string) {
	lvl, err := zerolog.ParseLevel(level)
	if err != nil {
		lvl = zerolog.InfoLevel
	}
	zerolog.SetGlobalLevel(lvl)

	if environment == "development" {
		log = zerolog.New(zerolog.ConsoleWriter{Out: os.Stderr}).With().Timestamp().Logger()
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
