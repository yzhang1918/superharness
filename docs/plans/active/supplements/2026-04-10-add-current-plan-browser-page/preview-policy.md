# Preview Policy Notes

The first `Plan` page preview policy is intentionally narrow so the browser can
be predictable before it becomes broad.

## Supported Rich Preview

- `md`
- `txt`
- `json`
- `yaml`
- `yml`

## Plain-Text Fallback

Text-readable files outside the richer allowlist should still open as plain
text when they are small enough and look non-binary.

## Explicitly Unsupported

- image files
- binary files
- oversized files beyond the initial preview threshold

The UI should say why preview is unavailable instead of silently hiding the
file or pretending the content loaded cleanly.
