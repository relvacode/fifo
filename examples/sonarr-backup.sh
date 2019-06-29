#!/bin/sh

## ~~~ Sonarr S3 Backup Script ~~~
## Finds the most recent backup from a Sonarr application
## and exports the .zip to an S3 bucket.
##
## Usage:
##   ./sonarr-backup.sh <s3_bucket> [<s3_path>]
##
## Requires:
##   - curl
##   - jq
##   - fifo
##
## Environment:
##   SONARR_URL:            Your Sonarr application
##   SONARR_API_KEY:        Your Sonarr API key
##   AWS_ACCESS_KEY_ID:     Your S3 key ID
##   AWS_SECRET_ACCESS_KEY: Your S3 secret key
##   [AWS_ENDPOINT]:        Optional S3 endpoint
##   [AWS_REGION]:          Optional S3 region

request() {
        echo curl -s --fail -H "x-api-key:${SONARR_API_KEY}" "${SONARR_URL}${1}"
}

find_backup() {
        $(request /api/v3/system/backup) | jq -r .[0].path
}

export_backup() {
        BACKUP_FILE="$(find_backup)"
        fifo --stdout="s3://${1}/$(basename ${BACKUP_FILE})" -- $(request "${BACKUP_FILE}")
}

function join { local IFS="/"; echo "$*"; }

export_backup "$(join $*)"