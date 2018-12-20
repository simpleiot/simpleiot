is_build_frontend() {
  (cd isfrontend && elm make src/Main.elm --output=public/elm.js) || return 1
  (cd isfrontend && cp index.html public/) || return 1
  return 0
}

is_build_assets() {
  mkdir -p assets/isfrontend || return 1
  genesis -C isfrontend/public -pkg isfrontend \
    index.html \
    elm.js \
    >assets/isfrontend/assets.go || return 1
  return 0
}

is_build_dependencies() {
  is_build_frontend || return 1
  is_build_assets || return 1
  return 0
}

is_build() {
  is_build_dependencies || return 1
  go build -o is cmd/injector-sentry/main.go || return 1
  return 0
}

is_run() {
  is_build_dependencies || return 1
  go run cmd/injector-sentry/main.go || return 1
  return 0
}
