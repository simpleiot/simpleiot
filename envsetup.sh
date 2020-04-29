RECOMMENDED_ELM_VERSION=0.19.1

if [ -z "$GOPATH" ]; then
  export GOPATH=$HOME/go
fi

export GOBIN=$GOPATH/bin

# map tools from project go modules

genesis() {
  go run github.com/benbjohnson/genesis/cmd/genesis "$@"
}

golint() {
  go run golang.org/x/lint/golint "$@"
}

siot_install_frontend_deps() {
  (cd "frontend$1" && npm install)
}

siot_check_elm() {
  if ! npx elm --version >/dev/null 2>&1; then
    echo "Please install elm >= 0.19"
    echo "https://guide.elm-lang.org/install.html"
    return 1
  fi

  version=$(npx elm --version)
  if [ "$version" != "$RECOMMENDED_ELM_VERSION" ]; then
    echo "found elm $version, recommend elm version $RECOMMENDED_ELM_VERSION"
    echo "not sure what will happen otherwise"
  fi

  return 0
}

siot_setup() {
  go mod download
  siot_check_elm || return 1
  return 0
}

siot_build_frontend() {
  rm -f "frontend$1/output"/*
  if [ "$1" = "2" ]; then
    (cd "frontend$1" && npx elm-spa build) || return 1
  fi
  (cd "frontend$1" && npx elm make --debug src/Main.elm --output=output/elm.js) || return 1
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
  frontend_version=""
  if [ "$1" = "2" ]; then
    frontend_version=2
    shift
  fi

  siot_build_dependencies "$frontend_version" || return 1
  go run cmd/siot/main.go "$@" || return 1
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

# please run the following before pushing -- best if your editor can be set up
# to do this automatically.
siot_test() {
  siot_build_dependencies 2
  go fmt ./...
  go test "$@" ./... || return 1
  golint -set_exit_status ./... || return 1
  go vet ./... || return 1
  return 0
}
