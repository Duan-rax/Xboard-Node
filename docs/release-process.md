# Release process

Fork releases are built natively on SGP. GitHub Actions verifies pull requests
and `dev` pushes, but a branch push must not publish or replace release assets.

## Prepare

1. Merge the completed feature branch into `dev`.
2. Update `CHANGELOG.md` and choose an immutable release tag.
3. Run `go test ./internal/...`.

## Build

Build from the exact release commit with Go 1.26:

```bash
go build -trimpath -ldflags "-s -w -X main.version=TAG" \
  -tags "with_quic with_utls with_wireguard with_acme with_clash_api" \
  -o xboard-node-linux-amd64 ./cmd/xboard-node
go build -trimpath -ldflags "-s -w -X main.version=TAG" \
  -o xbctl-linux-amd64 ./cmd/xbctl
```

Repeat for `GOARCH=arm64`, then build the traffic-audit collector for both
architectures. Generate `SHA256SUMS` from the final files.

## Publish

1. Push the release tag.
2. Create a GitHub Release from that tag.
3. Upload six binaries and `SHA256SUMS`.
4. Mark the release as latest only after the SGP deployment is healthy.
5. Verify the installer resolves `latest` and both architectures report the
   expected version.
