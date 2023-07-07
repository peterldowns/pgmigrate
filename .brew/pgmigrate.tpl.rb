class Pgmigrate < Formula
  desc "Pgmigrate is a modern Postgres migrations CLI and library for golang."
  homepage "https://github.com/peterldowns/pgmigrate"
  url "$SOURCE_URL"
  sha256 "$SOURCE_SHA_256"
  license "MIT"
  version "$VERSION"

  depends_on "go" => :build

  def install
    # Parse the version and commit from the tagref URL because the downloaded
    # .tar.gz isn't a git repository.
    tag_name = "$TAG_NAME"
    version = "$VERSION"
    commit = "$COMMIT"
    ldflags = "$LDFLAGS"
    # -s -w is standard to make small binaries without debugging information or symbol tables
    # https://stackoverflow.com/a/22276273/829926
    # std_go_args definition is here
    # https://github.com/Homebrew/brew/blob/6db7732fa33ab808e405f8ac7673735edd2c8787/Library/Homebrew/formula.rb#L1565
    system "go", "build", *std_go_args(ldflags: "-s -w " + ldflags, output: bin/"pgmigrate"), "./cmd/pgmigrate"
  end

  test do
    system bin/"pgmigrate", "--help"
  end
end

