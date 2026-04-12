class CubGen < Formula
  desc "Repo-side traceability CLI for GitOps config"
  homepage "https://github.com/confighub/cub-gen"
  version "0.3.0"

  on_macos do
    if Hardware::CPU.arm?
      url "https://github.com/confighub/cub-gen/releases/download/v0.3.0/cub-gen_0.3.0_darwin_arm64.tar.gz"
      sha256 "8abd8a9cfbb907f01be90d22dcd58003f79713bf0e3697da3ee008aa18d9a9e4"
    else
      url "https://github.com/confighub/cub-gen/releases/download/v0.3.0/cub-gen_0.3.0_darwin_amd64.tar.gz"
      sha256 "9a2a8138294862e5664325347671a60242fe500af7322e3ea45021e2172f9829"
    end
  end

  on_linux do
    if Hardware::CPU.arm?
      url "https://github.com/confighub/cub-gen/releases/download/v0.3.0/cub-gen_0.3.0_linux_arm64.tar.gz"
      sha256 "0cf51603a1470557d9c550d0d90ebae5ee11701645939caa6e257f27d2349711"
    else
      url "https://github.com/confighub/cub-gen/releases/download/v0.3.0/cub-gen_0.3.0_linux_amd64.tar.gz"
      sha256 "d9b9e3ec2c9af0855e73d298dac804c677236f0c7d955708db8ccca2364ad2bf"
    end
  end

  def install
    bin.install "cub-gen"
  end

  test do
    output = shell_output("#{bin}/cub-gen generators --json")
    assert_match "\"kind\": \"helm\"", output
    assert_match "\"count\": 9", output
  end
end
