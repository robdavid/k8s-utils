{
  description = "Kubernetes deployment tools";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-26.05";
  };

  outputs = { self, nixpkgs }:
    let
      supportedSystems = [ "x86_64-linux" "aarch64-linux" "x86_64-darwin" "aarch64-darwin" ];
      forAllSystems = nixpkgs.lib.genAttrs supportedSystems;
    in
    {
      packages = forAllSystems (system:
        let
          pkgs = nixpkgs.legacyPackages.${system};
        in
        {
          default = pkgs.buildGoModule {
            pname = "k8s-utils";
            version = "0.0.1";
            src = self;
            vendorHash = "sha256-mMx5XkDrSGaLV9z7lGKpHsQ+xxIwkyFJYyCSSIsr+/U=";
            checkFlags = [ "-short" ];
          };
        }
      );
      devShells = forAllSystems (system:
        let
          pkgs = nixpkgs.legacyPackages.${system};
        in
        {
          default = pkgs.mkShell {
            packages = with pkgs; [
              go
              gopls
              delve
              govulncheck
              kubernetes-helm
              docker-client
              bashInteractive # https://discourse.nixos.org/t/interactive-bash-with-nix-develop-flake/15486
              nixd
              nil
            ];
            hardeningDisable = [ "fortify" ];
          };
        }
      );
    };
}
