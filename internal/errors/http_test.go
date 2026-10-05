package errors

import (
	"fmt"
	"testing"
)

func TestIsClientError(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		err  error
		want bool
	}{
		{"not found", ErrNotFound, true},
		{"wrapped not found", fmt.Errorf("wrap: %w", ErrNotFound), true},
		{"validation", ErrValidation, true},
		{"unprocessable", ErrUnprocessableEntity, true},
		{"conflict", ErrConflict, true},
		{"other", fmt.Errorf("boom"), false},
		{"nil", nil, false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := IsClientError(tc.err); got != tc.want {
				t.Fatalf("IsClientError(%v)=%v want %v", tc.err, got, tc.want)
			}
		})
	}
}
