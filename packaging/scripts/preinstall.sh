#!/bin/sh
set -e

if [ -d /run/systemd/system ]; then
    systemctl daemon-reload

    if systemctl is-enabled --quiet icahazip.service; then
        systemctl restart icahazip.service
    fi
fi