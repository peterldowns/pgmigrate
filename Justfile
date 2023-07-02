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
  go build -o bin/pgmigrate ./cli

# test all packages
test *args='./... ./cli/...':
  go test -race -count=1 $@

# lint pgmigrate
lint *args='./... ./cli/...':
  golangci-lint run --fix --config .golangci.yaml $@

# lint nix files
lint-nix:
  find . -name '*.nix' | xargs nixpkgs-fmt

# tag pgtestdb with current version
tag:
  #!/usr/bin/env zsh
  raw="$(cat VERSION)"
  git tag "$raw"
  git tag "cli/$raw"

tidy:
  #!/usr/bin/env zsh
  go mod tidy
  pushd cli && go mod tidy && popd
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

# builds and pushes tbd-dbtools/migrate, tagged with :latest and :$COMMIT_SHA
build-docker:
  #!/usr/bin/env bash
  COMMIT_SHA=$(git log -1 | head -1 | cut -f 2 -d ' ')
  VERSION=$(cat ./VERSION)
  docker buildx build \
    --platform linux/arm64,linux/amd64 \
    --label pgmigrate \
    --tag ghcr.io/peterldowns/pgmigrate:"$COMMIT_SHA" \
    --tag ghcr.io/peterldowns/pgmigrate:"$VERSION-commit.$COMMIT_SHA" \
    --tag ghcr.io/peterldowns/pgmigrate:latest \
    --tag pgmigrate \
    --cache-from ghcr.io/peterldowns/pgmigrate:latest \
    --build-arg COMMIT_SHA="$COMMIT_SHA" \
    --build-arg VERSION="$VERSION" \
    --file ./Dockerfile \
    --output type=image,push=false \
    .
