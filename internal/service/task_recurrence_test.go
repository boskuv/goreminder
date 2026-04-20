package service

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestNextStartFromRRule_Daily(t *testing.T) {
	now := time.Date(2024, 6, 15, 10, 0, 0, 0, time.UTC)
	seriesStart := time.Date(2024, 1, 1, 9, 0, 0, 0, time.UTC)
	next, err := nextStartFromRRule(now, "FREQ=DAILY;INTERVAL=1", seriesStart)
	require.NoError(t, err)
	require.True(t, next.After(now))
}

func TestValidateCronExpressionAndRRuleExclusive(t *testing.T) {
	require.NoError(t, validateCronExpressionAndRRuleExclusive(nil, nil))
	require.NoError(t, validateCronExpressionAndRRuleExclusive(ptrString("0 * * * *"), nil))
	require.NoError(t, validateCronExpressionAndRRuleExclusive(nil, ptrString("FREQ=DAILY")))
	require.Error(t, validateCronExpressionAndRRuleExclusive(ptrString("0 * * * *"), ptrString("FREQ=DAILY")))
}
