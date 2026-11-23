# Changelog

## [0.5.1](https://github.com/ionut-maxim/rexkit/compare/v0.5.0...v0.5.1) (2026-11-23)


### Bug Fixes

* **deps:** update github.com/charmbracelet/ultraviolet digest to 8ecbcf0 ([#22](https://github.com/ionut-maxim/rexkit/issues/22)) ([b1f2d6a](https://github.com/ionut-maxim/rexkit/commit/b1f2d6a4c1f6abc6e028a13f34ba34d66cc7ef93))

## [0.5.0](https://github.com/ionut-maxim/rexkit/compare/v0.4.1...v0.5.0) (2026-11-20)


### Features

* **shell:** an action bar with big buttons that follows the screen's shape ([#20](https://github.com/ionut-maxim/rexkit/issues/20)) ([09127dc](https://github.com/ionut-maxim/rexkit/commit/09127dc5fad6953d5f0c07be952ac92aeda55901))

## [0.4.1](https://github.com/ionut-maxim/rexkit/compare/v0.4.0...v0.4.1) (2026-11-20)


### Performance Improvements

* long lists and every frame redraw cheaply ([#15](https://github.com/ionut-maxim/rexkit/issues/15)) ([6dd24e1](https://github.com/ionut-maxim/rexkit/commit/6dd24e1987277efdb319a8ffb92350552cc38d3d))

## [0.4.0](https://github.com/ionut-maxim/rexkit/compare/v0.3.0...v0.4.0) (2026-11-15)


### Features

* **lists:** lists that scroll freely, like a touch screen ([#12](https://github.com/ionut-maxim/rexkit/issues/12)) ([1f84225](https://github.com/ionut-maxim/rexkit/commit/1f84225c6af4c9c64eaf97d56fd27fb809f2c397))

## [0.3.0](https://github.com/ionut-maxim/rexkit/compare/v0.2.0...v0.3.0) (2026-11-15)


### ⚠ BREAKING CHANGES

* the markdown and kitty packages, bigtext.Heading, bigtext.CellKnown and the kitty.Cell global are gone; use github.com/ionut-maxim/markdown with Term.Caps and teamd, and pass Term.Cell to bigtext.Strip. Rendered markdown changes.

### Features

* use the markdown library for markdown, kitty graphics and headings ([a84774a](https://github.com/ionut-maxim/rexkit/commit/a84774a7ad0fad32a34e2c18d52795f4fa278a44))

## [0.2.0](https://github.com/ionut-maxim/rexkit/compare/v0.1.0...v0.2.0) (2026-11-05)


### Features

* **form:** a form screen with text fields for touch ([#9](https://github.com/ionut-maxim/rexkit/issues/9)) ([1cab75f](https://github.com/ionut-maxim/rexkit/commit/1cab75f77737dd3e127a3aad3b027b25706e3a61))
* **icons:** add copy, mail, plus, search and power ([#6](https://github.com/ionut-maxim/rexkit/issues/6)) ([54d5f0a](https://github.com/ionut-maxim/rexkit/commit/54d5f0afb0bb10e09bf839a840eeea1765d16d4e))
* **shell:** tell apps when the terminal gains focus ([#7](https://github.com/ionut-maxim/rexkit/issues/7)) ([30ca6de](https://github.com/ionut-maxim/rexkit/commit/30ca6dea17a7d8d835ae52fc550785c72ee0b4cf))

## 0.1.0 (2026-11-05)


### Features

* **shell:** the app frame, with an example app ([#5](https://github.com/ionut-maxim/rexkit/issues/5)) ([932e4a0](https://github.com/ionut-maxim/rexkit/commit/932e4a03ed11a871655bdc17ea54ffce731c34de))
* theme, icons, components, touch, kitty, bigtext, markdown and rexstatus ([#2](https://github.com/ionut-maxim/rexkit/issues/2)) ([fd9d0aa](https://github.com/ionut-maxim/rexkit/commit/fd9d0aa17ab21957b89bc837bb0b50c20235fc09))
