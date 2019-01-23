. envsetup.sh

is_setup() {
  app_setup
}

is_build_frontend() {
  (cd frontend && elm make src/Farmation/Is/Main.elm --output=public/elm.js) || return 1
  (cd frontend && cp index.html public/) || return 1
  return 0
}

FONTSTRING="ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789"

is_gen_fonts() {
  cd farmation/fonts/tightpixel15
  rm tightpixel15.go
  cmd -x=9 -y=37 -h=10 -a=$FONTSTRING -v -img tightpixel15.png >tightpixel15.txt
  cmd -x=9 -y=37 -h=10 -a=$FONTSTRING -v -img tightpixel15.png -o tightpixel15
  #cmd -txt tightpixel15.txt -v -o tightpixel15
  cd -
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
  return 0
}

is_build_windows() {
  is_build_dependencies || return 1
  GOOS=windows go build -o is.exe farmation/cmd/injector-sentry/main.go || return 1
}

is_run() {
  is_build_dependencies || return 1
  go run farmation/cmd/injector-sentry/main.go || return 1
  return 0
}

is_build_gpio_test() {
  GOARCH="arm" go build -o gpio-test farmation/cmd/gpio-test/main.go
}
