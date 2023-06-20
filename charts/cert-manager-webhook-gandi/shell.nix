{ pkgs ? import <nixpkgs> {} }:

let
  unstable = import (builtins.fetchTarball {
    name = "nixos-unstable-2021-09-12";
    url = "https://github.com/nixos/nixpkgs/archive/2ce4d21663113020195f1d953e360213954645b3.tar.gz";
    sha256 = "15pnbmm702a4ni8dm2jdwl46b20qw7gfm5chlrvn7w54cm3h9p0c";
  }) {};
in pkgs.mkShell {
  buildInputs = with unstable; [
    go_1_16
    gopls
  ];
}
