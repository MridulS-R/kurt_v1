# Auto-update SHA256 values after each release with: brew audit --fix
#
# This is a placeholder formula. URLs and SHA256 values must be replaced
# with the real release artifacts before the tap is usable.
class Kurt < Formula
  desc "Fast, modular shell prompt and AI assistant CLI"
  homepage "https://github.com/strk/kurt"
  license "MIT"
  version "0.0.0"

  on_macos do
    on_arm do
      url "https://github.com/strk/kurt/releases/download/v0.0.0/kurt_darwin_arm64"
      sha256 "0000000000000000000000000000000000000000000000000000000000000000"
    end
    on_intel do
      url "https://github.com/strk/kurt/releases/download/v0.0.0/kurt_darwin_amd64"
      sha256 "0000000000000000000000000000000000000000000000000000000000000000"
    end
  end

  on_linux do
    on_arm do
      url "https://github.com/strk/kurt/releases/download/v0.0.0/kurt_linux_arm64"
      sha256 "0000000000000000000000000000000000000000000000000000000000000000"
    end
    on_intel do
      url "https://github.com/strk/kurt/releases/download/v0.0.0/kurt_linux_amd64"
      sha256 "0000000000000000000000000000000000000000000000000000000000000000"
    end
  end

  def install
    bin.install "kurt"
  end

  test do
    system "#{bin}/kurt", "version"
  end
end
