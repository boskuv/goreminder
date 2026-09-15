# API contracts (goreminder)

## Attachments gRPC

| Path | Purpose |
|------|---------|
| `proto/attachments/v1/attachments.proto` | Source of truth — keep in sync with the attachments service repository |
| `gen/attachments/v1/attachments.pb.go` | Generated message types (`Attachment`, `InitUploadRequest`, …) — `protoc-gen-go` |
| `gen/attachments/v1/attachments_grpc.pb.go` | Generated gRPC client/server (`AttachmentServiceClient`, `AttachmentServiceServer`) — `protoc-gen-go-grpc` |

The monorepo ships **contract + client stubs** only. The attachment **service implementation** (S3, attachment DB, outbox worker) lives in a separate repository.

### Regenerate

```bash
make proto-attachments        # local protoc
make proto-attachments-docker # Docker, no local protoc
```

### Import in code

```go
import attachmentsv1 "github.com/boskuv/goreminder/api/gen/attachments/v1"
```

Core wraps the gRPC client in `pkg/attachments` (domain types, timeouts, error mapping).

### RPC surface (`AttachmentService`)

| RPC | Used by core for |
|-----|------------------|
| `InitUpload` | Presigned upload init (`POST` JSON on `.../attachments`) |
| `UploadDirect` | Multipart direct upload (small files, `ready` immediately) |
| `CompleteUpload` | After client `PUT` to object storage |
| `ListAttachments` | `GET .../attachments`, `GET /tasks/{id}` detail |
| `GetDownloadURL` | `GET .../download` presigned URL |
| `DownloadDirect` | `GET .../content` proxy download |
| `DeleteAttachment` | `DELETE .../attachments/{id}` |
| `PurgeByTask` / `PurgeByUser` | Best-effort purge on task/user soft-delete |

### Workflow (client → core → attachment service)

```
Client                    Core (REST)              Attachment service (gRPC + S3)
  |                          |                              |
  |-- POST JSON ------------>|-- InitUpload --------------->|
  |<-- upload_url, pending --|<------------------------------|
  |-- PUT file to S3 ---------------------------------------->|
  |-- POST complete -------->|-- CompleteUpload ------------>|
  |<-- ready ----------------|<------------------------------|

  |-- POST multipart file -->|-- UploadDirect -------------->|
  |<-- ready (no complete) -|<------------------------------|
```
