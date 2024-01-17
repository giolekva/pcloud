#!/bin/sh

rm -rf apps/*
cp -r ../../core/installer/values-tmpl tmp
cd tmp

tar -czvf jellyfin-0.0.1.tar.gz jellyfin.cue
tar -czvf matrix-0.0.1.tar.gz matrix.cue
tar -czvf penpot-0.0.1.tar.gz penpot.cue
tar -czvf pihole-0.0.1.tar.gz pihole.cue
tar -czvf qbittorrent-0.0.1.tar.gz qbittorrent.cue
tar -czvf rpuppy-0.0.1.tar.gz rpuppy.cue
tar -czvf soft-serve-0.0.1.tar.gz soft-serve.cue
tar -czvf vaultwarden-0.0.1.tar.gz vaultwarden.cue

mv *.tar.gz ../apps

cd ../
rm -rf tmp
