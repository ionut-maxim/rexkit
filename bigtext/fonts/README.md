# Fonts

`SymbolsNerdFontMono-subset.ttf` is Symbols Nerd Font Mono from Nerd Fonts
v3.5.1 (https://github.com/ryanoasis/nerd-fonts, MIT licence in
`NerdFonts-LICENSE.txt`), cut down to the Font Awesome (U+F000–F2FF) and
Powerline (U+E0A0–E0D7) ranges with fontTools:

    pyftsubset SymbolsNerdFontMono-Regular.ttf \
      --unicodes="U+F000-F2FF,U+E0A0-E0D7" --no-hinting --desubroutinize \
      --output-file=SymbolsNerdFontMono-subset.ttf

Buttons drawn as images fall back to it for the icons the label font
lacks, so they show the same icons as the Rex apps draw in text.
