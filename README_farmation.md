# Farmation Go code

This repository stores Go code used in various Farmation
projects and is based on the Simple IoT project.

All farmation specific Go code currently lives in the
`farmation` sub directory. This may change later but is at
least a place to start.

All frontend code that is specific to farmation lives in
the `frontend/farmation/` directory.

## Fonts

This project uses https://github.com/pbnjay/pixfont to extract fonts from an image
and then generate fonts from it. To modify a font:

- install the fonts we use on your system
  - `sudo mkdir /usr/share/fonts/farmation`
  - `sudo cp farmation/fonts/*.ttf /usr/share/fonts/farmation`
  - `sudo fc-cache`
- changing a font
  - `gimp farmation/fonts/fonts.xcf`
    - modify the string you want to change
    - select export as (png)
    - turn off all extra options in export (exif data, thumbnail, etc) as it messes up fontgen
  - modify `FONTSTRING` is `is_envsetup.sh` to match the string in the image
  - `. is_envsetup.sh`
  - `is_gen_fonts`
