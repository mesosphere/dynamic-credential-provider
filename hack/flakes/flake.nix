{
  description = "Useful flakes for golang and Kubernetes projects";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixpkgs-unstable";
    flake-utils.url = "github:numtide/flake-utils";
  };

  outputs = inputs @ { self, nixpkgs, flake-utils }:
    flake-utils.lib.eachDefaultSystem (system:
      with nixpkgs.legacyPackages.${system}; rec {
        packages = rec {
          govulncheck = pkgs.govulncheck.override { buildGoModule = buildGo124Module; };

          golangci-lint = buildGo124Module rec {
            pname = "golangci-lint";
            version = "1.64.5";

            src = fetchFromGitHub {
              owner = "golangci";
              repo = "golangci-lint";
              rev = "v${version}";
              hash = "sha256-PRI82Ia2R2GH9xV/UZvfXTmCrfsxvHfysXuAek/4a+0=";
            };

            vendorHash = "sha256-oCaVXjflmOMUDEDynbnUwA9KOPNDcEwI4WqOi2KoCG4=";

            subPackages = [ "cmd/golangci-lint" ];

            ldflags = [
              "-s"
              "-X main.version=${version}"
              "-X main.commit=v${version}"
              "-X main.date=19700101-00:00:00"
            ];
          };
        };

        formatter = alejandra;
      }
    );
}
