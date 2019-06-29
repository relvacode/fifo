#!/bin/sh

## ~~~ Plex Backup Script ~~~
## Streams the Plex application directory (excluding Cache)
## into a compressed tar archive and uploads it
## to a remote S3 bucket using the current date.
##
## Usage:
##   ./plex-backup.sh <bucket> [<plex_data_directory>]
##
## Requires:
##   - tar
##   - fifo
##
## Environment:
##   AWS_ACCESS_KEY_ID:     Your S3 key ID
##   AWS_SECRET_ACCESS_KEY: Your S3 secret key
##   [AWS_ENDPOINT]:        Optional S3 endpoint
##   [AWS_REGION]:          Optional S3 region

export_data() {
    fifo -t archive=s3://${1}/plex-backup-$(date +%Y-%m-%d).tar.gz -- tar -C "${2}" --exclude 'Cache/*' -cvz . -f %{archive}
}

PLEX_DATA_DIRECTORY="${2:-/var/lib/plexmediaserver/Library/Application Support/Plex Media Server}"
export_data "${1}" "${PLEX_DATA_DIRECTORY}"