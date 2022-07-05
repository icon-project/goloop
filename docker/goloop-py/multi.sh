#!/bin/bash
set -e

PRE_PWD=$(pwd)
WORKDIR=$(dirname "$(readlink -f ${0})")
cd $WORKDIR

IMAGE_GOLOOP_PY=${IMAGE_GOLOOP_PY:-goloop-py:latest}
GOLOOP_DATA=${GOLOOP_DATA:-/goloop/data}
GOLOOP_DOCKER_REPLICAS=${GOLOOP_DOCKER_REPLICAS:-4}
GOLOOP_DOCKER_NETWORK=${GOLOOP_DOCKER_NETWORK:-goloop_net}
GOLOOP_DOCKER_VOLUME=${GOLOOP_DOCKER_VOLUME:-goloop_data}
GOLOOP_DOCKER_MOUNT=${GOLOOP_DOCKER_MOUNT:-${GOLOOP_DATA}}
GOLOOP_DOCKER_PREFIX=${GOLOOP_DOCKER_PREFIX:-goloop}
GOLOOP_GENESIS_STORAGE=${GOLOOP_DATA}/gs.zip
GOLOOP_DEF_WAIT_TIMEOUT=${GOLOOP_DEF_WAIT_TIMEOUT:-3000}
GOLOOP_RPC_DUMP=${GOLOOP_RPC_DUMP:-false}
GOLOOP_CHANNEL=${GOLOOP_CHANNEL:-test}
GSTOOL=${GSTOOL:-../../bin/gstool}

function create(){
    docker network create --driver overlay --attachable ${GOLOOP_DOCKER_NETWORK} || echo "already created ${GOLOOP_DOCKER_NETWORK}"
    docker volume create ${GOLOOP_DOCKER_VOLUME} || echo "already created ${GOLOOP_DOCKER_VOLUME}"

    for i in $(seq 0 $((${GOLOOP_DOCKER_REPLICAS}-1)));do
        GOLOOP_NODE_DIR="${GOLOOP_DATA}/${i}"
        GOLOOP_CONFIG="${GOLOOP_NODE_DIR}/config.json"
        GOLOOP_KEY_STORE="${GOLOOP_NODE_DIR}/keystore.json"
        GOLOOP_KEY_SECRET="${GOLOOP_NODE_DIR}/secret"
        GOLOOP_LOGFILE="${GOLOOP_NODE_DIR}/goloop.log"

        # keystore
        mkdir -p $(dirname ${GOLOOP_KEY_SECRET})
        echo -n "${GOLOOP_DOCKER_PREFIX}-${i}" > ${GOLOOP_KEY_SECRET}
        echo "${GSTOOL} ks gen -o ${GOLOOP_KEY_STORE} -p \$(cat ${GOLOOP_KEY_SECRET})"
        ${GSTOOL} ks gen -o "${GOLOOP_KEY_STORE}" -p $(cat ${GOLOOP_KEY_SECRET})

        docker run -d \
          --mount type=volume,src=${GOLOOP_DOCKER_VOLUME},dst=${GOLOOP_DOCKER_MOUNT} \
          --network ${GOLOOP_DOCKER_NETWORK} \
          --network-alias ${GOLOOP_DOCKER_PREFIX}-${i} \
          --name ${GOLOOP_DOCKER_PREFIX}-${i} \
          --hostname ${GOLOOP_DOCKER_PREFIX}-${i} \
          --env TASK_SLOT=${i} \
          --env GOLOOP_NODE_DIR=${GOLOOP_NODE_DIR} \
          --env GOLOOP_CONFIG=${GOLOOP_CONFIG} \
          --env GOLOOP_KEY_STORE=${GOLOOP_KEY_STORE} \
          --env GOLOOP_KEY_SECRET=${GOLOOP_KEY_SECRET} \
          --env GOLOOP_LOGFILE=${GOLOOP_LOGFILE} \
          --env GOLOOP_P2P=${GOLOOP_DOCKER_PREFIX}-${i}:8080 \
          --env GOLOOP_RPC_DUMP=${GOLOOP_RPC_DUMP} \
          ${IMAGE_GOLOOP_PY}

        set +e
        MAX_RETRY=10
        echo -n "waiting for start server "
        for j in $(seq 1 $MAX_RETRY);do
          RESULT=$(docker exec ${GOLOOP_DOCKER_PREFIX}-${i} goloop system info 2>&1)
          if [ "$?" == "0" ];then
            echo "ok"
            break
          fi
          echo -n "."
          sleep 0.5
        done
        echo $RESULT
        docker exec ${GOLOOP_DOCKER_PREFIX}-${i} goloop system config rpcIncludeDebug true
        set -e
    done
}

