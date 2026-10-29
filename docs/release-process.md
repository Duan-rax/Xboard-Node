# Release process

Fork releases are built and published by GitHub Actions when an immutable
`v*` tag is pushed. Branches only run verification; they never publish release
assets or replace `latest`.

## Prepare

1. Merge the completed feature branch into `dev`.
2. Update `CHANGELOG.md` and choose an immutable release tag.
3. Run `go test ./internal/...`.

## Publish

1. Push the release tag.
2. Actions runs tests, builds six binaries, generates `SHA256SUMS`, builds a
   multi-architecture container and creates the GitHub Release.
3. The tag also publishes `ghcr.io/duan-rax/xboard-node:TAG` and `latest`.
4. Deploy the produced image or binaries and verify all affected nodes.
