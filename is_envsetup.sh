. ./envsetup.sh

# IS Web UI

is_setup() {
  app_setup
}

is_build_frontend() {
  rm frontend/output/* || true
  (cd frontend && elm make src/Farmation/Is/Main.elm --output=output/elm.js) || return 1
  (cd frontend && cp public/index.html output/) || return 1
  return 0
}

FONTSTRING="ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789./:%,-"
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
  is_gen_font tightpixel15fixed 9 37 10 $FONTSTRING
  is_gen_font agencyfbbold40 14 118 31 $FONTSTRING_NUM -v
  is_gen_font agencyfbbold20 14 158 15 $FONTSTRING_NUM -v
}

is_build_assets() {
  mkdir -p farmation/assets/isfrontend || return 1
  genesis -C frontend/output -pkg isfrontend \
    index.html \
    elm.js \
    >farmation/assets/isfrontend/assets.go || return 1

  genesis -C farmation/assets/lcdassets -pkg lcdassets \
    $(
      cd farmation/assets/lcdassets
      ls *.png
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
  GOARCH=arm go build -o lindsay_sim farmation/cmd/lindsay-sim/main.go || return 1
  return 0
}

is_build_windows() {
  is_build_dependencies || return 1
  GOOS=windows go build -o is.exe farmation/cmd/injector-sentry/main.go || return 1
}

is_run() {
  is_build_dependencies || return 1
  go run farmation/cmd/injector-sentry/main.go -sim \
    -portal http://localhost:8080 \
    -serialNumber "wk231" \
    $1 $2 $4 $5 $6 || return 1
  return 0
}

is_build_gpio_test() {
  GOARCH="arm" go build -o gpio-test farmation/cmd/gpio-test/main.go
}

# Portal

is_portal_uglify() {
  (cd frontend/output && mv elm.js x &&
    uglifyjs x --compress 'pure_funcs="F2,F3,F4,F5,F6,F7,F8,F9,A2,A3,A4,A5,A6,A7,A8,A9",pure_getters,keep_fargs=false,unsafe_comps,unsafe' | uglifyjs --mangle --output=elm.js)
}

is_portal_build_frontend() {
  DEBUG=$1
  rm frontend/output/* || true
  if [ -z "$DEBUG" ]; then
    # build production version
    (cd frontend && elm make --optimize src/Farmation/Portal/Main.elm --output=output/elm.js) || return 1
    is_portal_uglify || return 1
  else
    # build debug version (can use Debug.log)
    (cd frontend && elm make src/Farmation/Portal/Main.elm --output=output/elm.js) || return 1
  fi
  (cd frontend && cp src/Farmation/public/index.html output/) || return 1
  (cd frontend && cp src/Farmation/public/*.png output/) || return 1
  return 0
}

is_portal_build_assets() {
  mkdir -p farmation/assets/portal || return 1
  genesis -C frontend/output -pkg portal \
    index.html \
    elm.js \
    farmation-logo.png \
    Injector.png \
    WaterOn.png \
    Irrigator.png \
    Armed.png \
    Shutdown.png \
    >farmation/assets/portal/assets.go || return 1
  return 0
}

is_portal_build_dependencies() {
  DEBUG=$1
  is_portal_build_frontend $DEBUG || return 1
  is_portal_build_assets || return 1
  return 0
}

is_portal_build() {
  is_portal_build_dependencies || return 1
  go build -o is-portal farmation/cmd/portal/main.go || return 1
  return 0
}

is_portal_run() {
  export SIOT_DATA=./portal_db
  mkdir -p $SIOT_DATA
  is_portal_build_dependencies debug || return 1
  go run farmation/cmd/portal/main.go || return 1
  return 0
}
