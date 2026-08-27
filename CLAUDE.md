# CLAUDE.md

Guidance for Claude Code when working in this repository.

## Building and publishing a custom/dev image

The provider is a **crossplane package (xpkg)**, not a plain container image. Crossplane's
package manager (a `Provider` resource referencing the image) parses the OCI artifact expecting
a `package.yaml` manifest inside it. A plain Docker image built and pushed via
`cluster/images/crossplane-provider-ionoscloud`'s `img.build`/`docker push` does **not** have
this manifest and fails to install with an error like:

```
cannot initialize parser backend: couldn't find "package.yaml" file after checking N files in the archive
```

That target's own Makefile says as much (`img.publish: ... Publish is deferred to xpkg
machinery`) — `img.build` only produces the intermediate runtime image the xpkg build later
wraps; it is not the installable artifact on its own. This was discovered the hard way: a
manually-built-and-pushed `docker push` image installed fine as a plain container reference but
failed as a crossplane `Provider` package with exactly the error above.

### Correct build sequence

```bash
export BUILD_ARGS=--load   # see "the --load gotcha" below
make build
```

Without `BUILD_ARGS=--load` exported, `make build` fails partway through with:
```
crossplane: error: failed to get runtime base image options: failed to pull runtime image: Error response from daemon: No such image: build-<hash>/crossplane-provider-ionoscloud-amd64:latest
```
This is because the internal `img.build` step (see `build/makelib/imagelight.mk`) runs
`docker buildx build` with the `docker-container` driver and no `--load`/`--push`, so the
resulting image never lands in the local Docker daemon's image store — it stays in BuildKit's
own build cache only. The subsequent `xpkg.build` step then tries to reference that image by
name (`docker pull`) and finds nothing. Exporting `BUILD_ARGS=--load` as an environment variable
(not just a Makefile command-line var) propagates into the recursive `$(MAKE)` call that builds
the runtime image, so it actually gets loaded into the daemon and the xpkg step can find it.

This produces `_output/xpkg/linux_amd64/<pkg-name>-<version>.xpkg` — the real installable
artifact, where `<version>` is auto-derived from `git describe` (e.g.
`crossplane-provider-ionoscloud-v1.2.6-5.gc8c1404.xpkg` for the 5th commit past the `v1.2.6` tag,
short SHA `c8c1404`).

### Correct push mechanism

Push the `.xpkg` file with the `crossplane` CLI's own `xpkg push`, **not** `docker push`:

```bash
CLI=".cache/tools/linux_x86_64/crossplane-cli-v2.0.2"   # exact version varies; see CROSSPLANE_CLI in build/makelib/k8s_tools.mk
"$CLI" xpkg push --package-files _output/xpkg/linux_amd64/<pkg-name>-<version>.xpkg <registry>/<repo>:<tag>
```

`docker login <registry>` beforehand is sufficient for auth — `xpkg push` reuses the standard
Docker/OCI credential store, no separate crossplane-specific login step needed. Do not `docker
push` the plain `img.build` output as a substitute for this — see above.

The `make publish.artifacts` / `xpkg.release.publish.*` targets (see `build/makelib/xpkg.mk`) do
exactly this in CI, gated to `main`/`master`/`release-*` branches only — this is the reference
for the exact invocation shape if the manual command above ever needs re-deriving.
