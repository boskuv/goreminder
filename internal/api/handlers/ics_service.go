package handlers

import (
	"context"
	"io"

	"github.com/boskuv/goreminder/internal/service"
)

// ICSImporter is the use-case surface used by ICSHandler.
type ICSImporter interface {
	Import(ctx context.Context, r io.Reader, opts service.ICSImportOptions) (*service.ICSImportResult, error)
}
