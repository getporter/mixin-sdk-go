# Versioning and Compatibility

`mixin-sdk-go` follows plain Go module semver via git tags — no extra
tooling, just what `go get`/`go mod` already assume.

## Pre-1.0

The SDK is currently `v0.x.y`. Per semver's own pre-1.0 rule, a breaking
change to the `Mixin` interface or any other exported API can land in a
minor release (`v0.4.0`, not just a hypothetical `v1.0.0`). If you're
depending on the SDK:

- Pin an exact version once your mixin ships, rather than tracking
  `@latest`. `mixin-init` scaffolds a pinned version by default and
  prints a reminder of this after scaffolding.
- Review the diff (`go get -u github.com/getporter/mixin-sdk-go` and
  check what changed) before bumping, rather than upgrading blindly.

## Reaching v1.0.0

Once the `Mixin` interface has been validated by real-world mixin usage
and feels settled — not likely to need another breaking change soon —
this SDK cuts `v1.0.0`. That's the signal that the API is stable enough
to build a mixin against without expecting to revisit it every upgrade.

## After v1.0.0

Breaking changes still happen eventually, but not without warning:

- Anything slated for removal is marked `// Deprecated:` in its godoc
  comment for at least one release before it's actually removed. Go
  tooling (govulncheck, IDEs, `go vet`) surfaces `// Deprecated:` markers
  automatically, so this isn't a silent signal — it shows up in your
  editor and CI.
- Whether a breaking change after `v1.0.0` requires a new major version
  and Go's `/v2` import-path convention is not decided yet. This will be
  settled if and when it's actually needed, not preemptively.

## Porter compatibility

A given Porter release may require a newer `mixin-sdk-go` version — for
example, if the mixin wire protocol itself changes. If that happens, it
follows the same policy as any other breaking change above: it's called
out explicitly in that release, not required silently. A mixin built
against an older SDK version isn't at risk of quietly breaking against a
newer Porter release without that being documented somewhere you'd see
it (the SDK's release notes).

## See also

- [README's Versioning section](../README.md#versioning) — the short
  version of this doc.
- [docs/tutorial.md](./tutorial.md) — building a mixin against the
  current SDK.
