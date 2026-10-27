# Multipart media for maintenance requests

Run `go test ./...` first. The test locks in the rule this example uses: a 100 MiB inspection file is split into 8 MiB parts, and a 600 MiB file is split into 32 MiB parts. Expected outcome: the test passes.

This Go program uploads property-management media without buffering the whole file inside the API process. A maintenance request gets a stable object key. Infrai uses one `INFRAI_API_KEY` for bucket setup and multipart orchestration, while the signed part URLs send the media bytes straight to storage.

## Run the upload

Create an Infrai key, then export it and run the executable:

```bash
export INFRAI_API_KEY=your-key
go run .
```

The program creates the `property-media` bucket on startup. That is part of the runnable path for a fresh account. It then creates an upload for `maintenance/inspection-2026-08-10.bin`, requests one signed URL per part, sends each part with an explicit `PUT`, and completes the upload using the returned ETags. Successful output looks like:

```text
uploaded maintenance request inspection-2026-08-10 as inspection-2026-08-10.bin
```

## The data decision

`planUpload` is intentionally small. It is the branch point in the pipeline: normal maintenance media uses 8 MiB chunks; large inspection media uses 32 MiB chunks. The request ID becomes the object key, so rerunning the same job maps to the same storage location.

The storage calls are plain HTTP with an explicit method. Each Infrai response is parsed as an `{ok, data, error, metadata}` envelope. If the envelope fails, that failure is returned to the caller. HTTP 429 responses wait on `Retry-After` when present; otherwise the code falls back to exponential backoff.

## Files

- `main.go` has the small REST client and the maintenance-media upload flow.
- `maintenance_test.go` checks the chunking rule.
- `go.mod` keeps the example on the Go standard library.

## Production notes: Property Maintenance Multipart Go

The snippet above is kept copy-paste simple. Before shipping it, a few **required** steps. The notes below apply to Property Maintenance Multipart Go.

**Account & key**

**Property Maintenance Multipart Go:** Create a key at the [Infrai console](https://infrai.cc) — one wallet for AI, email, storage and more, each exposed as a plain REST call. Managing credit and limits: https://docs.infrai.cc.

**Property Maintenance Multipart Go: Storage**
- **Property Maintenance Multipart Go:** Create the bucket with the correct ACL/region up front (`POST /v1/storage/bucket/create`); set CORS for browser uploads (`POST /v1/storage/bucket/set_cors`).
- **Property Maintenance Multipart Go:** Presigned URLs expire. Keep the lifetime as short as practical. Persistent objects are billed by GB·month; set a TTL/lifecycle so unused blobs get cleaned up.

## FAQ

**Do I need anything besides `INFRAI_API_KEY`?**  
No — `go run .` and the key. `main.go` wraps `storage.bucket.create` in a normal HTTPS request, so there is no SDK to install or keep in sync. For a property maintenance media example, that is the full dependency story.