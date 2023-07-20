#!/usr/bin/env bash

# This script assumes 3 or 4 params:
# 1. the kubeconfig context where the created kubeconfig should be pointing to
# 2. the name of the secret
# 3. the namespace for the secret
# 4. optionally, the name of the service account in the namespace of the context specified in the first parameter to use the token of. If not specified, OpenShift is assumed and the token of the current user is used.

THIS_DIR="$(dirname "$(realpath "$0")")"

KUBECONFIG=${KUBECONFIG:-${HOME}/.kube/config}
if [ -z "$4" ]; then
    TOKEN=$(oc --kubeconfig="${KUBECONFIG}" --context="$1" whoami -t)
else
    TOKEN=$(kubectl --kubeconfig="${KUBECONFIG}" --context="$1" create token "$4")
fi

CLUSTER_NAME=$(yq ".contexts[] | select(.name == \"$1\") | .context.cluster" < "${KUBECONFIG}")

kubectl apply -f - <<EOF
apiVersion: v1
kind: Secret
metadata:
  name: $2
  namespace: $3
data:
  kubeconfig: $(echo "${TOKEN}" | "${THIS_DIR}"/create-kubeconfig.sh "${CLUSTER_NAME}" | base64 -w0)
EOF

