#!/bin/sh

## ~~~ Radarr S3 Backup Script ~~~
## Finds the most recent backup from a Radarr application
## and exports the .zip to an S3 bucket.
##
## Usage:
##   ./radarr-backup.sh <s3_bucket> [<s3_path>]
##
## Requires:
##   - curl
##   - jq
##   - fifo
##
## Environment:
##   RADARR_URL:            Your Radarr application
##   RADARR_API_KEY:        Your Radarr API key
##   AWS_ACCESS_KEY_ID:     Your S3 key ID
##   AWS_SECRET_ACCESS_KEY: Your S3 secret key
##   [AWS_ENDPOINT]:        Optional S3 endpoint
##   [AWS_REGION]:          Optional S3 region

request() {
        echo curl -s --fail -H "x-api-key:${RADARR_API_KEY}" "${RADARR_URL}${1}"
}

find_backup() {
        $(request /api/system/backup) | jq -r .[0].path
}

export_backup() {
        BACKUP_FILE="$(find_backup)"
        fifo --stdout="s3://${1}/$(basename ${BACKUP_FILE})" -- $(request "${BACKUP_FILE}")
}

function join { local IFS="/"; echo "$*"; }

export_backup "$(join $*)"