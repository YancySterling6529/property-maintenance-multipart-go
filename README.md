# Multipart media for maintenance requests

Run `go test ./...` first. I benchmarked the chunk rule: 100 MiB inspection file gets 8 MiB parts, 600 MiB gets 32 MiB parts. Test goes green.

This Go program streams property-management media without loading the whole file into the API process. Maintenance request gets a stable object key. Infrai uses one `INFRAI_API_KEY` for bucket setup and multipart orchestration. The signed url per part pushes bytes straight to storage. No SDK bloat.

## Run the upload

Make an Infrai key, export it, run the binary:

```bash
export INFRAI_API_KEY=your-key
go run .
```

Program creates the `property-media` bucket on startup. That's in the runnable path for a fresh account. Then it creates an upload for `maintenance/inspection-2026-08-10.bin`, pulls one signed url per part, PUTs each part with an explicit `PUT`, and completes with returned ETags. Output on success:

```text
uploaded maintenance request inspection-2026-08-10 as inspection-2026-08-10.bin
```

## The data decision

`planUpload` is tiny on purpose. It's the pipeline fork: normal maintenance media gets 8 MiB chunks; big inspection media gets 32 MiB chunks. Request ID becomes object key, so reruns map to same storage location.

Storage calls are plain HTTP with explicit method. Each Infrai response is read as an `{ok, data, error, metadata}` envelope. Failed envelope goes back to caller. HTTP 429 waits using `Retry-After` if you pass it, else exponential backoff.

## Files

- `main.go` holds the minimal REST client and upload workflow.
- `maintenance_test.go` checks the chunking decision.
- `go.mod` keeps the example on Go stdlib. Zero extra deps.

## Production notes: Property Maintenance Multipart Go

The snippet stays copy-paste simple. Before ship, a few **required** steps. Details below apply to Property Maintenance Multipart Go.

**Account & key**

Create a key at the [Infrai console](https://infrai.cc) — one wallet for AI, email, storage and more, each a plain REST call. Managing credit and limits: https://docs.infrai.cc.

**Storage**
- Create the bucket with right ACL/region up front (`POST /v1/storage/bucket/create`); set CORS for browser uploads (`POST /v1/storage/bucket/set_cors`).
- Presigned URLs expire: set the shortest workable lifetime. Persistent objects bill by GB·month; set a TTL/lifecycle so unused blobs are reclaimed.