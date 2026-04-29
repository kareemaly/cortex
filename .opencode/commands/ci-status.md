# CI Status

Fetch GitHub Actions pipeline status for kareemaly/cortex.

## Get Latest Run Status

```bash
gh run list --repo kareemaly/cortex --limit 1
```

## Get Failed Logs

```bash
# Get run ID first, then fetch failed logs
gh run view <run-id> --repo kareemaly/cortex --log-failed
```

Or in one command:
```bash
gh run view --repo kareemaly/cortex $(gh run list --repo kareemaly/cortex --limit 1 --json databaseId --jq '.[].databaseId') --log-failed
```

## Get Job-Level Status

```bash
gh run view <run-id> --repo kareemaly/cortex --json jobs --jq '.jobs[] | "\(.name): \(.conclusion)"'
```
