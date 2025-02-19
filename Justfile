# This Justfile contains rules/targets/scripts/commands that are used when
# developing. Unlike a Makefile, running `just <cmd>` will always invoke
# that command. For more information, see https://github.com/casey/just
#
#
# this setting will allow passing arguments through to tasks, see the docs here
# https://just.systems/man/en/chapter_24.html#positional-arguments
set positional-arguments

# print all available commands by default
default:
  just --list

# build pgmigrate
build:
  #!/usr/bin/env bash
  ldflags=$(./scripts/golang-ldflags.sh)
  go build -ldflags "$ldflags" -o bin/pgmigrate ./cmd/pgmigrate

# test all packages
test *args='./... ./cmd/pgmigrate/...':
  go test -race -count=1 $@

# lint pgmigrate
lint *args='./... ./cmd/pgmigrate/...':
  golangci-lint config verify --config .golangci.yaml
  golangci-lint run --fix --config .golangci.yaml $@

# lint nix files
lint-nix:
  find . -name '*.nix' | xargs nixpkgs-fmt

# tag pgtestdb with current version
tag:
  #!/usr/bin/env zsh
  raw="$(cat VERSION)"
  git tag "$raw"

tag-cli:
  #!/usr/bin/env zsh
  raw="$(cat VERSION)"
  git tag "cmd/pgmigrate/$raw"

tag-example:
  #!/usr/bin/env zsh
  raw="$(cat VERSION)"
  git tag "example/$raw"

tidy:
  #!/usr/bin/env zsh
  go mod tidy
  pushd cmd/pgmigrate && go mod tidy && popd
  pushd example && go mod tidy && popd
  rm -rf go.work.sum
  go mod tidy
  go work sync
  go mod tidy

# set the VERSION and go.mod versions.
update-version version:
  #!/usr/bin/env zsh
  OLD_VERSION=$(cat VERSION)
  NEW_VERSION=$1
  echo "bumping $OLD_VERSION -> $NEW_VERSION"
  echo $NEW_VERSION > VERSION
  sed -i -e "s/$OLD_VERSION/$NEW_VERSION/g" **/README.md
  sed -i -e "s/pgmigrate $OLD_VERSION/pgmigrate $NEW_VERSION/g" **/go.mod

# builds local-pgmigrate, tagged with :latest and :$COMMIT_SHA
build-docker:
  #!/usr/bin/env bash
  COMMIT_SHA=$(git rev-parse --short HEAD || echo "unknown")
  VERSION=$(cat ./VERSION)
  docker build \
    --label pgmigrate \
    --tag local-pgmigrate \
    --build-arg COMMIT_SHA="$COMMIT_SHA" \
    --build-arg VERSION="$VERSION" \
    --file ./Dockerfile \
    .
