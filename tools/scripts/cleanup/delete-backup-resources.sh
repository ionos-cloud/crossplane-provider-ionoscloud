#!/usr/bin/env bash

# ENV Vars:
# IONOS_USERNAME - username for IONOS Cloud APIs
# IONOS_PASSWORD - password for IONOS Cloud APIs

delete_all_args='--all --force'

function delete_backup_resources() {
    ionosctl backupunit delete $delete_all_args -w

    echo_step_completed
}
