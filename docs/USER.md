In this Manual we consider the main SPI use cases as well as give SPI API references for more advanced cases.    

## Table of Contents
- [Use Cases](#use-cases)
    - [Delivering the secrets interactively](#delivering-the-secrets-interactively)
    - [Creating RemoteSecret and target in a single action](#creating-remotesecret-and-target-in-a-single-action)
    - [Define the structure of the secrets in the targets](#define-the-structure-of-the-secrets-in-the-targets)
    - [Associating the secret with a service account in the targets](#associating-the-secret-with-a-service-account-in-the-targets)
    - [RemoteSecret has to be created with target namespace and Environment](#RemoteSecret-has-to-be-created-with-target-namespace-and-Environment)
    - [RemoteSecret has to be created all Environments of certain component and application](#RemoteSecret-has-to-be-created-all-Environments-of-certain-component-and-application)


### Use Cases
#### Delivering the secrets interactively

At first, the targets to which the secret should be deployed might not yet be known. Nevertheless, the remote secret can be created at this point in time. The creator just doesn't declare any targets.

```yaml
apiVersion: appstudio.redhat.com/v1beta1
kind: RemoteSecret
metadata:
    name: test-remote-secret
    namespace: default
spec:
    secret:
        name: db-credentials
    targets: []
status:
  conditions:
  - lastTransitionTime: "..."
    message: ""
    reason: AwaitingData
    status: "False"
    type: DataObtained
  targets: []
```

After creating the remote secret, the secret data may be associated with it, still without any targets. The user creates an `UploadSecret` that associates the data with the remote secret.

```yaml
apiVersion: v1
kind: Secret
metadata:
    name: upload-secret-data-for-remote-secret
    namespace: default
    labels:
        appstudio.redhat.com/upload-secret: remotesecret
    annotations:
        appstudio.redhat.com/remotesecret-name: test-remote-secret
type: Opaque
stringData:
    username: u
    password: passw0rd
```

After this step, the data is associated with the `RemoteSecret` which is reflected in its status.

```yaml
apiVersion: appstudio.redhat.com/v1beta1
kind: RemoteSecret
metadata:
    name: test-remote-secret
    namespace: default
spec:
    secret:
        name: db-credentials
    targets: []
status:
  conditions:
  - lastTransitionTime: "..."
    message: ""
    reason: DataFound
    status: "True"
    type: DataObtained
  targets: []
```

The targets of the `RemoteSecret` can be defined at any time. If the secret data is not yet associated with the `RemoteSecret`, nothing is delivered to the targets. If there is secret data associated with the secret, it is immediatelly delivered to the targets, if any are defined.

```yaml
apiVersion: appstudio.redhat.com/v1beta1
kind: RemoteSecret
metadata:
    name: test-remote-secret
    namespace: default
spec:
    secret:
        name: db-credentials
    targets:
    - namespace: ns
status:
  conditions:
  - lastTransitionTime: "..."
    message: ""
    reason: DataFound
    status: "True"
    type: DataObtained
  targets:
  - namespace: ns
    secretName: db-credentials
```

**Caution:** When you create a `RemoteSecret` with a specific secret type (`Opaque` is assumed if no type is provided), the `uploadSecret` type has to match it.
If the types do not match the `UploadSecret` will be deleted and the data will not be stored. Instead, a Kubernetes `Event` will be created,
explaining the error, in the same namespace and name as the `UploadSecret`.

Example:`RemoteSecret` with a secret type `kubernetes.io/dockercfg`:
```yaml
apiVersion: appstudio.redhat.com/v1beta1
kind: RemoteSecret
metadata:
    name: test-remote-secret
    namespace: default
spec:
    secret:
        name: pull-secret
        type: kubernetes.io/dockercfg
    targets: []
```

An `uploadSecret` with a matching type:
```yaml
apiVersion: v1
kind: Secret
metadata:
    name: upload-secret-data-for-remote-secret
    namespace: default
    labels:
        appstudio.redhat.com/upload-secret: remotesecret
    annotations:
        appstudio.redhat.com/remotesecret-name: test-remote-secret
type: kubernetes.io/dockercfg
data:
  .dockercfg: |
    "<base64 encoded ~/.dockercfg file>"  
```

`Event` created in case of mismatching types:
```yaml
apiVersion: v1
involvedObject:
  apiVersion: v1
  kind: Secret
  name: test-remote-secret-secret
  namespace: default
kind: Event
lastTimestamp: "..."
message: 'validation of upload secret failed: the type of upload secret and remote
  secret spec do not match, uploadSecret: Opaque, remoteSecret: kubernetes.io/service-account-token '
metadata:
  creationTimestamp: "..."
  name: test-remote-secret-secret
  namespace: default
  resourceVersion: "25579"
  uid: d10e348b-71b9-4479-b581-01b6f21a40f7
reason: cannot process upload secret
type: Error
```

#### Creating RemoteSecret and target in a single action

If a remote secret is supposed to have only one simple target (containing namespace only), it can be created in a single operation by using a special annotation in the upload secret: 

```yaml
apiVersion: v1
kind: Secret
metadata:
    name: upload-secret-data-for-remote-secret
    namespace: default
    labels:
        appstudio.redhat.com/upload-secret: remotesecret
    annotations:
        appstudio.redhat.com/remotesecret-name: test-remote-secret
        appstudio.redhat.com/remotesecret-target-namespace: abcd
type: Opaque
stringData:
    username: u
    password: passw0rd
```
The remote secret will be created and target from `appstudio.redhat.com/remotesecret-target-namespace` annotation will be set:

```yaml
apiVersion: appstudio.redhat.com/v1beta1
kind: RemoteSecret
metadata:
    name: test-remote-secret
    namespace: default
spec:
    secret: {}
    targets:
    - namespace: abcd
status:
  conditions:
  - lastTransitionTime: "..."
    message: ""
    reason: DataFound
    status: "True"
    type: DataObtained
  targets:
    - namespace: abcd
      secretName: test-remote-secret-secret-2nb46
```

#### Define the structure of the secrets in the targets

```yaml
apiVersion: appstudio.redhat.com/v1beta1
kind: RemoteSecret
metadata:
    name: test-remote-secret
    namespace: default
spec:
    secret:
        name: secret-from-remote
    targets: []
status:
  conditions:
  - lastTransitionTime: "..."
    message: ""
    reason: AwaitingData
    status: "False"
    type: DataObtained
  targets: []
```

This example illustrates that we can prescribe the `name` of the secret in the targets. If not specified, as in this case, the type of the secret defaults to `Opaque`.

```yaml
apiVersion: appstudio.redhat.com/v1beta1
kind: RemoteSecret
metadata:
    name: test-remote-secret
    namespace: default
spec:
    secret:
        generateName: secret-from-remote-
    targets:
    - namespace: ns
status:
  conditions:
  - lastTransitionTime: "..."
    message: ""
    reason: DataFound
    status: "True"
    type: DataObtained
  targets:
  - namespace: ns
    secretName: secret-from-remote-sdkfl
```
Here, we merely illustrate that the secret might have a dynamic name when using the `generateName` property. To learn the actual name of the secret when created in the target, the user can inspect the status of the remote secret.

```yaml
apiVersion: appstudio.redhat.com/v1beta1
kind: RemoteSecret
metadata:
    name: test-remote-secret
    namespace: default
spec:
    secret:
        name: secret-from-remote
        type: kubernetes.io/basic-auth
    targets: []
status:
    ...
```

It is also possible to declare the required annotations and labels that the secret should have in the targets:

```yaml
apiVersion: appstudio.redhat.com/v1beta1
kind: RemoteSecret
metadata:
    name: test-remote-secret
    namespace: default
spec:
    secret:
        name: secret-from-remote
        type: kubernetes.io/basic-auth
        labels:
            key: value
        annotations:
            key: value
    targets: []
status:
    ...
```
#### Associating the secret with a service account in the targets
The spec of the `RemoteSecret` can specify that the secret should be linked to a service account in the targets. This is identical to the [feature](https://github.com/redhat-appstudio/service-provider-integration-operator/blob/main/docs/USER.md#providing-secrets-to-a-service-account) present in the `SPIAccessTokenBinding`.

The secret may be linked to a service account that must be already present in the target namespace. When deleting the `RemoteSecret`, such service account is kept in place and only the link to the secret that is being deleted is removed from it.

```yaml
apiVersion: appstudio.redhat.com/v1beta1
kind: RemoteSecret
metadata:
    name: test-remote-secret
    namespace: default
spec:
    secret:
        name: secret-from-remote
        type: kubernetes.io/basic-auth
        linkedTo:
        - serviceAccount:
            reference:
                name: app-sa
    targets: []
status:
    ...
```

It is also possible to create a managed service account. Such service account shares the lifecycle of the `RemoteSecret`.

```yaml
apiVersion: appstudio.redhat.com/v1beta1
kind: RemoteSecret
metadata:
    name: test-remote-secret
    namespace: default
spec:
    secret:
        name: secret-from-remote
        type: kubernetes.io/basic-auth
        linkedTo:
        - serviceAccount:
            managed:
                name: app-sa
    targets: []
status:
    ...
```

It is possible to link the secret to the service account either as an ordinary secret but also as an image pull secret.

```yaml
apiVersion: appstudio.redhat.com/v1beta1
kind: RemoteSecret
metadata:
    name: test-remote-secret
    namespace: default
spec:
    secret:
        name: secret-from-remote
        type: kubernetes.io/basic-auth
        linkedTo:
        - serviceAccount:
            as: imagePullSecret
            managed:
                name: app-sa
    targets: []
status:
    ...
```

#### Inspecting the state of the deployment to targets

```yaml
apiVersion: appstudio.redhat.com/v1beta1
kind: RemoteSecret
metadata:
    name: test-remote-secret
    namespace: default
spec:
    secret:
        generateName: secret-from-remote-
        linkedTo:
            - serviceAccount:
                  managed:
                      generateName: sa-from-remote-
    targets:
        - namespace: "test-target-namespace-1"
        - namespace: "test-target-namespace-2"
        - namespace: "test-target-namespace-3"
        - namespace: "test-target-namespace-rainbow"
          apiUrl: "over-the-rainbow"
          clusterCredentialsSecret: "team-a--prod-dtc--secret"
status:
  conditions:
  - lastTransitionTime: "..."
    message: ""
    reason: DataFound
    status: "True"
    type: DataObtained
  - lastTransitionTime: "..."
    message: "some of the targets were not deployed to"
    reason: PartiallyInjected
    status: "False"
    type: Deployed
  targets:
  - namespace: "test-target-namespace-1"
    secretName: secret-from-remote-lsdjf
    serviceAccountNames:
    - sa-from-remote-llrkt
  - namespace: "test-target-namespace-2"
    secretName: secret-from-remote-lemvs
    serviceAccountNames:
    - sa-from-remote-lkejr
  - namespace: "test-target-namespace-3"
    secretName: secret-from-remote-kjfdl
    serviceAccountNames:
    - sa-from-remote-lmval
  - namespace: "test-target-namepace-rainbow"
    apiUrl: "over-the-rainbow"
    error: "Connection refused"
```
> There are 2 conditions in the status expressing the state of data readiness (`DataObtained` condition type with `AwaitingData` and `DataFound` as possible reasons) and the overall deployment status (`Deployed` condition type with the condition either missing altogether if there are no targets or `PartiallyInjected` or `Injected` reasons).
> Additionally, the status contains the details of the deployment of each of the targets in the spec. The entries might not come in the same order as in the spec but correspond to each entry in the spec by the `namespace` + `apiUrl` compound key (we don't support 2 targets of a single remote secret pointing to the same namespace atm). The status of the target contains the actual names of the secret and the (optional) service accounts (this is important in case of using `generateName` for the secret or the service account(s)) and optionally also an `error` that explains why certain target was not deployed to.

#### RemoteSecret has to be created with target namespace and Environment
```yaml
apiVersion: appstudio.redhat.com/v1beta1
kind: RemoteSecret
metadata:
    name: test-remote-secret-secret
    labels:
        appstudio.redhat.com/environment: prod
        appstudio.redhat.com/component: m-service
        appstudio.redhat.com/application: coffee-shop
spec:
    secret:
        name: test-remote-secret-secret
    target:
        - namespace: jdoe-tenant
status:
  conditions:
  - lastTransitionTime: "..."
    message: ""
    reason: AwaitingData
    status: "False"
    type: DataObtained
```

#### RemoteSecret has to be created all Environments of certain component and application
```yaml
apiVersion: appstudio.redhat.com/v1beta1
kind: RemoteSecret
metadata:
    name: test-remote-secret-secret
    labels:
        appstudio.redhat.com/component: m-service
        appstudio.redhat.com/application: coffee-shop
spec:
    secret:
        name: test-remote-secret-secret
    target:
        - namespace: jdoe-tenant
status:
  conditions:
  - lastTransitionTime: "..."
    message: ""
    reason: AwaitingData
    status: "False"
    type: DataObtained
```