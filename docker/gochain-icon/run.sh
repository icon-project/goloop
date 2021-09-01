#!/bin/sh
#
# Copyright 2021 ICON Foundation
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
#

#
#   GOCHAIN autostart script
#
#   Usable environment variables.
#
#   GOCHAIN_DATA    (default:"./data")
#                   Path to to store BlockData, Contracts and WAL.
#
#   GOCHAIN_CONFIG  (default:"./config.json")
#                   Path to configuration file. If it doesn't exist, it will
#                   automatically generate one.
#
#   GOCHAIN_DB_TYPE (default:"goleveldb")
#                   Name of database system.
#
#   GOCHAIN_GENESIS (optional)
#                   Path to the genesis transaction or template file.
#                   It will override configuration file.
#
#   GOCHAIN_GENESIS_STORAGE (optional)
#                   Path to the genesis storage.
#
#   GOCHAIN_ADDRESS (recommended, default:"$HOST:8080")
#                   Address for the node, "<host ip or name>:<port>", which is
#                   used by other nodes.
#
#   GOCHAIN_KEYSTORE (default:"./keystore.json")
#                   It tries to load specified keychain if it's exist.
#                   Otherwise, it exports to the file.
#
#   GOCHAIN_KEYSECRET (recommended)
#                   File path including password for KeyStore.
#                   It overrides GOCHAIN_KEYPASSWORD.
#
#   GOCHAIN_KEYAUTO (recommended)
#                   Set it as "1" to enable automatic generation of KeySecret
#                   through user input.
#
#   GOCHAIN_LOGFILE (optional)
#                   Path to Log using 'tee'
set -e

GOCHAIN="gochain"

GOCHAIN_DATA=${GOCHAIN_DATA:-"./data"}
GOCHAIN_CONFIG=${GOCHAIN_CONFIG:-"./config.json"}
GOCHAIN_KEYSTORE=${GOCHAIN_KEYSTORE:-"./keystore.json"}
GOCHAIN_DB_TYPE=${GOCHAIN_DB_TYPE:-"goleveldb"}

if [ ${GOCHAIN_CLEAN_DATA} == "true" ] ; then
    rm -rf ${GOCHAIN_DATA} || exit 1
    if [ "${GOCHAIN_LOGFILE}" != "" ] ; then
        rm -rf "${GOCHAIN_LOGFILE}" || exit 1
    fi
fi
if [ ! -d ${GOCHAIN_DATA} ] ; then
    mkdir -p ${GOCHAIN_DATA} || exit 1
fi

GOCHAIN_OPTIONS="--chain_dir ${GOCHAIN_DATA}"
GOCHAIN_OPTIONS="$GOCHAIN_OPTIONS --ee_socket /tmp/socket"
GOCHAIN_OPTIONS="$GOCHAIN_OPTIONS --role 3"
GOCHAIN_OPTIONS="$GOCHAIN_OPTIONS --platform icon"

if [ "${GOCHAIN_DB_TYPE}" != "" ] ; then
    GOCHAIN_OPTIONS="$GOCHAIN_OPTIONS --db_type ${GOCHAIN_DB_TYPE}"
fi

if [ "${GOCHAIN_GENESIS}" != "" ] ; then
    GOCHAIN_OPTIONS="$GOCHAIN_OPTIONS --genesis ${GOCHAIN_GENESIS}"
fi

if [ "${GOCHAIN_ENGINES}" != "" ] ; then
    GOCHAIN_OPTIONS="$GOCHAIN_OPTIONS --engines ${GOCHAIN_ENGINES}"
fi

if [ "${GOCHAIN_GENESIS_STORAGE}" != "" ] ; then
    GOCHAIN_OPTIONS="$GOCHAIN_OPTIONS --genesis_storage ${GOCHAIN_GENESIS_STORAGE}"
fi

if [ -r "${GOCHAIN_KEYSTORE}" ] ; then
    GOCHAIN_OPTIONS="$GOCHAIN_OPTIONS --key_store ${GOCHAIN_KEYSTORE}"
else
    GOCHAIN_OPTIONS="$GOCHAIN_OPTIONS --save_key_store ${GOCHAIN_KEYSTORE}"
fi

if [ "${GOCHAIN_KEYAUTO}" == "1" ] ; then
    if [ ! -r "${GOCHAIN_KEYSECRET:="./keystore.secret"}" ] ; then
        echo "[#] Creating secret file"
        read -sp "Password:" GOCHAIN_PASSWORD
        echo ""
        echo -n "${GOCHAIN_PASSWORD}" > "${GOCHAIN_KEYSECRET}"
    fi
fi

if [ "${GOCHAIN_KEYSECRET}" != "" ] ; then
    GOCHAIN_OPTIONS="$GOCHAIN_OPTIONS --key_secret ${GOCHAIN_KEYSECRET}"
fi

if [ "${GOCHAIN_ADDRESS}" != "" ] ; then
    GOCHAIN_OPTIONS="$GOCHAIN_OPTIONS --p2p ${GOCHAIN_ADDRESS}"
else
    GOCHAIN_OPTIONS="$GOCHAIN_OPTIONS --p2p ${HOST}:8080"
fi
GOCHAIN_OPTIONS="$GOCHAIN_OPTIONS --p2p_listen :8080"

if [ ! -r "${GOCHAIN_CONFIG}" ] ; then
    echo "[!] Generate config path=[${GOCHAIN_CONFIG}]"
    ${GOCHAIN} ${GOCHAIN_OPTIONS} --save ${GOCHAIN_CONFIG} || exit 1
fi

if [ "${GOCHAIN_LOGFILE}" != "" ] ; then
  GOCHAIN_LOGDIR=$(dirname ${GOCHAIN_LOGFILE})
  if [ ! -d "${GOCHAIN_LOGDIR}" ] ; then
    mkdir -p ${GOCHAIN_LOGDIR}
  fi
  ${GOCHAIN} --config ${GOCHAIN_CONFIG} ${GOCHAIN_OPTIONS} 2>&1 | tee -a ${GOCHAIN_LOGFILE}
else
  ${GOCHAIN} --config ${GOCHAIN_CONFIG} ${GOCHAIN_OPTIONS}
fi
