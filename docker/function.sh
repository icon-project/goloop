#!/bin/sh

get_hash_of_files() {
    cat "$@" | sha1sum | cut -d " " -f1
}

get_hash_of_any() {
    for item in "$@"; do
        if [ "${item#@}" == "${item}" ] ; then
          echo "${item}"
        else
          cat "${item#@}"
        fi
    done | sha1sum | cut -d " " -f1
}

get_label_of_image() {
    local LABEL=$1
    local TAG=$2
    docker image inspect -f "{{.Config.Labels.${LABEL}}}" ${TAG} 2> /dev/null || echo 'none'
}

get_id_with_hash() {
    local REPO=$1
    local LABEL=$2
    local HASH=$3
    docker images --filter="reference=${REPO}" --filter="label=${LABEL}=${HASH}" --format="{{.ID}}" | head -n 1
}