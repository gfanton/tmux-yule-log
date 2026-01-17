{
  description = "Yule Log - A tmux screensaver plugin with fire animation and git commit ticker";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixpkgs-unstable";
    flake-utils.url = "github:numtide/flake-utils";
  };

  outputs =
    {
      self,
      nixpkgs,
      flake-utils,
    }:
    flake-utils.lib.eachDefaultSystem (
      system:
      let
        pkgs = nixpkgs.legacyPackages.${system};

        # Single Go binary
        yule-log = pkgs.buildGoModule rec {
          pname = "yule-log";
          version = "0.1.0";
          src = ./.;

          vendorHash = "sha256-Fdnu2rnD604aNMpgpkIH9tCV4iCZRWA+gFUXkPDvEoc=";

          nativeBuildInputs = [ pkgs.makeWrapper ];

          postInstall = ''
            wrapProgram $out/bin/yule-log \
              --prefix PATH : ${pkgs.lib.makeBinPath [ pkgs.git pkgs.tmux ]}
          '';

          meta = with pkgs.lib; {
            description = "Yule Log screensaver for tmux with fire animation and git commit ticker";
            homepage = "https://github.com/gfanton/tmux-yule-log";
            license = licenses.mit;
            platforms = platforms.unix;
          };
        };

        # Tmux plugin package
        plugin = pkgs.tmuxPlugins.mkTmuxPlugin {
          pluginName = "tmux-yule-log";
          version = "0.1.0";
          src = ./.;
          rtpFilePath = "yule-log.tmux";

          nativeBuildInputs = [ pkgs.makeWrapper ];

          postInstall = ''
            # Link binary into plugin's bin/ directory
            mkdir -p $out/share/tmux-plugins/tmux-yule-log/bin
            rm -f $out/share/tmux-plugins/tmux-yule-log/bin/yule-log
            ln -s ${yule-log}/bin/yule-log $out/share/tmux-plugins/tmux-yule-log/bin/yule-log

            # Make scripts executable and wrap with dependencies
            chmod +x $out/share/tmux-plugins/tmux-yule-log/yule-log.tmux
            chmod +x $out/share/tmux-plugins/tmux-yule-log/scripts/tmux/*.sh

            wrapProgram $out/share/tmux-plugins/tmux-yule-log/yule-log.tmux \
              --prefix PATH : ${pkgs.lib.makeBinPath [
                pkgs.tmux
                pkgs.git
                pkgs.bash
                pkgs.coreutils
              ]}
          '';

          meta = with pkgs.lib; {
            description = "A tmux screensaver plugin with fire animation and git commit ticker";
            homepage = "https://github.com/gfanton/tmux-yule-log";
            license = licenses.mit;
            platforms = platforms.unix;
          };
        };
      in
      {
        packages = {
          default = plugin;
          inherit yule-log plugin;
        };
      }
    )
    // {
      overlays.default = final: prev: {
        tmuxPlugins = prev.tmuxPlugins // {
          tmux-yule-log = self.packages.${final.system}.plugin;
        };
      };
    };
}
