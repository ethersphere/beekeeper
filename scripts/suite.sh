#!/usr/bin/env bash

set -euo pipefail

#/
#/ Usage: 
#/ ./suite.sh ACTION [OPTION]
#/ 
#/ Description:
#/ Run beekeeper tests on local or remote cluster
#/ 
#/ Example:
#/ ./suite.sh all -n bee -r 3
#/
#/ Actions:

# parse file and print usage text
usage() { grep '^#/' "$0" | cut -c4- ; exit 0 ; }
expr "$*" : ".*-h" > /dev/null && usage
expr "$*" : ".*--help" > /dev/null && usage

declare -x NAMESPACE="bee"
declare -x DOMAIN="localhost"
declare -x REPLICA=3
declare -x ACTION=""
declare -x NAMESPACE_OPTION=""
declare -x BEEKEEPER_BIN="../dist/beekeeper"

_fullconnectivity() {
    "${BEEKEEPER_BIN}" check fullconnectivity --api-scheme http --debug-api-scheme http ${NAMESPACE_OPTION} --debug-api-domain "${DOMAIN}" --api-domain "${DOMAIN}" --node-count "${REPLICA}"
}

_pingpong() {
    "${BEEKEEPER_BIN}" check pingpong --api-scheme http --debug-api-scheme http ${NAMESPACE_OPTION} --debug-api-domain "${DOMAIN}" --api-domain "${DOMAIN}" --node-count "${REPLICA}"
}

_balances() {
    "${BEEKEEPER_BIN}" check balances --api-scheme http --debug-api-scheme http ${NAMESPACE_OPTION} --debug-api-domain "${DOMAIN}" --api-domain "${DOMAIN}" --node-count "${REPLICA}" --upload-node-count "${REPLICA}"
}

_settlements() {
    "${BEEKEEPER_BIN}" check settlements --api-scheme http --debug-api-scheme http ${NAMESPACE_OPTION} --debug-api-domain "${DOMAIN}" --api-domain "${DOMAIN}" --node-count "${REPLICA}" --upload-node-count 10 -t 10000
}

_pushsync() {
    "${BEEKEEPER_BIN}" check pushsync --api-scheme http --debug-api-scheme http ${NAMESPACE_OPTION} --debug-api-domain "${DOMAIN}" --api-domain "${DOMAIN}" --node-count "${REPLICA}" --upload-node-count "${REPLICA}" --chunks-per-node 3
    "${BEEKEEPER_BIN}" check pushsync --api-scheme http --debug-api-scheme http ${NAMESPACE_OPTION} --debug-api-domain "${DOMAIN}" --api-domain "${DOMAIN}" --node-count "${REPLICA}" --upload-node-count "${REPLICA}" --chunks-per-node 3 --upload-chunks
}

_retrieval() {
    "${BEEKEEPER_BIN}" check retrieval --api-scheme http --debug-api-scheme http ${NAMESPACE_OPTION} --debug-api-domain "${DOMAIN}" --api-domain "${DOMAIN}" --node-count "${REPLICA}" --upload-node-count "${REPLICA}" --chunks-per-node 3
}

_pullsync() {
    "${BEEKEEPER_BIN}" check pullsync --api-scheme http --debug-api-scheme http ${NAMESPACE_OPTION} --debug-api-domain "${DOMAIN}" --api-domain "${DOMAIN}" --node-count "${REPLICA}" --upload-node-count "${REPLICA}" --chunks-per-node 3
}

_chunkrepair() {
    "${BEEKEEPER_BIN}" check chunkrepair --api-scheme http --debug-api-scheme http ${NAMESPACE_OPTION} --debug-api-domain "${DOMAIN}" --api-domain "${DOMAIN}" --node-count "${REPLICA}"
}

_manifest() {
    "${BEEKEEPER_BIN}" check manifest --api-scheme http --debug-api-scheme http ${NAMESPACE_OPTION} --debug-api-domain "${DOMAIN}" --api-domain "${DOMAIN}" --node-count "${REPLICA}"
}

