package validation

import (
	"testing"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/stretchr/testify/require"
)

func TestFutureDateValidatorForUpdateStartDate(t *testing.T) {
	v := validator.New()
	require.NoError(t, RegisterCustomValidators(v))

	type updatePayload struct {
		StartDate *time.Time `validate:"omitempty,future_date"`
	}

	past := time.Now().UTC().Add(-1 * time.Hour)
	err := v.Struct(updatePayload{StartDate: &past})
	require.Error(t, err, "past start_date must be rejected when explicitly provided")

	future := time.Now().UTC().Add(1 * time.Hour)
	err = v.Struct(updatePayload{StartDate: &future})
	require.NoError(t, err, "future start_date must pass validation")

	err = v.Struct(updatePayload{StartDate: nil})
	require.NoError(t, err, "missing start_date in partial update must skip future_date validation")
}
