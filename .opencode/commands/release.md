# Release

Create a new cortex release.

## Process

1. **Pre-flight checks**
   ```bash
   git status  # Must be on main, clean working tree
   make test
   make lint
   ```

2. **Determine version** (auto-increment from last tag)
   ```bash
   git describe --tags --abbrev=0  # Current version
   # Increment: patch (0.1.0 → 0.1.1), minor (0.1.0 → 0.2.0), major (0.1.0 → 1.0.0)
   ```

3. **Build release artifacts**
   ```bash
   make release-build
   ls -la dist/  # Verify 8 binaries + checksums.txt
   ```

4. **Create and push tag**
   ```bash
   git tag -a v<VERSION> -m "Release v<VERSION>"
   git push origin v<VERSION>
   ```

5. **Create GitHub release with auto-generated notes**
   ```bash
   gh release create v<VERSION> ./dist/cortex-* ./dist/cortexd-* ./dist/checksums.txt ./install.sh \
     --title "v<VERSION>" \
     --generate-notes
   ```

6. **Edit release notes** (if needed)
   ```bash
   gh release edit v<VERSION> --notes-file -  # Opens editor
   ```

7. **Test in Docker**
   ```bash
   docker run --rm -it ubuntu:22.04 bash -c '
     apt-get update && apt-get install -y curl git tmux ca-certificates
     curl -fsSL https://github.com/kareemaly/cortex/releases/download/v<VERSION>/install.sh | bash
     export PATH="$HOME/.local/bin:$PATH"
     cortex version
     cortexd version
     mkdir /tmp/test && cd /tmp/test && git init
     cortex init --global-only
     timeout 3 cortexd serve || echo "daemon OK"
   '
   ```

8. **Verify release**
   ```bash
   gh release view v<VERSION>
   ```

9. **Verify macOS notarization**
   ```bash
   gh release download v<VERSION> -p 'cortex-darwin-arm64' -p 'cortexd-darwin-arm64'
   chmod +x cortex-darwin-arm64 cortexd-darwin-arm64
   spctl --assess --type open --context context:primary-signature -vv ./cortex-darwin-arm64
   spctl --assess --type open --context context:primary-signature -vv ./cortexd-darwin-arm64
   ```
   Both must print `accepted` with `source=Notarized Developer ID`. (Use `--type open`, not `-t execute` — the latter is for `.app` bundles and rejects bare CLI binaries with "not an app" even when properly signed.) If either fails, hold the release — see Rollback. Notarization is mandatory; do not ship unsigned.

## Version Guidelines

- **patch** (0.1.x): Bug fixes, minor improvements
- **minor** (0.x.0): New features, backward compatible
- **major** (x.0.0): Breaking changes

## Rollback

If release is broken:
```bash
gh release delete v<VERSION> --yes
git push --delete origin v<VERSION>
git tag -d v<VERSION>
```

If the workflow fails at the signing/notarization step, the failed run's `Cleanup signing inputs` step prints `quill.log` (collapsed group) — that has the Apple submission ID and rejection reason. Fix forward, delete + recreate the tag.
