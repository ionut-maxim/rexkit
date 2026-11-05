# rexkit

rexkit is a design system for terminal apps that stay open in a tab, used from a phone as much as from a desktop. It's built on [Charm v2](https://charm.land): Bubble Tea, Bubbles and Lip Gloss.

It came out of a terminal UI for Linear. Its first target is the Rex terminal, whose iOS client has three layouts on a foldable phone. The packages work in any modern terminal, and the kitty graphics parts work in any terminal that speaks the kitty graphics protocol.

```sh
go get github.com/ionut-maxim/rexkit
```

## Packages

| Package | What it gives you |
|---|---|
| `theme` | A dark and a light palette, `Set(dark)` when the terminal reports its background, and the shared styles. |
| `icons` | Nerd Font icons, and plain Unicode fallbacks for terminals without a Nerd Font (`Pick`). |
| `components` | Layout classes for three widths (`SizeOf`: compact under 64 columns, regular up to 119, wide from 120). Buttons with rounded ends that can be tall (`Btn`, `RenderSegs`), pills, chips that wrap (`JoinChips`), section headers, progress bars, priority bars, card surfaces (`FillBg`), OSC 8 link buttons and list rows. |
| `touch` | Tap zones and hit tests. Key hints drawn as buttons, with a "? more" button (`Hints`). Taps that send the same key a press would (`KeyMsg`), so keys and taps share one code path. List row clicks and wheel scrolling. |
| `kitty` | Kitty graphics with Unicode placeholders. An image is sent once and drawn as text cells, so it scrolls and clips with the view. Also a capability check (`Query`), sizing (`Fit`), and sharp images on high-density screens (`TransmitSharp`). |
| `bigtext` | Big headings and tall image buttons drawn as kitty images, for terminals that can't draw bigger text. JetBrains Mono Bold is embedded. |
| `markdown` | GitHub-flavoured markdown drawn in the app's palette. Code is highlighted with Chroma, tables are grids or one card per row, links are OSC 8, and images are split out for you to draw. |
| `rexstatus` | Rex's Program Status Protocol (OSC 7501): a spinner, done or error mark on the app's tab while you're elsewhere. |

| `shell` | The app frame. It draws a header with a tab for each view and breadcrumbs, the screen on top, and a footer with the status line and key hints as buttons. Each view keeps its own stack of screens, so switching away and back returns to where you were. It probes the terminal (background colour, kitty graphics, cell size), counts running jobs for Rex's tab status, and focuses on hover with a desktop pointer. Apps plug in their views, global keys, a loader, and a `Hook` that sees every message first. |

## Start an app

`examples/hello` is the smallest app: two views as tabs, a list of cards you can tap, and a detail screen with markdown. Copy it to start a new one.

```sh
go run github.com/ionut-maxim/rexkit/examples/hello
```

## Design rules

- **Touch first.** Anything you can tap looks like a button and has a tap zone. A tap shows pressed for a moment (`PressDelay`) before its action runs.
- **Three widths.** Lay out for compact, regular and wide. On a folded phone that means one thing per line.
- **Tall screens.** With 40 rows or more, header and footer buttons get an empty row above and below that's part of their tap zone. Don't draw tall buttons with half blocks: they leave seams on iOS. For a real tall button, use an image (`bigtext.Strip`).
- **Images as text.** Use kitty placeholders only, never direct placements. Make images after an update, never in `View`, and delete them when the screen closes.
- **Links open where you tap.** Use OSC 8 link buttons for things to open, not app actions.

## Licence

MIT, see `LICENSE`. JetBrains Mono is under the SIL Open Font License 1.1 (`bigtext/fonts/OFL.txt`).
