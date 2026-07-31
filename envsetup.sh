#!/bin/sh

if [ -f local.sh ]; then
	echo "reading local settings"
	. ./local.sh
fi

RECOMMENDED_ELM_VERSION=0.19.1

# map tools from project go modules

air() {
	go run github.com/cosmtrek/air "$@"
}

siot_install_proto_gen_go() {
	cd ~ && go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
	cd - || exit
}

siot_install_frontend_deps() {
	(cd "frontend" && npm install)
	(cd "frontend" && npx elm-tooling install)
	(cd "frontend/lib" && npm ci)
}

siot_check_elm() {
	# this no longer works with the way we are installing elm
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

siot_check_go() {
	# Get the installed Go version
	go_version=$(go version | awk '{print $3}' | sed 's/go//g')

	# Split the version into major, minor, and patch components
	major=$(echo "$go_version" | awk -F'.' '{print $1}')
	minor=$(echo "$go_version" | awk -F'.' '{print $2}')
	patch=$(echo "$go_version" | awk -F'.' '{print $3}')

	# Check if the version is greater than 1.22
	if [ "$major" -gt 1 ] || { [ "$major" -eq 1 ] && [ "$minor" -gt 22 ]; } || { [ "$major" -eq 1 ] && [ "$minor" -eq 22 ] && [ "$patch" -gt 0 ]; }; then
		echo "Go version $go_version is greater than 1.22"
		return 0
	else
		echo "Go version $go_version is not greater than 1.22"
		return 1
	fi
}

siot_setup() {
	siot_check_go || return 1
	siot_install_frontend_deps
	# the following is to work around a race condition
	# where the first time you run npx elm, you get an error:
	# elm: Text file busy
	(cd frontend && (npx elm || true))
	# make sure elm-spa auto-generated stuff is set up
	(cd frontend && npx elm-spa build)
	return 0
}

siot_build_frontend() {
	(cd "frontend" && npx elm-spa build) || return 1
	gzip -f frontend/public/dist/elm.js
	return 0
}

siot_version() {
	git describe --tags HEAD
}

siot_build_backend() {
	BINARY_NAME=siot
	if [ "${GOOS}" = "windows" ]; then
		BINARY_NAME=siot.exe
	fi
	CGO_ENABLED=0 go build -ldflags="-s -w -X main.version=$(siot_version)" -o $BINARY_NAME cmd/siot/main.go || return 1
	return 0
}

siot_build() {
	siot_build_frontend || return 1
	siot_build_backend || return 1
}

siot_build_arm() {
	siot_build_frontend || return 1
	GOARCH=arm GOARM=7 go build -ldflags="-s -w -X main.version=$(siot_version)" -o siot_arm cmd/siot/main.go || return 1
	return 0
}

siot_build_arm64() {
	siot_build_frontend || return 1
	GOARCH=arm64 go build -ldflags="-s -w -X main.version=$(siot_version)" -o siot_arm64 cmd/siot/main.go || return 1
	return 0
}

siot_build_arm_debug() {
	siot_build_frontend || return 1
	GOARCH=arm GOARM=7 go build -ldflags="-s -w -X main.version=$(siot_version)" -o siot_arm cmd/siot/main.go || return 1
	return 0
}

siot_deploy() {
	siot_build_frontend || return 1
	gcloud app deploy cmd/portal || return 1
	return 0
}

siot_run() {
	siot_build_frontend || return 1
	go build -ldflags="-X main.version=$(siot_version)" -o siot -race cmd/siot/main.go || return 1
	./siot "$@"
	return 0
}

# run siot_mkcert first
siot_run_tls() {
	export SIOT_NATS_TLS_CERT=server-cert.pem
	export SIOT_NATS_TLS_KEY=server-key.pem
	siot_build_frontend || return 1
	go run cmd/siot/main.go "$@" || return 1
	return 0
}

# please install mkcert and run mkcert -install first
siot_mkcert() {
	mkcert -cert-file server-cert.pem -key-file server-key.pem localhost ::1
}

find_src_files() {
	find . -not \( -path ./frontend/src/Spa/Generated -prune \) -not \( -path ./assets -prune \) -name "*.go"
}

siot_watch_go() {
	echo "watch args: $*"
	air serve -dev "$*"
}

siot_watch_elm() {
	(cd frontend && npx elm-watch hot) || false
}

siot_watch() {
	npx run-pty \
		% /bin/sh -c ". ./envsetup.sh && siot_watch_elm" \
		% /bin/sh -c ". ./envsetup.sh && siot_watch_go $*"
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
	(cd frontend && npx elm-test || return 1) || return 1
	(cd frontend && npx elm-review || return 1) || return 1
}

siot_test_frontend_lib() {
	(cd ./frontend/lib && npm run lint || return 1) || return 1
	echo "Starting SimpleIOT..."
	./siot serve --store siot_test_frontend_lib.sqlite --resetStore 2>/dev/null &
	PID=$!
	sleep 1
	(cd ./frontend/lib && npm run test || return 1)
	CODE=$?
	echo "Stopping SimpleIOT..."
	kill -s SIGINT $PID
	wait $PID
	echo "SimpleIOT Stopped"
	if [ "$CODE" = "0" ]; then
		rm siot_test_frontend_lib.sqlite
	fi
}

siot_frontend_fix() {
	(cd frontend && npx elm-review --fix-all)
}

siot_format() {
	echo "Formatting Go code..."
	gofmt -w .
	echo "Formatting Elm code..."
	(cd frontend && npx elm-format --yes src/)
}

# please run the following before pushing -- best if your editor can be set up
# to do this automatically.
siot_test() {
	echo "Build frontend ..."
	siot_build_frontend || return 1
	echo "Test frontend ..."
	siot_test_frontend || return 1
	echo "Test backend ..."
	go test -p=1 -race "$@" ./... || return 1
	echo "Lint backend ..."
	golangci-lint run || return 1
	echo "Testing passed :-)"
	return 0
}

# following can be used to set up influxdb for local testing
siot_setup_influx() {
	export SIOT_INFLUX_URL=http://localhost:8086
	#export SIOT_INFLUX_USER=admin
	#export SIOT_INFLUX_PASS=admin
	export SIOT_INFLUX_DB=siot
}

siot_protobuf_go() {
	protoc --proto_path=internal/pb internal/pb/*.proto --go_out=./ || return 1
}

siot_protobuf_js() {
	protoc --proto_path=internal/pb internal/pb/*.proto --js_out=import_style=commonjs,binary:./frontend/lib/protobuf/ || return 1
}

siot_protobuf() {
	echo "generating protobufs"
	siot_protobuf_go
	siot_protobuf_js
}

siot_edge_run() {
	go run cmd/edge/main.go "$*"
}

# download goreleaser from https://github.com/goreleaser/goreleaser/releases/
# and put in /usr/local/bin
# This can be useful to test/debug the release process locally
siot_goreleaser_build() {
	goreleaser build --snapshot --clean
}

# pushing the tag starts the Release workflow (.github/workflows/release.yml),
# which builds the binaries and publishes the GitHub release
# the release notes come from the CHANGELOG.md section for the version being
# released, so move the "Next" section under a "[X.Y.Z] - <date>" heading first
siot_release() {
	VERSION=$1
	if [ -z "$VERSION" ]; then
		echo "must provide version in format vX.Y.Z"
		return 1
	fi

	# update elm.js.gz
	siot_build_frontend || return 1
	git commit -m "update FE assets" frontend/public/dist/elm.js.gz || return 1
	git push || return 1
	git tag -f "$VERSION" || return 1
	git push origin "$VERSION" || return 1
	siot_deploy_docs || return 1
	# refresh godocs site
	wget "https://proxy.golang.org/github.com/simpleiot/simpleiot/@v/${VERSION}.info" || return 1
	rm "${VERSION}.info"
}

# dblab keyboard shortcuts
# - Ctrl+space execute query
# - Ctrl+H,J,K,L move to panel left,below,above,right
# see more keybindings here: https://github.com/danvergara/dblab#key-bindings
siot_dblab() {
	STORE=siot.sqlite
	if [ "$1" != "" ]; then
		STORE=$1
	fi
	go run github.com/danvergara/dblab@latest --db "$STORE" --driver sqlite3
}

MDBOOK_VERSION=0.5.4
MDBOOK_IMAGE=simpleiot/mdbook:$MDBOOK_VERSION

# build the documentation image from Dockerfile.mdbook. The Dockerfile needs no
# files from the repo, so it is fed on stdin with an empty build context.
siot_mdbook_image() {
	docker build --build-arg "MDBOOK_VERSION=$MDBOOK_VERSION" \
		-t "$MDBOOK_IMAGE" - <Dockerfile.mdbook
}

siot_mdbook_image_ensure() {
	if docker image inspect "$MDBOOK_IMAGE" >/dev/null 2>&1; then
		return 0
	fi
	echo "building $MDBOOK_IMAGE"
	siot_mdbook_image
}

# run mdbook in the documentation image. book.toml sets src = ".", so mdbook
# treats the whole repo as book source and copies every file it walks into the
# output. Mounting only the files the book is built from keeps the output to
# documentation alone. Arguments are appended to the docker command line, so
# they start with any further docker options and end with the mdbook command.
siot_mdbook_run() {
	siot_mdbook_image_ensure || return 1
	mkdir -p book || return 1
	docker run --rm --user "$(id -u):$(id -g)" \
		-v "$(pwd)/book.toml":/book/book.toml:ro \
		-v "$(pwd)/SUMMARY.md":/book/SUMMARY.md:ro \
		-v "$(pwd)/README.md":/book/README.md:ro \
		-v "$(pwd)/docs":/book/docs:ro \
		-v "$(pwd)/book":/book/book \
		"$@"
}

siot_mdbook() {
	siot_mdbook_run -p 3333:3000 $MDBOOK_IMAGE serve -n 0.0.0.0
}

siot_mdbook_build() {
	siot_mdbook_run $MDBOOK_IMAGE build
}

siot_mdbook_cleanup() {
	rm -rf book
}

# the book output now holds documentation only, so the whole tree is deployed
# apart from book.toml, which mdbook copies through because it sits in src, and
# the editor backup files draw.io leaves next to the diagram sources.
# book/ rather than book/* so that dotfiles, notably the .nojekyll marker
# mdbook writes, are included and --delete compares the same set of files.
siot_deploy_docs() {
	siot_mdbook_cleanup
	siot_mdbook_build || return 1
	rsync -av --delete \
		--exclude='book.toml' \
		--exclude='~$*' \
		--exclude='*.bkp' \
		book/ bec-systems.com:/srv/http/siot/docs/ || return 1
	return 0
}
