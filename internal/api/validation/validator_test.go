package validation

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIsPrintableUTF8(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  bool
	}{
		{name: "ascii", input: "meeting", want: true},
		{name: "cyrillic", input: "встреча", want: true},
		{name: "emoji", input: "задача 🚀", want: true},
		{name: "mixed", input: "Встреча team 🎉", want: true},
		{name: "space", input: "a b", want: true},
		{name: "newline", input: "a\nb", want: false},
		{name: "tab", input: "a\tb", want: false},
		{name: "null", input: "a\x00b", want: false},
		{name: "invalid utf8", input: "a\xffb", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, isPrintableUTF8(tt.input))
		})
	}
}

func TestValidateOptionalUTF8StringQuery(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("accepts cyrillic and emoji", func(t *testing.T) {
		c, _ := gin.CreateTestContext(httptest.NewRecorder())
		c.Request = httptest.NewRequest(http.MethodGet, "/?title=%D0%B2%D1%81%D1%82%D1%80%D0%B5%D1%87%D0%B0+%F0%9F%9A%80", nil)

		got, err := ValidateOptionalUTF8StringQuery(c, "title")
		require.NoError(t, err)
		assert.Equal(t, "встреча 🚀", got)
	})

	t.Run("rejects control characters", func(t *testing.T) {
		c, _ := gin.CreateTestContext(httptest.NewRecorder())
		c.Request = httptest.NewRequest(http.MethodGet, "/?title=a%0Ab", nil)

		got, err := ValidateOptionalUTF8StringQuery(c, "title")
		require.Error(t, err)
		assert.Empty(t, got)
	})

	t.Run("empty is ok", func(t *testing.T) {
		c, _ := gin.CreateTestContext(httptest.NewRecorder())
		c.Request = httptest.NewRequest(http.MethodGet, "/", nil)

		got, err := ValidateOptionalUTF8StringQuery(c, "title")
		require.NoError(t, err)
		assert.Empty(t, got)
	})
}
