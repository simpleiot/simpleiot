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
  (cd "frontend" && npm install)
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
  ELMARGS=$1
  echo "Elm args: $ELMARGS"
  rm -f "frontend/output"/*
  (cd "frontend" && npx elm-spa build) || return 1
  (cd "frontend" && npx elm make "$ELMARGS" src/Main.elm --output=output/elm.js) || return 1
  cp "frontend/public"/* "frontend/output/" || return 1
  cp "frontend/public/index.html" "frontend/output/index.html" || return 1
  cp docs/simple-iot-app-logo.png "frontend/output/" || return 1
  return 0
}

siot_build_assets() {
  mkdir -p assets/frontend || return 1
  genesis -C "frontend/output" -pkg frontend \
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
  ELMARGS=$1
  siot_build_frontend "$ELMARGS" || return 1
  siot_build_assets || return 1
  return 0
}

# the following can be used to build v2 of the frontend: siot_build 2
siot_build() {
  siot_build_dependencies --optimize || return 1
  go build -o siot cmd/siot/main.go || return 1
  return 0
}

siot_deploy() {
  siot_build_dependencies || return 1
  gcloud app deploy cmd/portal || return 1
  return 0
}

siot_run() {
  echo "run args: $*"
  siot_build_dependencies --debug || return 1
  go run cmd/siot/main.go "$@" || return 1
  return 0
}

find_src_files() {
  find . -not \( -path ./frontend/src/Generated -prune \) -not \( -path ./assets -prune \) -name "*.go" -o -name "*.elm"
}

siot_watch() {
  echo "watch args: $*"
  cmd=". ./envsetup.sh; siot_run $*"
  find_src_files | entr -r /bin/sh -c "$cmd"
}

siot_build_docs() {
  # download snowboard binary from: https://github.com/bukalapak/snowboard/releases
  # and stash in /usr/local/bin
  snowboard lint docs/api.apib || return 1
  snowboard html docs/api.apib -o docs/api.html || return 1
}

# TODO finish this and add to siot_test ...
check_go_format() {
  gofiles=$(find . -name "*.go")
  unformatted=$(gofmt -l "$gofiles")
  if [ -n "$unformatted" ]; then
    return 1
  fi
  return 0
}

# please run the following before pushing -- best if your editor can be set up
# to do this automatically.
siot_test() {
  siot_build_dependencies --optimize || return 1
  (cd frontend && npx elm-analyse || return 1) || return 1
  #gofmt -l ./... || return 1
  go test "$@" ./... || return 1
  golint -set_exit_status ./... || return 1
  go vet ./... || return 1
  return 0
}
