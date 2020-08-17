. ./envsetup.sh

### IS Web UI

is_setup() {
  app_setup
}

is_build_frontend() {
  rm frontend/output/* || true
  (cd frontend && elm make src/Farmation/Is/Main.elm --output=output/elm.js) || return 1
  (cd frontend && cp src/Farmation/Is/index.html output/) || return 1
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
  cd farmation/fonts || {
    echo "cd failure"
    exit 1
  }
  rm -rf "$name"
  mkdir -p "$name"
  fontgen -x="$x" -y="$y" -h="$h" -a="$fontstring" "$var" -img fonts.png >"$name/$name.txt"
  fontgen -x="$x" -y="$y" -h="$h" -a="$fontstring" "$var" -img fonts.png -o "$name"
  mv "$name.go" "$name/"
  cd - || exit 1
}

is_gen_fonts() {
  go get github.com/pbnjay/pixfont/cmd/fontgen
  is_gen_font tightpixel15 9 37 10 $FONTSTRING -v
  is_gen_font tightpixel15fixed 9 37 10 $FONTSTRING
  is_gen_font agencyfbbold40 14 118 31 $FONTSTRING_NUM -v
  is_gen_font agencyfbbold20 14 158 15 $FONTSTRING_NUM -v
}

is_build_assets_frontend() {
  mkdir -p farmation/assets/isfrontend || return 1
  genesis -C frontend/output -pkg isfrontend \
    index.html \
    elm.js \
    >farmation/assets/isfrontend/assets.go || return 1
  return 0
}

is_build_assets_lcd() {

  # shellcheck disable=SC2046
  # shellcheck disable=SC2091
  genesis -C farmation/assets/lcdassets -pkg lcdassets \
    $(
      cd farmation/assets/lcdassets
      # shellcheck disable=SC2035
      ls *.png
    ) \
    >farmation/assets/lcdassets/assets.go || return 1
  return 0
}

is_build_assets() {
  is_build_assets_frontend || return 1
  is_build_assets_lcd || return 1

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
  GOARCH=arm GOARM=7 go build -ldflags="-s -w" -o is_arm farmation/cmd/injector-sentry/main.go || return 1
  GOARCH=arm go build -o lcd_test farmation/cmd/lcd-test/main.go || return 1
  GOARCH=arm go build -o lindsay_sim farmation/cmd/lindsay-sim/main.go || return 1
  GOARCH=arm go build -o issplash farmation/cmd/splash/main.go || return 1
  return 0
}

is_build_windows() {
  is_build_dependencies || return 1
  GOOS=windows go build -o is.exe farmation/cmd/injector-sentry/main.go || return 1
}

is_run() {
  is_build_dependencies || return 1
  go run farmation/cmd/injector-sentry/main.go \
    -sim \
    -webUI \
    -portal nats://localhost:4333 \
    -serialNumber "wk231" \
    -auth="1234" \
    "$1" "$2" "$4" "$5" "$6" || return 1
  return 0
}

is_find_src_files() {
  find . -not \( -path ./frontend/src/Generated -prune \) \
    -not \( -path ./farmation/assets -prune \) \
    -not \( -path ./assets -prune \) \
    -name "*.go" -o -name "*.elm"
}

is_watch() {
  cmd=". ./is-envsetup.sh; is_run $*"
  is_find_src_files | entr -r /bin/sh -c "$cmd"
}

is_build_gpio_test() {
  GOARCH="arm" go build -o gpio-test farmation/cmd/gpio-test/main.go
}

### Portal

is_portal_build_frontend() {
  ELMARGS="$1"
  rm frontend/output/* || true
  siot_build_frontend "$ELMARGS"
  (cd frontend && cp src/Farmation/public/*.png output/) || return 1
  return 0
}

is_portal_build_assets() {
  genesis -C "assets/files" -pkg files \
    dummy \
    >assets/files/assets.go || return 1

  mkdir -p assets/frontend || return 1
  genesis -C frontend/output -pkg frontend \
    index.html \
    elm.js \
    main.js \
    ble.js \
    ports.js \
    styles.css \
    farmation-logo.png \
    Injector.png \
    WaterOn.png \
    Irrigator.png \
    Armed.png \
    Shutdown.png \
    >assets/frontend/assets.go || return 1
  return 0
}

is_portal_build_dependencies() {
  ELMARGS=$1
  is_portal_build_frontend "$ELMARGS" || return 1
  is_portal_build_assets || return 1
  return 0
}

is_portal_build() {
  is_portal_build_dependencies --optimize || return 1
  go build -o is-portal farmation/cmd/portal/main.go || return 1
  return 0
}

is_portal_run() {
  export SIOT_DATA=./portal_db
  export SIOT_AUTH=1234
  export SIOT_NATS_PORT=4333
  export SIOT_NATS_TLS_CERT=server-cert.pem
  export SIOT_NATS_TLS_KEY=server-key.pem
  export SIOT_NATS_SERVER=nats://localhost:4333
  mkdir -p $SIOT_DATA
  is_portal_build_dependencies --debug || return 1
  go run farmation/cmd/portal/main.go "$@" || return 1
  return 0
}

is_portal_watch() {
  echo "watch args: $*"
  cmd=". ./is-envsetup.sh; is_portal_run $*"
  is_find_src_files | entr -r /bin/sh -c "$cmd"
}

### Google Cloud server

INSTANCE="instance-1"

is_vm_start() {
  gcloud compute instances start $INSTANCE
}

is_vm_stop() {
  # We have to ssh into the vm to shut it off
  USER=$1
  gcloud compute ssh "$USER@$INSTANCE" --command="sudo poweroff"
}

is_vm_ssh() {
  USER=$1
  gcloud compute ssh "$USER@$INSTANCE"
}

is_vm_scp() {
  USER=$1 PATH_CURRENT=$2 PATH_NEW=$3
  gcloud compute scp "$USER@$INSTANCE:$PATH_CURRENT $PATH_NEW"
}

# For uploading images from server
is_vm_upload() {
  gsutil
}