function join(){
    local GENESIS_TEMPLATE=${1:-${GOLOOP_DATA}/genesis/genesis.json}
    local GOD_KEYSTORE=${2}

    # collect node addresses
    for i in $(seq 0 $((${GOLOOP_DOCKER_REPLICAS}-1)));do
        ADDRESS=$(docker exec ${GOLOOP_DOCKER_PREFIX}-${i} goloop system info --format "{{.Setting.Address}}")
        VALIDATORS="${VALIDATORS} -v ${ADDRESS}"
        ADDRESSES="${ADDRESSES} ${ADDRESS}"
    done

    # god keystore
    if [ "${GOD_KEYSTORE}" != "" ] && [ ! -f ${GOD_KEYSTORE} ]; then
      mkdir -p $(dirname ${GOD_KEYSTORE})
      ${GSTOOL} ks gen -o ${GOD_KEYSTORE}
    fi
    GSTOOL_CMD="${GSTOOL} gn --god ${GOD_KEYSTORE}"
    # genesis
    if [ ! -f ${GENESIS_TEMPLATE} ];then
        mkdir -p $(dirname ${GENESIS_TEMPLATE})
        GSTOOL_CMD="${GSTOOL_CMD} -o ${GENESIS_TEMPLATE} gen ${ADDRESSES}"
    else
        GSTOOL_CMD="${GSTOOL_CMD} ${VALIDATORS} edit ${GENESIS_TEMPLATE}"
    fi
    echo ${GSTOOL_CMD}
    ${GSTOOL_CMD}

    echo ${GSTOOL} gs gen -i ${GENESIS_TEMPLATE} -o ${GOLOOP_GENESIS_STORAGE}
    ${GSTOOL} gs gen -i ${GENESIS_TEMPLATE} -o ${GOLOOP_GENESIS_STORAGE}

    for i in $(seq 0 $((${GOLOOP_DOCKER_REPLICAS}-1)));do
        docker exec ${GOLOOP_DOCKER_PREFIX}-${i} goloop chain join --genesis ${GOLOOP_GENESIS_STORAGE} --seed "${GOLOOP_DOCKER_PREFIX}-0":8080 --channel ${GOLOOP_CHANNEL} --default_wait_timeout ${GOLOOP_DEF_WAIT_TIMEOUT}
    done
}

function start(){
    local GENESIS_CID=$(${GSTOOL} gs info -c ${GOLOOP_GENESIS_STORAGE})

    for i in $(seq 0 $((${GOLOOP_DOCKER_REPLICAS}-1)));do
        docker exec ${GOLOOP_DOCKER_PREFIX}-${i} goloop chain start ${GENESIS_CID}
    done
}

function env(){
    local GENESIS_NID=$(${GSTOOL} gs info -n ${GOLOOP_GENESIS_STORAGE})
    local ENVFILE=${1}
    cp ${ENVFILE} ${ENVFILE}.backup
    grep "^chain" ${ENVFILE}.backup | sed -e "s/chain0.nid=.*/chain0.nid=${GENESIS_NID}/" > ${ENVFILE}
    for i in $(seq 0 $((${GOLOOP_DOCKER_REPLICAS}-1)));do
        NODE_PREFIX="node${i}"
        echo -e "${NODE_PREFIX}.url=http://${GOLOOP_DOCKER_PREFIX}-${i}:9080" >> ${ENVFILE}
        echo -e "${NODE_PREFIX}.channel0.nid=${GENESIS_NID}" >> ${ENVFILE}
        echo -e "${NODE_PREFIX}.channel0.name=${GOLOOP_CHANNEL}" >> ${ENVFILE}
        GOLOOP_NODE_DIR="${GOLOOP_DATA}/${i}"
        GOLOOP_KEY_STORE="${GOLOOP_NODE_DIR}/keystore.json"
        GOLOOP_KEY_SECRET="${GOLOOP_NODE_DIR}/secret"
        echo -e "${NODE_PREFIX}.wallet=${GOLOOP_KEY_STORE}" >> ${ENVFILE}
        echo -e "${NODE_PREFIX}.walletPassword=$(cat ${GOLOOP_KEY_SECRET})" >> ${ENVFILE}
    done
}

function rm(){
    for i in $(seq 0 $((${GOLOOP_DOCKER_REPLICAS}-1)));do
        echo "docker stop $(docker stop ${GOLOOP_DOCKER_PREFIX}-${i})"
        echo "docker rm $(docker rm ${GOLOOP_DOCKER_PREFIX}-${i})"
    done
    echo "docker network rm $(docker network rm ${GOLOOP_DOCKER_NETWORK})"
    echo "docker volume rm $(docker volume rm ${GOLOOP_DOCKER_VOLUME})"
}

case $1 in
create)
  create
;;
join)
  join $2 $3
;;
start)
  start
;;
env)
  env $2
;;
rm)
  rm
;;
*)
  echo "Usage: $0 [create,join,start,env,rm]"
  echo "  create: $0 create"
  echo "  join: $0 join [GENESIS_TEMPLATE] [GOD_KEYSTORE]"
  echo "  start: $0 start"
  echo "  env: $0 env [ENV_PROPERTIES]"
  echo "  rm: $0 rm"
  cd $PRE_PWD
  exit 1
;;
esac

cd $PRE_PWD
