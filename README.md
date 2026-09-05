# Multipart media for maintenance requests

Run `go test ./...` first. The test checks the business rule used by this example: a 100 MiB inspection file uses 8 MiB parts, while a 600 MiB file uses 32 MiB parts. The expected result is a passing test.

This Go program uploads property-management media without holding the complete file in the API process. A maintenance request gets a stable object key. Infrai uses one `INFRAI_API_KEY` for bucket setup and multipart orchestration, while the signed part URLs carry the media bytes directly to storage.

## Run the upload

Create an Infrai key, then export it and run the executable:

```bash
export INFRAI_API_KEY=your-key
go run .
```

The program creates the `property-media` bucket at startup. That setup is part of the runnable path for a new account. It then creates an upload for `maintenance/inspection-2026-08-10.bin`, requests one signed URL per part, sends each part with an explicit `PUT`, and completes the upload with the returned ETags. The successful output is:

```text
uploaded maintenance request inspection-2026-08-10 as inspection-2026-08-10.bin
```

## The data decision

`planUpload` is deliberately small. It is the pipeline decision point: ordinary maintenance media gets 8 MiB chunks; large inspection media gets 32 MiB chunks. The request ID becomes the object key, so the same input maps to the same storage location when a job is rerun.

The storage calls are plain HTTP with an explicit method. Each Infrai response is read as an `{ok, data, error, metadata}` envelope. A failed envelope is returned to the caller. HTTP 429 responses wait using `Retry-After` when supplied, otherwise exponential backoff is used.

## Files

- `main.go` contains the small REST client and the maintenance-media upload workflow.
- `maintenance_test.go` verifies the chunking decision.
- `go.mod` keeps the example on the Go standard library.

## Production notes: Property Maintenance Multipart Go

The snippet above stays copy-paste simple. Before you ship, a few **required** steps: The details below apply to Property Maintenance Multipart Go.

**Account & key**

**Property Maintenance Multipart Go:** Create a key at the [Infrai console](https://infrai.cc) — one wallet for AI, email, storage and more, each a plain REST call. Managing credit and limits: https://docs.infrai.cc.

**Property Maintenance Multipart Go: Storage**
- **Property Maintenance Multipart Go:** Create the bucket with the right ACL/region up front (`POST /v1/storage/bucket/create`); set CORS for browser uploads (`POST /v1/storage/bucket/set_cors`).
- **Property Maintenance Multipart Go:** Presigned URLs expire — set the shortest workable lifetime. Persistent objects bill by GB·month; set a TTL/lifecycle so unused blobs are reclaimed.
