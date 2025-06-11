{
  description = "Twitch Bot Environment and build tools";

  inputs.nixpkgs.url = "github:NixOS/nixpkgs/nixpkgs-unstable";
  outputs = inputs @ {flake-parts, ...}:
    flake-parts.lib.mkFlake {inherit inputs;} {
      systems = ["x86_64-linux" "aarch64-linux" "aarch64-darwin" "x86_64-darwin"];

      perSystem = {
        config,
        self',
        inputs',
        pkgs,
        system,
        ...
      }: let
        name = "twitch-bot";
        vendorHash = "sha256-QF+4UZsc7V8togJNUUV3f154lXNznRzZTUYTPOSR4ds=";
        version = "0.1.0";
      in {
        devShells = {
          default = pkgs.mkShell {
            inputsFrom = [self'.packages.default];
            nativeBuildInputs = with pkgs; [air act gopls gotools revive prettierd];
          };
        };
        packages = {
          default = pkgs.buildGoModule {
            inherit name vendorHash;
            src = ./.;
            subPackages = ["cmd/bot"];
          };

          docker = pkgs.dockerTools.buildImage {
            inherit name;
            tag = version;
            config = {
              Cmd = "${self'.packages.default}/bin/${name}";
              Env = [
                "SSL_CERT_FILE=${pkgs.cacert}/etc/ssl/certs/ca-bundle.crt"
              ];
            };
          };
        };
      };
    };
}
