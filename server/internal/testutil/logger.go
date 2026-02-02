package testutil

import (
	"github.com/johnrirwin/rotorlife/internal/logging"
)

// NullLogger returns a logger that discards most output
func NullLogger() *logging.Logger {
	return logging.New(logging.LevelError)
}
