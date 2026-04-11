# Review Explorer Containment Validation

Containment fix commit: `ef1b89a`

This note records the direct 220px-width validation used to close the final
review explorer containment repair for
`2026-04-11-stabilize-review-ui-layout`.

## Browser Evidence

- Explorer row screenshot: `output/playwright/review-explorer-row-220.png`
- Header metadata screenshot: `output/playwright/review-header-meta.png`

Both screenshots were refreshed after commit `ef1b89a` using the rebuilt
embedded `harness ui` binary with `harness-ui:explorer-width:review=220` in
browser local storage.

## DOM Measurements

The direct browser probe against the selected review row at the 220px explorer
width reported:

- button: `clientWidth=219`, `scrollWidth=219`
- main column: `clientWidth=182`, `scrollWidth=182`
- title row: `clientWidth=182`, `scrollWidth=182`
- subtitle row: `clientWidth=182`, `scrollWidth=182`
- metadata cell: `clientWidth=151`, `scrollWidth=151`
- title text: `clientWidth=182`, `scrollWidth=241`

Interpretation:

- the row container and subtitle layout are fully contained at the minimum
  explorer width because their `clientWidth` and `scrollWidth` values match
- the long title still overflows its own text box, but that overflow is now
  handled by ellipsis inside the title cell rather than by pushing the row
  wider than the explorer item

## Supporting Checks

- `pnpm --dir web check`
- `pnpm --dir web build`
- `git diff --check`
