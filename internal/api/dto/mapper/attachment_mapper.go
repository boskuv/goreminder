package mapper

import (
	"github.com/boskuv/goreminder/internal/api/dto"
	"github.com/boskuv/goreminder/pkg/attachments"
)

// AttachmentsToResponse maps attachment client models to API DTOs.
// Returns nil when empty so JSON omitempty omits the field.
func AttachmentsToResponse(list []attachments.Attachment) []dto.AttachmentResponse {
	if len(list) == 0 {
		return nil
	}
	out := make([]dto.AttachmentResponse, 0, len(list))
	for i := range list {
		a := &list[i]
		out = append(out, dto.AttachmentResponse{
			ID:           a.ID,
			TaskID:       a.TaskID,
			OriginalName: a.OriginalName,
			ContentType:  a.ContentType,
			SizeBytes:    a.SizeBytes,
			Status:       a.Status,
			CreatedAt:    a.CreatedAt,
		})
	}
	return out
}
