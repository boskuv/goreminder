package handlers_test

import (
	"bytes"
	"context"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/boskuv/goreminder/internal/api/handlers"
	errs "github.com/boskuv/goreminder/internal/errors"
	"github.com/boskuv/goreminder/internal/service"
	"github.com/boskuv/goreminder/pkg/logger"
)

type stubICSImporter struct {
	importFn func(ctx context.Context, r io.Reader, opts service.ICSImportOptions) (*service.ICSImportResult, error)
}

func (s *stubICSImporter) Import(ctx context.Context, r io.Reader, opts service.ICSImportOptions) (*service.ICSImportResult, error) {
	if s.importFn == nil {
		panic("unexpected Import")
	}
	return s.importFn(ctx, r, opts)
}

func TestICSHandler_Import_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)
	log := logger.New(io.Discard, zerolog.DebugLevel, false)

	var gotOpts service.ICSImportOptions
	var gotBody string
	h := handlers.NewICSHandler(&stubICSImporter{
		importFn: func(_ context.Context, r io.Reader, opts service.ICSImportOptions) (*service.ICSImportResult, error) {
			gotOpts = opts
			b, err := io.ReadAll(r)
			require.NoError(t, err)
			gotBody = string(b)
			return &service.ICSImportResult{
				CreatedTaskIDs: []int64{10, 11},
				SkippedTODO:    1,
				Errors: []service.ICSImportItemError{
					{Index: 0, UID: "bad", Message: "no start"},
				},
			}, nil
		},
	}, log)

	body := &bytes.Buffer{}
	w := multipart.NewWriter(body)
	part, err := w.CreateFormFile("file", "sample.ics")
	require.NoError(t, err)
	_, err = io.WriteString(part, "BEGIN:VCALENDAR\nEND:VCALENDAR\n")
	require.NoError(t, err)
	require.NoError(t, w.WriteField("group_id", "5"))
	require.NoError(t, w.WriteField("messenger_related_user_id", "3"))
	require.NoError(t, w.Close())

	req := httptest.NewRequest(http.MethodPost, "/api/v1/users/42/ics/import", body)
	req.Header.Set("Content-Type", w.FormDataContentType())
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = req
	c.Params = gin.Params{{Key: "user_id", Value: "42"}}

	h.Import(c)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, int64(42), gotOpts.UserID)
	require.NotNil(t, gotOpts.GroupID)
	assert.Equal(t, int64(5), *gotOpts.GroupID)
	require.NotNil(t, gotOpts.MessengerRelatedUserID)
	assert.Equal(t, 3, *gotOpts.MessengerRelatedUserID)
	assert.Contains(t, gotBody, "BEGIN:VCALENDAR")
	assert.Contains(t, rec.Body.String(), `"created_task_ids":[10,11]`)
	assert.Contains(t, rec.Body.String(), `"skipped_todo":1`)
	assert.Contains(t, rec.Body.String(), `"count":2`)
}

func TestICSHandler_Import_MissingFile(t *testing.T) {
	gin.SetMode(gin.TestMode)
	log := logger.New(io.Discard, zerolog.DebugLevel, false)
	h := handlers.NewICSHandler(&stubICSImporter{}, log)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/users/1/ics/import", strings.NewReader(""))
	req.Header.Set("Content-Type", "multipart/form-data; boundary=x")
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = req
	c.Params = gin.Params{{Key: "user_id", Value: "1"}}

	h.Import(c)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestICSHandler_Import_ValidationError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	log := logger.New(io.Discard, zerolog.DebugLevel, false)
	h := handlers.NewICSHandler(&stubICSImporter{
		importFn: func(context.Context, io.Reader, service.ICSImportOptions) (*service.ICSImportResult, error) {
			return nil, errs.ErrValidation
		},
	}, log)

	body := &bytes.Buffer{}
	w := multipart.NewWriter(body)
	part, err := w.CreateFormFile("file", "bad.ics")
	require.NoError(t, err)
	_, err = io.WriteString(part, "nope")
	require.NoError(t, err)
	require.NoError(t, w.Close())

	req := httptest.NewRequest(http.MethodPost, "/api/v1/users/1/ics/import", body)
	req.Header.Set("Content-Type", w.FormDataContentType())
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = req
	c.Params = gin.Params{{Key: "user_id", Value: "1"}}

	h.Import(c)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}
