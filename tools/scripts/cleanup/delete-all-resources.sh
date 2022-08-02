#!/usr/bin/env bash

# ENV Vars:
# IONOS_USERNAME - username for IONOS Cloud APIs
# IONOS_PASSWORD - password for IONOS Cloud APIs


# load delete functions
source ./delete-backup-resources.sh
source ./delete-dbaas-resources.sh
source ./delete-k8s-resources.sh
source ./delete-compute-engine-resources.sh

# load print utils
source ./print-utils.sh


echo_step "starting cleanup on Managed Backup resources"
delete_backup_resources

echo_step "starting cleanup on Database as a Service resources"
delete_dbaas_resources

echo_step "starting cleanup on Managed Kubernetes resources"
delete_k8s_resources

echo_step "starting cleanup on Compute Engine resources"
delete_compute_engine_resources

echo_success "Job completed"
