# Releasing lazyncu

Releases are fully automated: pushing a `v*` tag runs the `release` GitHub
Actions workflow, which tests the code and runs [goreleaser](https://goreleaser.com)
to publish binaries, checksums, a changelog, and a Homebrew cask.

## One-time setup (required before the first release)

The pipeline pushes a Homebrew cask to a separate tap repository. That needs a
repo and a token, created once by hand.

### 1. Create the tap repository

Create a **public** repository named exactly `homebrew-tap` under the `luchrv`
account (Homebrew resolves `luchrv/tap` to `luchrv/homebrew-tap`):

```sh
gh repo create luchrv/homebrew-tap --public \
  --description "Homebrew tap for luchrv tools" \
  --add-readme
```

Or via web: <https://github.com/new> → name `homebrew-tap` → Public →
initialize with a README (the default branch must exist). Nothing else is
needed — goreleaser creates `Casks/lazyncu.rb` on the first release.

### 2. Create a fine-grained PAT

The workflow's default `GITHUB_TOKEN` cannot push to other repositories, so a
personal access token scoped to the tap is required.

1. Go to <https://github.com/settings/personal-access-tokens/new>.
2. **Token name**: `goreleaser-homebrew-tap`.
3. **Expiration**: your call (max allowed recommended; note the date — renew before it lapses).
4. **Resource owner**: `luchrv`.
5. **Repository access**: *Only select repositories* → `luchrv/homebrew-tap`.
6. **Permissions → Repository permissions → Contents**: *Read and write*
   (Metadata: read-only is added automatically). Nothing else.
7. Generate and **copy the token now** — it is shown only once.

### 3. Store the token as an Actions secret on lazyncu

```sh
gh secret set HOMEBREW_TAP_TOKEN --repo luchrv/lazyncu
# paste the token when prompted
```

Or via web: `luchrv/lazyncu` → Settings → Secrets and variables → Actions →
*New repository secret* → name `HOMEBREW_TAP_TOKEN`, value = the PAT.

### 4. Verify

- `gh repo view luchrv/homebrew-tap` shows the public repo.
- `gh secret list --repo luchrv/lazyncu` lists `HOMEBREW_TAP_TOKEN`.

## Cutting a release

1. Make sure `main` is green: `make check`.
2. Optionally validate the pipeline locally: `make release-check`
   (needs `brew install goreleaser`; builds everything into `dist/` without publishing).
3. Tag and push:

   ```sh
   git tag v0.1.0
   git push origin main --tags
   ```

4. Watch the workflow: `gh run watch --repo luchrv/lazyncu`.

The result: a GitHub Release with six archives (linux/darwin/windows ×
amd64/arm64), `checksums.txt`, a changelog grouped from conventional commits,
and an updated cask in the tap. Users install with:

```sh
brew install luchrv/tap/lazyncu
```

## Troubleshooting

- **Cask push fails (401/403)**: the PAT expired or lacks `Contents:
  read/write` on `luchrv/homebrew-tap`. Fix the token, update the secret, then
  re-run the failed job — the release itself is already published and the cask
  step is safe to retry.
- **Need to redo a release**: delete the GitHub Release and the tag
  (`git push origin :refs/tags/v0.1.0`), fix, re-tag, push again.
- **Changelog looks wrong**: it is grouped from conventional-commit prefixes;
  `chore`, `docs`, `test`, and `ci` commits are excluded by config.
