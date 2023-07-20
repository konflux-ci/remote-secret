#!/bin/sh

# This script copies the config directory containing the kustomize templates into a subdirectory under .tmp.
# It then replaces the placeholders in those templates using envsubst, sets the images according to the SPIO_IMG
# and SPIS_IMG env vars (as defined by the Makefile) and deploys.

set -e

# the path to the kustomize executable
KUSTOMIZE=$1

# the name of the deployment - only used as a part of the directory the templates are copied into
DEPL_NAME=$2

# The name of the kustomize overlay directory to apply
OVERLAY=$3

THIS_DIR="$(dirname "$(realpath "$0")")"
TEMP_DIR="${THIS_DIR}/../.tmp/deployment_${DEPL_NAME}"

OVERLAY_DIR="${TEMP_DIR}/${OVERLAY}"
GENCERTS_DIR="${TEMP_DIR}/webhook/certs"

# we need this to keep kustomize patches intact
export patch="\$patch"

function gen_certs() {
  echo "generating certificates ..."
  openssl genrsa -out ${GENCERTS_DIR}/ca.key 2048
  openssl req -x509 -new -nodes -key ${GENCERTS_DIR}/ca.key -subj "/CN=remote-secret-webhook-service.remotesecret.svc" -days 10000 -out ${GENCERTS_DIR}/ca.crt
  openssl genrsa -out ${GENCERTS_DIR}/tls.key 2048
  openssl req -new -key ${GENCERTS_DIR}/tls.key -out ${GENCERTS_DIR}/tls.csr -config ${THIS_DIR}/csr.conf
  openssl x509 -req -in ${GENCERTS_DIR}/tls.csr -CA ${GENCERTS_DIR}/ca.crt -CAkey ${GENCERTS_DIR}/ca.key -CAcreateserial -out ${GENCERTS_DIR}/tls.crt -days 10000 -extensions v3_ext -extfile ${THIS_DIR}/csr.conf -sha256
}

mkdir -p "${TEMP_DIR}"
mkdir -p "${GENCERTS_DIR}"
cp -r "${THIS_DIR}/../config/"* "${TEMP_DIR}"
gen_certs
export CA_BUNDLE=$(cat ${GENCERTS_DIR}/ca.crt | base64 | tr -d '\n')
find "${TEMP_DIR}" -name '*.yaml' | while read -r f; do
  tmp=$(mktemp)
  envsubst > "$tmp" < "$f"
  mv "$tmp" "$f"
done

CURDIR=$(pwd)
cd "${OVERLAY_DIR}" || exit
if [ ! -z ${IMG} ]; then
  ${KUSTOMIZE} edit set image quay.io/redhat-appstudio/remote-secret-controller="${IMG}"
fi

${KUSTOMIZE} build . | kubectl apply -f -
cd "${CURDIR}" || exit
