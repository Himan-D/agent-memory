## Summary

<!-- What changed and why (1-3 sentences) -->

## Type

- [ ] feat — new feature
- [ ] fix — bug fix
- [ ] chore — tooling, deps, repo hygiene
- [ ] docs — documentation only
- [ ] ci — GitHub Actions / automation

## Testing

```bash
go build ./...
go test -short ./...
```

- [ ] Built locally
- [ ] Tested affected endpoints/components

## Deploy impact

- [ ] Landing (`hystersis.com`)
- [ ] Docs (`docs.hystersis.com` / `/docs`)
- [ ] API (`api.hystersis.ai`)
- [ ] Dashboard (`app.hystersis.com`)
- [ ] None

## Agent notes

Cloud Agent PRs on `cursor/*` branches are auto-labeled `automerge` and squash-merged when **CI Success** passes.

Add label `do-not-merge` to block auto-merge.
