#!/bin/sh
# This script is the post-build script for goreleaser.
# Because it is run for each binary built, and global post-release hooks are 
# not available in the OSS version, we check for file existence to avoid multiple downloads/copies.

if [ ! -f ./artifacts/stanza-plugins.tar.gz ]; then
    curl -fL https://github.com/observiq/stanza-plugins/releases/latest/download/stanza-plugins.tar.gz -o ./artifacts/stanza-plugins.tar.gz
fi
if [ ! -f ./artifacts/stanza-plugins.zip ]; then
    curl -fL https://github.com/observiq/stanza-plugins/releases/latest/download/stanza-plugins.tar.gz -o ./artifacts/stanza-plugins.tar.gz
fi
if [ ! -f ./artifacts/version.json ]; then
    curl -fL https://github.com/observiq/stanza-plugins/releases/latest/download/version.json -o ./artifacts/version.json
fi
if [ ! -f ./artifacts/unix-install.sh ]; then
    cp ./scripts/unix-install.sh ./artifacts/unix-install.sh
fi
if [ ! -f ./artifacts/windows-install.ps1 ]; then
    cp ./scripts/windows-install.ps1 ./artifacts/windows-install.ps1
fi

tar -xf ./artifacts/stanza-plugins.tar.gz -C ./artifacts