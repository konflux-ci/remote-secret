#!/bin/sh

# This script creates a number of remote secrets with data and then starts the controller using `make run` with profiling enabled
# and collects heap profiles each second for a number of times. This can be used to estimate the memory consumption during the startup
# of the controller.

NOF_RESOURCES=500
PROFILING_TIME=60

THIS_DIR="$(dirname "$(realpath "$0")")"

echo "creating $NOF_RESOURCES remote secrets with data"

for i in $(seq 1 $NOF_RESOURCES); do
	kubectl apply -f - <<EOF
apiVersion: appstudio.redhat.com/v1beta1
kind: RemoteSecret
metadata:
  name: perf-rs-$i
  namespace: default
  labels:
    perf-test: "true"
spec:
  secret: {}
  targets:
  - namespace: default
EOF

	kubectl apply -f - <<EOF
apiVersion: v1
kind: Secret
metadata:
  name: upload-$i
  namespace: default
  labels:
    appstudio.redhat.com/upload-secret: remotesecret
  annotations:
    appstudio.redhat.com/remotesecret-name: perf-rs-$i
stringData:
  k1: v1
  k2: v2
EOF

done

echo "starting the remote secret controller"

sh -c 'echo $$ > pid.file; exec make -C '"$THIS_DIR"'/.. run EXPOSEPROFILING=true' &
until [ -f pid.file ]; do
	sleep 1
done
while [ -z $PID ]; do
	PID=$(cat pid.file)
done
rm pid.file

echo "collecting the heap info for $PROFILING_TIME consecutive seconds"

for i in $(seq 1 $PROFILING_TIME); do
	curl --retry 5 --retry-connrefused --retry-delay 1 -o heap-$i.out "http://localhost:8080/debug/pprof/heap"
	sleep 1
done

sleep 10

echo "deleting all the created remote secrets"

kubectl delete remotesecret -n default -l perf-test=true

curl -o heap-after.out "http://localhost:8080/debug/pprof/heap"

echo "killing the remote secret controller with PID $PID"
kill $PID
