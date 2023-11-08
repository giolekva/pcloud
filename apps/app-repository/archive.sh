#!/bin/sh

cd apps-unarchived/pihole-0.0.1
tar -czvf pihole-0.0.1.tar.gz *.md *.json *.yaml templates
cd ../../

cd apps-unarchived/rpuppy-0.0.1
tar -czvf rpuppy-0.0.1.tar.gz *.md *.json *.yaml templates
cd ../../

cd apps-unarchived/soft-serve-0.0.1
tar -czvf soft-serve-0.0.1.tar.gz *.md *.json *.yaml templates
cd ../../

cd apps-unarchived/vaultwarden-0.0.1
tar -czvf vaultwarden-0.0.1.tar.gz *.md *.json *.yaml templates
cd ../../

mv apps-unarchived/*/*.tar.gz apps
