#!/usr/bin/env bash

mkdir -p .tmp/
SECRET=$(kubectl get -n remotesecret secret vault-approle-remote-secret-operator -o json)

echo "$SECRET" | jq -r ".data.role_id" | base64 -d > .tmp/role_id
echo "$SECRET" | jq -r ".data.secret_id" | base64 -d > .tmp/secret_id

