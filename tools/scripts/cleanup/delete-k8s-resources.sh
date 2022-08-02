#!/usr/bin/env bash

# ENV Vars:
# IONOS_USERNAME - username for IONOS Cloud APIs
# IONOS_PASSWORD - password for IONOS Cloud APIs

delete_all_args='--all --force -v'

function delete_k8s_resources() {
  k8s_clusters_list=$(ionosctl k8s cluster list --cols ClusterId --no-headers)

  for cluster in $k8s_clusters_list; do
    echo_sub_step "deleting all resources from ${cluster} cluster"

    echo_info "deleting nodepools"
    ionosctl k8s nodepool delete --cluster-id $cluster $delete_all_args
  done

  echo_sub_step "deleting all clusters"
  ionosctl k8s cluster delete $delete_all_args -w

  echo_step_completed
}
