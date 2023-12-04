#!/bin/sh

# Use this script when you want to setup connection for a locally running webhook, from a minikube cluster.

# This script creates certificates in /tmp/k8s-webhook-server/serving-certs/ required for webhook and afterward
# creates/patches the MutatingWebhookConfiguration so that requests are routed out of minikube into the localhost where
# controller (with webhook) will be running.

# Note that if you change your mind and want to run the controller in cluster, you will have to create/restore the original
# MutatingWebhookConfiguration. The easiest way is to run `make deploy_minikube`

set -e

THIS_DIR="$(dirname "$(realpath "$0")")"

export GENCERTS_DIR="/tmp/k8s-webhook-server/serving-certs/"
export CSR_FILE="${THIS_DIR}/minikube_webhook_csr.conf"

CA_BUNDLE=$("${THIS_DIR}/generate_webhook_ca.sh")
export CA_BUNDLE

yq eval '
  .webhooks[0].clientConfig.url = "https://host.minikube.internal:9443/mutate-appstudio-redhat-com-v1beta1-remotesecret" |
  .webhooks[0].clientConfig.service = null |
  .webhooks[0].clientConfig.caBundle = strenv(CA_BUNDLE)
' "${THIS_DIR}/../config/webhook/base/manifests.yaml" \
| kubectl apply -f -
