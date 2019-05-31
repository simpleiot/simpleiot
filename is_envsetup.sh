. ./envsetup.sh

is_setup() {
  app_setup
}

is_build_frontend() {
  (cd frontend && elm make src/Farmation/Is/Main.elm --output=public/elm.js) || return 1
  (cd frontend && cp index.html public/) || return 1
  return 0
}

FONTSTRING="ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789./:"
FONTSTRING_NUM="0123456789."

is_gen_font() {
  name=$1
  x=$2
  y=$3
  h=$4
  fontstring=$5
  var=$6
  cd farmation/fonts
  rm -rf $name
  mkdir -p $name
  fontgen -x=$x -y=$y -h=$h -a=$fontstring $var -img fonts.png >$name/$name.txt
  fontgen -x=$x -y=$y -h=$h -a=$fontstring $var -img fonts.png -o $name
  mv $name.go $name/
  cd -
}

is_gen_fonts() {
  go get github.com/pbnjay/pixfont/cmd/fontgen
  is_gen_font tightpixel15 9 37 10 $FONTSTRING -v
  is_gen_font agencyfbbold40 14 118 31 $FONTSTRING_NUM -v
  is_gen_font agencyfbbold20 14 158 15 $FONTSTRING_NUM -v
}

is_build_assets() {
  mkdir -p farmation/assets/isfrontend || return 1
  genesis -C frontend/public -pkg isfrontend \
    index.html \
    elm.js \
    >farmation/assets/isfrontend/assets.go || return 1

  genesis -C farmation/assets/lcdassets -pkg lcdassets \
    $(
      cd farmation/assets/lcdassets
      ls *.bmp
    ) \
    >farmation/assets/lcdassets/assets.go || return 1

  return 0
}

is_build_dependencies() {
  is_build_frontend || return 1
  is_build_assets || return 1
  return 0
}

is_build() {
  is_build_dependencies || return 1
  go build -o is farmation/cmd/injector-sentry/main.go || return 1
  return 0
}

is_build_arm() {
  is_build_dependencies || return 1
  GOARCH=arm go build -o is_arm farmation/cmd/injector-sentry/main.go || return 1
  GOARCH=arm go build -o lcd_test farmation/cmd/lcd-test/main.go || return 1
  return 0
}

is_build_windows() {
  is_build_dependencies || return 1
  GOOS=windows go build -o is.exe farmation/cmd/injector-sentry/main.go || return 1
}

is_run() {
  is_build_dependencies || return 1
  go run farmation/cmd/injector-sentry/main.go -sim $1 $2 $4 $5 $6 || return 1
  return 0
}

is_build_gpio_test() {
  GOARCH="arm" go build -o gpio-test farmation/cmd/gpio-test/main.go
}
