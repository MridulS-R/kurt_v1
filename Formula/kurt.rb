# To update after a new release:
#   1. Download checksums.txt from the release
#   2. Replace sha256 values and version below
class Kurt < Formula
  desc "Fast modular shell prompt with built-in AI (think, diff, RAG, vision, eval)"
  homepage "https://github.com/MridulS-R/kurt_v1"
  license "MIT"
  version "0.1.0"

  on_macos do
    on_arm do
      url "https://github.com/MridulS-R/kurt_v1/releases/download/v0.1.0/kurt_darwin_arm64"
      sha256 "91d89f5d67a2ca72d75929c1bda5c319e01c4c7569b777ab5333ee5f76596ec2"
    end
    on_intel do
      url "https://github.com/MridulS-R/kurt_v1/releases/download/v0.1.0/kurt_darwin_amd64"
      sha256 "1c813e829eb2edb4735962fb19b6d20fbf3d3cb2ecef7a421e10b8a0d40629f4"
    end
  end

  on_linux do
    on_arm do
      url "https://github.com/MridulS-R/kurt_v1/releases/download/v0.1.0/kurt_linux_arm64"
      sha256 "15e86be23ba01274b217fdbe1805020aa39b654537fbe7e325c00bb725d09159"
    end
    on_intel do
      url "https://github.com/MridulS-R/kurt_v1/releases/download/v0.1.0/kurt_linux_amd64"
      sha256 "b4fe2578a4164bf30d68c07d6128eab5d768d41c36c8dcdcd56534064507dd9d"
    end
  end

  def install
    arch = Hardware::CPU.arm? ? "arm64" : "amd64"
    bin.install "kurt_#{OS.kernel_name.downcase}_#{arch}" => "kurt"
  end

  test do
    assert_match version.to_s, shell_output("#{bin}/kurt version")
  end
end
