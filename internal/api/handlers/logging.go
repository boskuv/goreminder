package handlers

import (
	"github.com/rs/zerolog"

	errs "github.com/boskuv/goreminder/internal/errors"
)

// errEvent returns an Info event for expected client errors (no stack),
// or an Error+Stack event for unexpected failures.
func errEvent(log zerolog.Logger, err error) *zerolog.Event {
	if errs.IsClientError(err) {
		return log.Info().Err(err)
	}
	return log.Error().Stack().Err(err)
}
