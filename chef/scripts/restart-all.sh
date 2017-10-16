#!/bin/sh

set -e
lang=${1:-perl}
chmod 755 /var/log/nginx
touch /var/log/nginx/access.log
touch /var/log/mysql/slow.log
mv /var/log/nginx/access.log /var/log/nginx/access.log.`date +%Y%m%d-%H%M%S`
mv /var/log/mysql/slow.log /var/log/mysql/slow.log.`date +%Y%m%d-%H%M%S`
# mysqladmin flush-logs -uroot
systemctl restart isuda.$lang
systemctl restart isutar.$lang
systemctl reload nginx
systemctl status nginx
journalctl -f -u isuda.$lang
