#!/usr/bin/env bash

# ENV Vars:
# IONOS_USERNAME - username for IONOS Cloud APIs
# IONOS_PASSWORD - password for IONOS Cloud APIs

delete_all_args='--all --force'

function delete_dbaas_resources() {
  ionosctl dbaas postgres cluster delete $delete_all_args -W

  echo_step_completed
}
