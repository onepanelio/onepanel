# Helper scripts

## gen-release-md.go
Generates markdown for releases.

Usage:
```bash
go run cmd/gen-release-md.go -v=0.10.0 > /tmp/release.md
```

## goose.go
Supports both Go and SQL migrations.

```bash
go run cmd/goose up     # run up migrations
go run cmd/goose down   # run down migrations
```