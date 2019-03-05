#!/bin/bash
#
#   GOCHAIN autostart script
#
#   Usable environment variables.
#
#   GOCAHIN_DATA    (deault:"./data")
#                   Path to to store BlockData, Contracts and WAL.
#
#   GOCHAIN_CONFIG  (default:"./config.json")
#                   Path to configuration file. If it doesn't exist, it will
#                   automatically generate one.
#
#   GOCHAIN_GENESIS (optional)
#                   Path to the genesis transaction file. It will overrides
#                   configuration file.
#
#   GOCHAIN_GENESIS_DATA (optional)
#                   Path to the genesis data. If there is extra genesis data,
#                   then set to specified directory or file.
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
#   GOCHAIN_KEYAUTO (recommeded)
#                   Set it as "1" to enable automatic generation of KeySecret
#                   through user input.
#


GOCHAIN="./gochain"

GOCHAIN_DATA=${GOCHAIN_DATA:-"./data"}
GOCHAIN_CONFIG=${GOCHAIN_CONFIG:-"./config.json"}
GOCHAIN_KEYSTORE=${GOCHAIN_KEYSTORE:-"./keystore.json"}

if [ ! -d ${GOCHAIN_DATA} ] ; then
    mkdir -p ${GOCHAIN_DATA} || exit -1
fi

GOCHAIN_OPTIONS=(-node_dir "${GOCHAIN_DATA}")
GOCHAIN_OPTIONS+=(-ee_socket "/tmp/socket")
GOCHAIN_OPTIONS+=(-db_type goleveldb)
GOCHAIN_OPTIONS+=(-role 3)

if [ "${GOCHAIN_GENESIS}" != "" ] ; then
    GOCHAIN_OPTIONS+=(-genesis "${GOCHAIN_GENESIS}")
fi

if [ "${GOCHAIN_GENESIS_DATA}" != "" ] ; then
    GOCHAIN_OPTIONS+=(-genesis_data "${GOCHAIN_GENESIS_DATA}")
fi

if [ -r "${GOCHAIN_KEYSTORE}" ] ; then
    GOCHAIN_OPTIONS+=(-key_store "${GOCHAIN_KEYSTORE}")
else
    GOCHAIN_OPTIONS+=(-save_key_store "${GOCHAIN_KEYSTORE}")
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
    GOCHAIN_OPTIONS+=(-key_secret "${GOCHAIN_KEYSECRET}")
fi

if [ "${GOCHAIN_ADDRESS}" != "" ] ; then
    GOCHAIN_OPTIONS+=(-p2p ${GOCHAIN_ADDRESS})
else
    GOCHAIN_OPTIONS+=(-p2p "${HOST}:8080")
fi
GOCHAIN_OPTIONS+=(-p2p_listen ":8080")

if [ -n -r "${GOCHAIN_CONFIG}" ] ; then
    echo "[!] Generate config path=[${GOCHAIN_CONFIG}]"
    ${GOCHAIN} "${GOCHAIN_OPTIONS[@]}" \
        -save ${GOCHAIN_CONFIG} || exit -1
fi

${GOCHAIN} -config ${GOCHAIN_CONFIG} "${GOCHAIN_OPTIONS[@]}"
