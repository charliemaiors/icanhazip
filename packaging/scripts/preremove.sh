#!/bin/sh
set -e

systemctl stop icahazip.service || true
systemctl disable icahazip.service || true