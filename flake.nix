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

        # Script to run VHS on tape files with yule-log in PATH
        # Note: tape files use "go run . run" so they need go in PATH.
        # yule-log binary is also available for tape files that use it directly.
        generate-gifs = pkgs.writeShellScriptBin "generate-gifs" ''
          set -e
          export PATH="${pkgs.lib.makeBinPath [ yule-log pkgs.go pkgs.git ]}:$PATH"

          # Default to current directory if no argument provided
          PROJECT_DIR="''${1:-.}"
          SCRIPTS_DIR="$PROJECT_DIR/scripts"

          if [ ! -d "$SCRIPTS_DIR" ]; then
            echo "Error: scripts directory not found at $SCRIPTS_DIR"
            echo "Usage: generate-gifs [project-dir]"
            exit 1
          fi

          cd "$SCRIPTS_DIR"

          echo "Generating GIFs from tape files..."
          echo "yule-log binary: $(which yule-log)"
          for tape in generate-gif-*.tape; do
            if [ -f "$tape" ]; then
              echo "Processing $tape..."
              ${pkgs.vhs}/bin/vhs "$tape"
            fi
          done
          echo "Done!"
        '';
      in
      {
        packages = {
          default = plugin;
          inherit yule-log plugin generate-gifs;
        };

        apps.generate-gifs = {
          type = "app";
          program = "${generate-gifs}/bin/generate-gifs";
        };

        devShells.default = pkgs.mkShell {
          buildInputs = [
            pkgs.go
            pkgs.git
            pkgs.tmux
            pkgs.vhs
            yule-log
          ];

          shellHook = ''
            echo "yule-log dev shell"
            echo "  - yule-log binary is in PATH"
            echo "  - Run 'nix run .#generate-gifs' to generate GIFs"
          '';
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
