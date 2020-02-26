RECOMMENDED_ELM_VERSION=0.19.0

siot_check_elm() {
  if ! elm --version >/dev/null 2>&1; then
    echo "Please install elm >= 0.19"
    echo "https://guide.elm-lang.org/install.html"
    return 1
  fi

  version=$(elm --version)
  if [ "$version" != "$RECOMMENDED_ELM_VERSION" ]; then
    echo "found elm $version, recommend elm version $RECOMMENDED_ELM_VERSION"
    echo "not sure what will happen otherwise"
  fi

  return 0
}

siot_check_gopath_bin() {
  if [ -z "$GOPATH" ]; then
    GOPATH=~/go
  fi

  GOBIN=$GOPATH/bin

  if [[ ":$PATH:" != *":$GOBIN:"* ]]; then
    echo "You must add \$GOPATH/bin to your environment PATH variable"
    echo "GOPATH defaults to ~/go"
    return 1
  fi

  return 0
}

siot_setup() {
  go mod download
  go install github.com/benbjohnson/genesis/... || return 1
  siot_check_elm || return 1
  siot_check_gopath_bin || return 1
  return 0
}

siot_build_frontend() {
  rm "frontend$1/output"/* || true
  (cd "frontend$1" && elm make src/Main.elm --output=output/elm.js) || return 1
  cp "frontend$1/public"/* "frontend$1/output/" || return 1
  cp "frontend$1/public/index$1.html" "frontend$1/output/index.html" || return 1
  cp docs/simple-iot-app-logo.png "frontend$1/output/" || return 1
  return 0
}

siot_build_assets() {
  mkdir -p assets/frontend || return 1
  genesis -C "frontend$1/output" -pkg frontend \
    index.html \
    elm.js \
    main.js \
    ble.js \
    simple-iot-app-logo.png \
    ports.js \
    styles.css \
    >assets/frontend/assets.go || return 1
  return 0
}

siot_build_dependencies() {
  siot_build_frontend "$1" || return 1
  siot_build_assets "$1" || return 1
  return 0
}

# the following can be used to build v2 of the frontend: siot_build 2
siot_build() {
  siot_build_dependencies "$1" || return 1
  go build -o siot cmd/siot/main.go || return 1
  return 0
}

siot_deploy() {
  siot_build_dependencies || return 1
  gcloud app deploy cmd/portal || return 1
  return 0
}

siot_run() {
  siot_build_dependencies || return 1
  go run cmd/siot/main.go || return 1
  return 0
}

siot_run_device_sim() {
  go run cmd/siot/main.go -sim || return 1
  return 0
}

siot_build_docs() {
  # download snowboard binary from: https://github.com/bukalapak/snowboard/releases
  # and stash in /usr/local/bin
  snowboard lint docs/api.apib || return 1
  snowboard html docs/api.apib -o docs/api.html || return 1
}

siot_test() {
  go test ./...
}
