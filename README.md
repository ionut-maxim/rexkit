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
| `lists` | Lists that scroll like a touch screen: a swipe (the wheel) moves the page and leaves the selection alone, a key brings the selection back into view, a tap selects and a tap on the selected row opens it. `Scroll` is the state for a list you draw yourself; `Rows` is a ready list of `components.Row` with a `/` filter, to use instead of a Bubbles list. |
| `bigtext` | Tall image buttons (`Strip`) drawn as kitty images, for terminals that can't draw bigger text, in the markdown library's JetBrains Mono Bold. |
| `rexstatus` | Rex's Program Status Protocol (OSC 7501): a spinner, done or error mark on the app's tab while you're elsewhere. |
| `form` | A screen that asks for a few lines of text: labelled fields you tap to focus, a hint and a check per field, Submit and Cancel buttons. It lays out for the three widths and tall screens. Submitting runs your command; answer with `form.Done` to close it or show why it failed. |
| `shell` | The app frame. It draws a header with a tab for each view and breadcrumbs, the screen on top, and a footer with the status line and key hints as buttons. Each view keeps its own stack of screens, so switching away and back returns to where you were. It probes the terminal (background colour, kitty graphics, cell size), counts running jobs for Rex's tab status, and focuses on hover with a desktop pointer. Apps plug in their views, global keys, a loader, and a `Hook` that sees every message first. |

## Start an app

Markdown, kitty graphics and big headings come from the [markdown](https://github.com/ionut-maxim/markdown) library: its renderer, `kitty` and `bigtext` packages, and `teamd` for screens. `components.MarkdownTheme` maps the palette to its theme, and `Term.Caps` gives it what the frame found out about the terminal; create `teamd.Pictures` with `NoProbe`, since the frame probes.

`examples/hello` is the smallest app: two views as tabs, a list of cards you can tap, a detail screen with markdown drawn by the library (headings, alerts and a diagram are pictures in Rex), and a form (`n` adds a note). Copy it to start a new one.

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

MIT, see `LICENSE`.
