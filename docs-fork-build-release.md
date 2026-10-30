# Fork Build and Release

Node Fork build infrastructure is isolated from upstream:

| Purpose | Fork file |
| --- | --- |
| Node image | `Dockerfile.fork` |
| Build/test/release | `.github/workflows/fork-ci.yml` |
| Installation and xbctl deployment | `install-fork.sh` |
| Fork build targets | `Makefile.fork` |

The official `Dockerfile`, `Makefile`, `ci.yml`, and `install.sh` stay close to
upstream. The upstream CI workflow is manual-only in this fork so branch and tag
pushes cannot publish an official-configuration image into the Fork namespace.

Local SGP verification:

```bash
make -f Makefile.fork test
make -f Makefile.fork build-linux
make -f Makefile.fork build-linux-arm64
docker build -f Dockerfile.fork -t xboard-node-local:test .
```

Fork installations should use:

```bash
sudo bash install-fork.sh
```

Release only after a verified batch has passed local SGP builds, canary tests,
and the full Fork CI.
