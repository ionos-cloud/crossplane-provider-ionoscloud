#!/usr/bin/env bash

# ENV Vars:
# IONOS_USERNAME - username for IONOS Cloud APIs
# IONOS_PASSWORD - password for IONOS Cloud APIs

delete_all_args='--all --force'

function delete_compute_engine_resources() {
  dc_list=$(ionosctl datacenter list --no-headers --cols DatacenterId)

  for datacenter in $dc_list; do

    server_list=$(ionosctl server list --datacenter-id $datacenter --no-headers --cols ServerId)

    for server in $server_list; do
      echo_sub_step "deleting all resources from ${server} server, ${datacenter} datacenter"

        nic_list=$(ionosctl nic list --datacenter-id $datacenter --server-id $server --no-headers --cols NicId)
        for nic in $nic_list; do
          echo_info "[INFO] deleting firewall rules from: nic ${nic}"
          ionosctl flowlog delete --datacenter-id $datacenter --server-id $server --nic-id $nic $delete_all_args
        done

      echo_info "[INFO] deleting nics"
      ionosctl nic delete --datacenter-id $datacenter --server-id $server $delete_all_args

      echo_info "[INFO] detaching volumes"
      ionosctl server volume detach --datacenter-id $datacenter --server-id $server $delete_all_args

      echo_info "[INFO] deleting firewallrules"
      ionosctl firewallrule delete --datacenter-id $datacenter --server-id $server $delete_all_args

    done

    echo_info "[INFO] deleting lans"
    ionosctl lan delete --datacenter-id $datacenter $delete_all_args

    echo_info "[INFO] deleting lans"
    ionosctl volume delete --datacenter-id $datacenter $delete_all_args

    echo_info "[INFO] deleting lans"
    ionosctl server delete --datacenter-id $datacenter $delete_all_args

    echo_info "[INFO] deleting lans"
    ionosctl loadbalancer delete --datacenter-id $datacenter $delete_all_args

    echo_info "[INFO] deleting datacenter ${datacenter}"
    ionosctl datacenter delete --datacenter-id $datacenter --force

  done

  echo_info "[INFO] deleting snapshots"
  ionosctl snapshot delete $delete_all_args

  echo_info "[INFO] deleting ipblocks"
  ionosctl ipblock delete $delete_all_args

  echo_info "[INFO] deleting pccs"
  ionosctl pcc delete $delete_all_args

  echo_step_completed
}
