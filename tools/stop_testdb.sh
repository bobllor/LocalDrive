#!/usr/bin/env bash

set -e

source "./tools/support/testdb/sql_vars.sh"
source "./tools/support/testdb/sql_utils.sh"

container_status=$(check_container_status $c_name)

if [[ "$container_status" == "true" ]]; then
    echo "Stopping container $c_name..."

    stop_docker_container $c_name
    docker volume rm $c_name | xargs -I x echo "Removing test volume x"

    test_folder="testapp"

    if [[ -e "$test_folder" ]]; then
        echo "Removing test folder $test_folder"
        rm -rf "$test_folder"
    fi
else
    echo "error: Container $c_name is not running"
fi