RECOMMENDED_ELM_VERSION=0.19.0

app_check_elm() {
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

app_setup() {
  go mod download
  go install github.com/benbjohnson/genesis/... || return 1
  app_check_elm || return 1
  return 0
}

app_build_frontend() {
  (cd frontend && elm make src/Main.elm --output=public/elm.js) || return 1
  (cd frontend && cp index.html public/) || return 1
  return 0
}

app_build_assets() {
  mkdir -p assets/frontend || return 1
  genesis -C frontend/public -pkg frontend index.html elm.js >assets/frontend/assets.go || return 1
  return 0
}

app_build_dependencies() {
  app_build_frontend || return 1
  app_build_assets || return 1
  return 0
}

app_build() {
  app_build_dependencies || return 1
  go build -o siot-portal cmd/portal/main.go || return 1
  return 0
}

app_deploy() {
  app_build_dependencies || return 1
  gcloud app deploy cmd/portal || return 1
  return 0
}

app_run() {
  app_build_dependencies || return 1
  go run cmd/portal/main.go || return 1
  return 0
}
