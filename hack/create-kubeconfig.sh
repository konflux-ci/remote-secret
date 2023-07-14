#!/usr/bin/env bash

# This script creates a kubeconfig for connecting to a cluster that is already present in the KUBECONFIG but will use
# a provided bearer token instead of the auth method specified in the KUBECONFIG.

# This expects the name of the cluster as a parameter and the bearer token on the stdin.

if [ $# -ne 1 ]; then
	echo "this script expects the name of the cluster as a sole parameter and the contents of the bearer token on standard input. The kubeconfig is printed to the standard output" 1>&2
	exit 1
fi

KUBECONFIG=${KUBECONFIG:-${HOME}/.kube/config}

CLUSTER_NAME=$1

read -r TOKEN

SERVER=$(yq ".clusters[] | select(.name == \"${CLUSTER_NAME}\") | .cluster.server" < "${KUBECONFIG}")
CA=$(yq ".clusters[] | select(.name == \"${CLUSTER_NAME}\") | .cluster.certificate-authority" < "${KUBECONFIG}")
INSECURE=$(yq ".clusters[] | select(.name == \"${CLUSTER_NAME}\") | .cluster.insecure-skip-tls-verify" < "${KUBECONFIG}")

# now construct a new kubeconfig on the standard output
echo "apiVersion: v1"
echo "kind: Config"
echo "current-context: ctx"
echo "clusters:"
echo "- name: cluster"
echo "  cluster:"
if [ "${INSECURE}" != "null" ]; then
echo "    insecure-skip-tls-verify: ${INSECURE}"
fi
if [ "${CA}" != "null" ]; then
echo "    certificate-authority-data: |+"
sed 's/^/      /'   < "${CA}"
fi
echo "    server: ${SERVER}"
echo "users:"
echo "- name: user"
echo "  user:"
echo "    token: \"${TOKEN}\""
echo "contexts:"
echo "- name: ctx"
echo "  context:"
echo "    cluster: cluster"
echo "    user: user"
echo "    namespace: default"
