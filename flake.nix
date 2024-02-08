{
  description = "Go Runtime for Project";

  inputs.nixpkgs.url = "github:NixOS/nixpkgs/nixpkgs-unstable";
  outputs = { nixpkgs, self }:
    let
      system = "x86_64-linux";
      pkgs = import nixpkgs { inherit system; };
      name = "twitch-bot";
      username = "mvaldes14";
    in
    {
      devShells.${system}.default = pkgs.mkShell {
        buildInputs = with pkgs;[
          go
          act
        ];
      };
      packages = {
        default = pkgs.buildGoPackage {
          inherit name;
          goPackagePath = "github.com/${username}/${name}";
          goDeps = [
            {
              goPackagePath = "github.com/gempir/go-twitch-irc";
              fetch = {
                type = "git";
                url = "https://github.com/gempir/go-twitch-irc";
                rev = "01bg6bx8ivqww56m2s73yi0991n18bsp366hd3vxic62ax7q4qy5";
                hash = "something";
              };
            }
          ];
          src = ./.;
        };
      };
    };
}
