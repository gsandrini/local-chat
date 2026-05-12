# Release

Releases are automated via GitHub Actions.

The pipeline builds for the supported platforms and uploads all artifacts to a GitHub Release.

---

## How to release

1. Make sure `main` is clean and tested
2. Push a version tag:

```bash
git tag v1.0.0
git push origin v1.0.0
```

The pipeline triggers automatically and within a few minutes

The release appears on the GitHub Releases page with the binaries for all supported platforms.

---

## Versioning

Follow [Semantic Versioning](https://semver.org/):

- `v1.0.0` - stable release
- `v1.1.0` - new features, backwards compatible
- `v1.0.1` - bug fixes only
- `v2.0.0` - breaking changes

---

## GitHub Actions workflow

The workflow file is at `.github/workflows/release.yml`.

It handles Go setup, system dependencies and the Wails build for each target platform.

No secrets or signing certificates are required.