_localpinning() {
    "${BEEKEEPER_BIN}" check localpinning --api-scheme http --debug-api-scheme http ${NAMESPACE_OPTION} --debug-api-domain "${DOMAIN}" --api-domain "${DOMAIN}" --node-count "${REPLICA}"
    "${BEEKEEPER_BIN}" check localpinning --api-scheme http --debug-api-scheme http ${NAMESPACE_OPTION} --debug-api-domain "${DOMAIN}" --api-domain "${DOMAIN}" --node-count "${REPLICA}" --large-file-disk-ratio 2
    "${BEEKEEPER_BIN}" check localpinning --api-scheme http --debug-api-scheme http ${NAMESPACE_OPTION} --debug-api-domain "${DOMAIN}" --api-domain "${DOMAIN}" --node-count "${REPLICA}" --large-file-disk-ratio 2 --large-file-count 10
}




if [[ "${BASH_SOURCE[0]}" = "$0" ]]; then

    while [[ $# -gt 0 ]]; do
        key="$1"

        case "${key}" in
#/   all              run all tests except fullconnectivity and localpinning
            all)
                ACTION="all"
                shift
            ;;
#/   localpinning     run localpinning tests; NOTE: clean cluster
            localpinning)
                ACTION="localpinning"
                shift
            ;;
#/   fullconnectivity run fullconnectivity check
            fullconnectivity)
                ACTION="fullconnectivity"
                shift
            ;;
#/   pingpong         run pingpong check
            pingpong)
                ACTION="pingpong"
                shift
            ;;
#/   balances         run balances test
            balances)
                ACTION="balances"
                shift
            ;;
#/   settlements      run settlements test
            settlements)
                ACTION="settlements"
                shift
            ;;
#/   pushsync         run pushsync test
            pushsync)
                ACTION="pushsync"
                shift
            ;;
#/   retrieval        run retrieval test
            retrieval)
                ACTION="retrieval"
                shift
            ;;
#/   pullsync         run pullsync test
            pullsync)
                ACTION="pullsync"
                shift
            ;;
#/   chunkrepair      run chunkrepair test
            chunkrepair)
                ACTION="chunkrepair"
                shift
            ;;
#/   manifest         run manifest test
            manifest)
                ACTION="manifest"
                shift
            ;;
#/
#/ Options:
#/   -n, --namespace name set namespace (default is bee)
            -n|--namespace)
                NAMESPACE="${2}"
                shift 2
            ;;
#/   -d, --domain fqdn    set domain (default is localhost)
            -d|--domain)
                DOMAIN="${2}"
                shift 2
            ;;
#/   -r, --replica n      set number of bee replicas (default is 3)
            -r|--replica)
                REPLICA="${2}"
                shift 2
            ;;
#/   -h, --help           display this help message
            *)
                usage
            ;;
        esac
    done

    if [[ $DOMAIN == "localhost" ]]; then
        NAMESPACE_OPTION="--disable-namespace"
    else
        NAMESPACE_OPTION="--namespace ${NAMESPACE}"
    fi

    if [[ $ACTION == "fullconnectivity" ]]; then
        _fullconnectivity
    fi
    if [[ $ACTION == "pingpong" ]] || [[ $ACTION == "all" ]]; then
        _pingpong
    fi
    if [[ $ACTION == "balances" ]] || [[ $ACTION == "all" ]]; then
        _balances
    fi
    if [[ $ACTION == "settlements" ]] || [[ $ACTION == "all" ]]; then
        _settlements
    fi
    if [[ $ACTION == "pushsync" ]] || [[ $ACTION == "all" ]]; then
        _pushsync
    fi
    if [[ $ACTION == "retrieval" ]] || [[ $ACTION == "all" ]]; then
        _retrieval
    fi
    if [[ $ACTION == "pullsync" ]] || [[ $ACTION == "all" ]]; then
        _pullsync
    fi
    if [[ $ACTION == "chunkrepair" ]] || [[ $ACTION == "all" ]]; then
        _chunkrepair
    fi
    if [[ $ACTION == "manifest" ]] || [[ $ACTION == "all" ]]; then
        _manifest
    fi
    if [[ $ACTION == "localpinning" ]]; then
        _localpinning
    fi
fi