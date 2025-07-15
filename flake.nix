{
  description = "pgmigrate is a modern Postgres migrations CLI and library for golang";
  inputs = {
    nixpkgs.url = "github:nixos/nixpkgs";

    flake-utils.url = "github:numtide/flake-utils";

    flake-compat.url = "github:edolstra/flake-compat";
    flake-compat.flake = false;

    nix-filter.url = "github:numtide/nix-filter";
  };

  outputs = { self, ... }@inputs:
    inputs.flake-utils.lib.eachDefaultSystem (system:
      let
        overlays = [ ];
        pkgs = import inputs.nixpkgs {
          inherit system overlays;
        };
        lib = pkgs.lib;
        version = (builtins.readFile ./VERSION);
        commit = if (builtins.hasAttr "rev" self) then (builtins.substring 0 7 self.rev) else "unknown";
      in
      rec {
        packages = rec {
          pgmigrate = pkgs.buildGo124Module {
            pname = "pgmigrate";
            version = version;
            # Every time you update your dependencies (go.mod / go.sum)  you'll
            # need to update the vendorSha256.
            #
            # To find the right hash, set
            #
            #   vendorHash = pkgs.lib.fakeHash;
            #
            # then run `nix build`, take the correct hash from the output, and set
            #
            #   vendorHash = <the updated hash>;
            #
            # (Yes, that's really how you're expected to do this.)
            #vendorHash = pkgs.lib.fakeHash;
            vendorHash = "sha256-wyy3GoUulLp0OvYIJBGm/QMbmvDVTi0nJjxMNXuPeGI=";
            GOWORK = "off";
            src =
              let
                # Set this to `true` in order to show all of the source files
                # that will be included in the module build.
                debug-tracing = false;
                source-files = inputs.nix-filter.lib.filter {
                  root = ./cmd/pgmigrate;
                };
              in
              (
                if (debug-tracing) then
                  pkgs.lib.sources.trace source-files
                else
                  source-files
              );

            # Add any extra packages required to build the binaries should go here.
            buildInputs = [ ];
            ldflags = [
              "-X github.com/peterldowns/pgmigrate/cmd/pgmigrate/shared.Version=${version}"
              "-X github.com/peterldowns/pgmigrate/cmd/pgmigrate/shared.Commit=${commit}"
            ];
            modRoot = ".";
            doCheck = false;
          };
          default = pgmigrate;
        };
        apps = rec {
          pgmigrate = {
            type = "app";
            program = "${packages.pgmigrate}/bin/pgmigrate";
          };
          default = pgmigrate;
        };
        devShells = rec {
          default = pkgs.mkShell {
            buildInputs = [ ];
            packages = with pkgs; [
              # Go
              delve
              go-outline
              go_1_24
              golangci-lint
              gopkgs
              gopls
              gotools
              # Nix
              nixpkgs-fmt
              # Other
              just
              postgresql
              docker
            ];

            shellHook = ''
              # The path to this repository
              shell_nix="''${IN_LORRI_SHELL:-$(pwd)/shell.nix}"
              workspace_root=$(dirname "$shell_nix")
              export WORKSPACE_ROOT="$workspace_root"

              # Puts the $GOPATH/$GOCACHE/$GOENV in $TOOLCHAIN_ROOT,
              # and ensures that the GOPATH's bin dir is on the PATH so tools
              # can be installed with `go install`.
              #
              # Any tools installed explicitly with `go install` will take precedence
              # over versions installed by Nix due to the ordering here.
              #
              # Puts the toolchain folder adjacent to the repo so that tools
              # running inside the repo don't ever scan its contents.
              export TOOLCHAIN_NAME=".toolchain-$(basename $WORKSPACE_ROOT)"
              export TOOLCHAIN_ROOT="$(dirname $WORKSPACE_ROOT)/$TOOLCHAIN_NAME"
              export GOROOT=
              export GOCACHE="$TOOLCHAIN_ROOT/go/cache"
              export GOENV="$TOOLCHAIN_ROOT/go/env"
              export GOPATH="$TOOLCHAIN_ROOT/go/path"
              export GOMODCACHE="$GOPATH/pkg/mod"
              export PATH=$(go env GOPATH)/bin:$PATH
              export CGO_ENABLED=0

              # Make it easy to test while developing by adding the built binary to
              # the PATH.
              export PATH="$workspace_root/bin:$workspace_root/result/bin:$PATH"
              # For testing purposes
              export MIGRATIONS='internal/migrations'
              export DATABASE='postgres://postgres:password@localhost:5433/postgres'
            '';

            # Need to disable fortify hardening because GCC is not built with -oO,
            # which means that if CGO_ENABLED=1 (which it is by default) then the golang
            # debugger fails.
            # see https://github.com/NixOS/nixpkgs/pull/12895/files
            hardeningDisable = [ "fortify" ];
          };
        };
      }
    );
}
