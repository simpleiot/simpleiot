app_setup() {
  go get -u github.com/benbjohnson/genesis/...
}

app_build_frontend() {
  (cd frontend && elm make src/Main.elm --output=public/elm.js) || return 1
  (cd frontend && cp index.html public/) || return 1
}

app_build_assets() {
  mkdir -p assets/frontend
  genesis -C frontend/public -pkg frontend index.html elm.js >assets/frontend/assets.go
}

app_build_dependencies() {
  app_build_frontend || return 1
  app_build_assets || return 1
}

app_build() {
  app_build_dependencies
  go build -o siot-portal cmd/portal/main.go || return 1
  return 0
}

app_deploy() {
  app_build_dependencies
  gcloud app deploy cmd/portal
}

app_run() {
  app_build_dependencies
  go run cmd/portal/main.go || return 1
  return 0
}
