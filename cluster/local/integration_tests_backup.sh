#!/usr/bin/env bash

set -e

## The purpose of this script is to have the tests for
## the Backup resources.
## Please name the functions the following format:
## <resource_name>_tests() and <resource_name>_tests_cleanup().

## BackupUnit CR Tests
function backupunit_tests() {
  echo_step "deploy a backupunit CR"
  INSTALL_RESOURCE_YAML="$(
    cat <<EOF
apiVersion: backup.ionoscloud.crossplane.io/v1alpha1
kind: BackupUnit
metadata:
  name: example
spec:
  managementPolicies:
    - "*"
  forProvider:
    name: exampleBackupUnit
    email: test12345@gmail.com
    password: "${TEST_IMAGE_PASSWORD}"
  providerConfigRef:
    name: example
EOF
  )"

  echo "${INSTALL_RESOURCE_YAML}" | "${KUBECTL}" apply -f -

  echo_step "waiting for backupunit CR to be ready & synced"
  kubectl wait --for=condition=ready backupunits/example
  kubectl wait --for=condition=synced backupunits/example

  echo_step "get backupunit CR"
  kubectl get backupunits

  echo_step "update backupunit CR"
  INSTALL_RESOURCE_YAML="$(
    cat <<EOF
apiVersion: backup.ionoscloud.crossplane.io/v1alpha1
kind: BackupUnit
metadata:
  name: example
spec:
  managementPolicies:
    - "*"
  forProvider:
    name: exampleBackupUnit
    email: test123456@gmail.com
    password: "${TEST_IMAGE_PASSWORD}"
  providerConfigRef:
    name: example
EOF
  )"

  echo "${INSTALL_RESOURCE_YAML}" | "${KUBECTL}" apply -f -

  echo_step "waiting for backupunit CR to be ready & synced"
  kubectl wait --for=condition=ready backupunits/example
  kubectl wait --for=condition=synced backupunits/example
}

function backupunit_tests_cleanup() {
  INSTALL_RESOURCE_YAML="$(
    cat <<EOF
apiVersion: backup.ionoscloud.crossplane.io/v1alpha1
kind: BackupUnit
metadata:
  name: example
spec:
  managementPolicies:
    - "*"
  forProvider:
    name: exampleBackupUnit
    email: test123456@gmail.com
    password: "${TEST_IMAGE_PASSWORD}"
  providerConfigRef:
    name: example
EOF
  )"

  echo_step "uninstalling backupunit CR"
  echo "${INSTALL_RESOURCE_YAML}" | "${KUBECTL}" delete -f -

  echo_step "wait for deletion backupunit CR"
  kubectl wait --for=delete backupunits/example
}
