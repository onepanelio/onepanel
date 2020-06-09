# Helper scripts

## gen-release-md.go
Generates markdown for releases.

Usage:
```bash
go run cmd/gen-release-md/gen-release-md.go -v=0.10.0 -u=[github-username] > /tmp/release.md
```

## goose.go
Supports both Go and SQL migrations.

```bash
go run cmd/goose/goose up     # run up migrations
go run cmd/goose/goose down   # run down migrations
```