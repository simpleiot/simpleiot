RECOMMENDED_ELM_VERSION=0.19.1

# map tools from project go modules

genesis() {
  GOARCH='' go run github.com/benbjohnson/genesis/cmd/genesis "$@"
}

golint() {
  GOARCH='' go run golang.org/x/lint/golint "$@"
}

bbolt() {
  go run go.etcd.io/bbolt/cmd/bbolt "$@"
}

# genji does not work very well like this, so install the binary and run that
#genji() {
#  go run github.com/genjidb/genji/cmd/genji "$@"
#}

siot_install_proto_gen_go() {
  cd ~ && go get -u google.golang.org/protobuf/cmd/protoc-gen-go
  cd - || exit
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
  siot_install_frontend_deps
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

  genesis -C "assets/files" -pkg files \
    dummy \
    >assets/files/assets.go || return 1
  return 0
}

siot_uglify() {
  (cd frontend && mv output/elm.js output/x &&
    npx uglifyjs output/x --compress 'pure_funcs="F2,F3,F4,F5,F6,F7,F8,F9,A2,A3,A4,A5,A6,A7,A8,A9",pure_getters,keep_fargs=false,unsafe_comps,unsafe' | npx uglifyjs --mangle --output output/elm.js)
}

siot_build_dependencies() {
  ELMARGS=$1
  siot_build_frontend "$ELMARGS" || return 1
  if [ "$ELMARGS" = "--optimize" ]; then
    echo "running uglify"
    siot_uglify
  fi
  siot_build_assets || return 1
  return 0
}

siot_build() {
  siot_build_dependencies --optimize || return 1
  BINARY_NAME=siot
  if [ "${GOOS}" = "windows" ]; then
    BINARY_NAME=siot.exe
  fi
  CGO_ENABLED=0 go build -ldflags="-s -w -X main.siotVersion=$(git describe --tags HEAD)" -o $BINARY_NAME cmd/siot/main.go || return 1
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
  go run -race cmd/siot/main.go "$@" || return 1
  return 0
}

# run siot_mkcert first
siot_run_tls() {
  echo "run args: $*"
  export SIOT_NATS_TLS_CERT=server-cert.pem
  export SIOT_NATS_TLS_KEY=server-key.pem
  siot_build_dependencies --debug || return 1
  go run cmd/siot/main.go "$@" || return 1
  return 0
}

# please install mkcert and run mkcert -install first
siot_mkcert() {
  mkcert -cert-file server-cert.pem -key-file server-key.pem localhost ::1
}

find_src_files() {
  find . -not \( -path ./frontend/src/Spa/Generated -prune \) -not \( -path ./assets -prune \) -name "*.go" -o -name "*.elm"
}

siot_watch() {
  echo "watch args: $*"
  cmd=". ./envsetup.sh; siot_run $*"
  find_src_files | entr -r /bin/sh -c "$cmd"
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

siot_test_frontend() {
  #(cd frontend && npx elm-analyse || return 1) || return 1
  (cd frontend && npx elm-test || return 1) || return 1
}

# please run the following before pushing -- best if your editor can be set up
# to do this automatically.
siot_test() {
  siot_build_dependencies --optimize || return 1
  siot_test_frontend || return 1
  #gofmt -l ./... || return 1
  go test "$@" ./... || return 1
  golint -set_exit_status ./... || return 1
  go vet ./... || return 1
  return 0
}

# following can be used to set up influxdb for local testing
siot_setup_influx() {
  export SIOT_INFLUX_URL=http://localhost:8086
  #export SIOT_INFLUX_USER=admin
  #export SIOT_INFLUX_PASS=admin
  export SIOT_INFLUX_DB=siot
}

siot_protobuf() {
  echo "generating protobufs"
  protoc --proto_path=internal/pb internal/pb/*.proto --go_out=./ || return 1
}

siot_edge_run() {
  go run cmd/edge/main.go "$*"
}

# download goreleaser from https://github.com/goreleaser/goreleaser/releases/
# and put in /usr/local/bin
# This can be useful to test/debug the release process locally
siot_goreleaser_build() {
  goreleaser build --skip-validate --rm-dist
}

# before releasing, you need to tag the release
siot_goreleaser_release() {
  #TODO add depend build to goreleaser config
  siot_build_dependencies --optimize
  goreleaser release
}
