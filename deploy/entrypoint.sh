#!/bin/sh
set -e

mkdir -p /app/data/logs

exec /usr/bin/supervisord -n -c /etc/supervisor/conf.d/supervisord.conf
