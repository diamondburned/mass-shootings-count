{
	pkgs ? import <nixpkgs> {
		overlays = [
			(self: super: {
				go = super.go_1_18;
				buildGoModule = super.buildGo118Module;
			})
		];
	},
}:

pkgs.mkShell {
	buildInputs = with pkgs; [
		go
		gopls
		gotools
	];
}
