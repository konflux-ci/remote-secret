## Building & Testing
This project provides a `Makefile` to run all the usual development tasks. If you simply run `make` without any arguments, you'll get a list of available "targets".

To build the project one needs to invoke:

```
make build
```

To test the code (WARNING: tests require a running cluster in the kubectl context):

```
make test
```
To run individual unit tests, you can use the normal `go test` workflow.

There are also many integration tests that are also run by `make test`.

The integration tests run with a `testenv` Kubernetes API server so they cannot be run simply by `go test`. You can run individual integration tests using 
```
make itest focus="..."
```
where the value of `focus` is the description of the Ginkgo integration test you want to run.

To build the docker images of the operator and oauth service one can run:

```
make docker-build
```

This will make a docker images called `quay.io/redhat-appstudio/remote-secret-controller:latest` which might or might not be what you want.
To override the name of the image build, specify it in the `IMG_BASE` and/or `TAG_NAME` environment variable, e.g.:

```
make docker-build IMG_BASE=quay.io/acme TAG_NAME=bugfix
```

To push the images to an image repository one can use:

```
make docker-push
```

The image being pushed can again be modified using the environment variable:
```
make docker-push IMG_BASE=quay.io/acme TAG_NAME=bugfix
```

To set precise image names, one can use `IMG` for operator image (see [Makefile](Makefile) for more details).

Before you push a PR to the repository, it is recommended to run an overall validity check of the codebase. This will
run the formatting check, static code analysis and all the tests:

```
make check
```
If you don't want to merely check that everything is OK but also make the modifications automatically, if necessary, you can, instead of `make check`, run:

```
make ready
```

which will automatically format and lint the code, update the `go.mod` and `go.sum` files and run tests. As such, this goal may modify the contents of the repository.

### Running out of cluster
There is a dedicated make target to run the operator locally:

```
make run
```

This will also deploy RBAC setup and the CRDs into the cluster and will run the operator locally with the permissions of the deployed service account as configure in the Kustomize files in the `config` directory.

To run the operator with the permissions of the currently active kubectl context, use:

```
make run_as_current_user
```
To run the OAuth service locally, one can use:

```
make run_oauth
```

### Running in cluster
Again, there is a dedicated make target to deploy the operator with OAuth service into the cluster:
```
make deploy_openshift       # OpenShift with Vault tokenstorage
make deploy_openshift_aws   # OpenShift with AWS tokenstorage
make deploy_minikube        # minikube with Vault tokenstorage
make deploy_minikube_aws    # minikube with AWS tokenstorage
```

## Debugging

It is possible to debug the operator using `dlv` or some IDE like `vscode`. Just point the debugger of your choice to `main.go` as the main program file and remember to configure the environment variables for the correct/intended function of the operator.

## Manual testing with custom images

This assumes the current working directory is your local checkout of this repository.

Then we can install our CRDs:

```
make install
```

Next, we're ready to build and push the custom operator and oauth images:
```
make docker-build docker-push IMG_BASE=<MY-CUSTOM-IMAGE-BASE> TAG_NAME=<MY-CUSTOM-TAG-NAME>
```

Next step is to deploy the operator along with all other Kubernetes objects to the cluster.


On OpenShift use:
```
make deploy_openshift IMG_BASE=<MY-CUSTOM-IMAGE-BASE> TAG_NAME=<MY-CUSTOM-TAG-NAME>
```

On Minikube use:
```
make deploy_minikube IMG_BASE=<MY-CUSTOM-IMAGE-BASE> TAG_NAME=<MY-CUSTOM-TAG-NAME>
```
