#!/bin/bash

set -e

rpm_install() {
    sudo rpm -i './dist/stanza_*_linux_amd64.rpm'
}

deb_install() {
    sudo apt-get install -y -f ./dist/stanza_*_linux_amd64.deb
}

start() {
    # Stanza can install on Centos 6 but we do not support the
    # centos 6 init system.
    if command -v systemctl &> /dev/null; then
        sudo systemctl enable stanza
        sudo systemctl start stanza
    fi
}

if command -v "dpkg" > /dev/null ; then
    deb_install
elif command -v "rpm" > /dev/null ; then
    rpm_install
else
    echo "failed to detect plaform type"
    exit 1
fi
start