#!/bin/bash

set -e

username="stanza"

if id "$username" &>/dev/null; then
    # Skip all user config if already exists
    echo "User ${username} already exists"
    exit 0
fi

useradd --shell /sbin/nologin --create-home --system "$username"
