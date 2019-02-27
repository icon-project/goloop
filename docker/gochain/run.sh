#!/bin/bash

GOCHAIN="./gochain"

GOCHAIN_DATA=${GOCHAIN_DATA:-"./data"}
GOCHAIN_CONFIG=${GOCHAIN_CONFIG:-"./config.json"}

if [ ! -d ${GOCHAIN_DATA} ] ; then
    mkdir -p ${GOCHAIN_DATA} || exit -1
fi

GOCHAIN_OPTIONS=(-node_dir "${GOCHAIN_DATA}")
GOCHAIN_OPTIONS+=(-db_type goleveldb)
GOCHAIN_OPTIONS+=(-role 3)

if [ "${GOCHAIN_GENESIS}" != "" ] ; then
    GOCHAIN_OPTIONS+=(-genesis "${GOCHAIN_GENESIS}")
fi

if [ "${GOCHAIN_GENESIS_DATA}" != "" ] ; then
    GOCHAIN_OPTIONS+=(-genesis_data "${GOCHAIN_GENESIS_DATA}")
fi

if [ "${GOCHAIN_ADDRESS}" != "" ] ; then
    GOCHAIN_OPTIONS+=(-p2p ${GOCHAIN_ADDRESS})
else
    GOCHAIN_OPTIONS+=(-p2p "${HOST}:8080")
fi
GOCHAIN_OPTIONS+=(-p2p_listen ":8080")

if [ ! -f "${GOCHAIN_CONFIG}" ] ; then
    ${GOCHAIN} "${GOCHAIN_OPTIONS[@]}" \
        -save ${GOCHAIN_CONFIG} || exit -1
fi

${GOCHAIN} -config ${GOCHAIN_CONFIG} "${GOCHAIN_OPTIONS[@]}"
