#!/usr/bin/env bash

# This script assumes 4 params:
# 1. the kubeconfig context where the created kubeconfig should be pointing to
# 2. the name of the cluster where the secret should be created (one of the clusters in the kube config)
# 3. the name of the secret
# 4. the namespace for the secret

# NOTE that this script only works if the target context (i.e. the first parameter) is pointing to an OpenShift cluster.

THIS_DIR="$(dirname "$(realpath "$0")")"

KUBECONFIG=${KUBECONFIG:-${HOME}/.kube/config}
TOKEN=$(oc --kubeconfig="${KUBECONFIG}" --context="$1" whoami -t)

kubectl apply -f - <<EOF
apiVersion: v1
kind: Secret
metadata:
  name: $3
  namespace: $4
data:
  kubeconfig: $(echo "${TOKEN}" | "${THIS_DIR}"/create-kubeconfig.sh "$2" | base64 -w0)
EOF

