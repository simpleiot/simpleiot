. envsetup.sh

is_setup() {
  app_setup
}

is_build_frontend() {
  (cd frontend && elm make src/Farmation/Is/Main.elm --output=public/elm.js) || return 1
  (cd frontend && cp index.html public/) || return 1
  return 0
}

is_build_assets() {
  mkdir -p farmation/assets/isfrontend || return 1
  genesis -C frontend/public -pkg isfrontend \
    index.html \
    elm.js \
    >farmation/assets/isfrontend/assets.go || return 1
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

is_run() {
  is_build_dependencies || return 1
  go run farmation/cmd/injector-sentry/main.go || return 1
  return 0
}
