In this Manual we consider the main SPI use cases as well as give SPI API references for more advanced cases.    

## Table of Contents
- [Use Cases](#use-cases)
    - [Delivering the secrets interactively](#delivering-the-secrets-interactively)
    - [Providing RemoteSecret data in a more secure and interactive way](#providing-remotesecret-data-in-a-more-secure-and-interactive-way)
    - [Creating RemoteSecret and target in a single action](#creating-remotesecret-and-target-in-a-single-action)
    - [Defining the structure of the secrets in the targets](#defining-the-structure-of-the-secrets-in-the-targets)
    - [Defining RemoteSecret with a set of required keys](#defining-RemoteSecret-with-a-set-of-required-keys)
    - [Associating the secret with a service account in the targets](#associating-the-secret-with-a-service-account-in-the-targets)
    - [RemoteSecret has to be created with target namespace and Environment](#RemoteSecret-has-to-be-created-with-target-namespace-and-Environment)
    - [RemoteSecret has to be created all Environments of certain component and application](#RemoteSecret-has-to-be-created-all-Environments-of-certain-component-and-application)
- [Security](#Security)
- [Partial Updates of the Secret Data](#Partial-Updates-of-the-Secret-Data)

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

The targets of the `RemoteSecret` can be defined at any time. If the secret data is not yet associated with the `RemoteSecret`, nothing is delivered to the targets. If there is secret data associated with the secret, it is immediately delivered to the targets, if any are defined.

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

#### Providing RemoteSecret data in a more secure and interactive way

If the `uploadSecret` way of data delivery is not secure or convenient enough, the data for existing or new `RemoteSecret` can be provided in an alternative way. The user creates a `RemoteSecret` with a data field, which is processed by webhook and never enters the etcd.
Also, in case of an error, it will be immediately reported to the caller, without the need to wait for the reconciliation loop and check the object's status. 

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
data:
  username: Z2VuYQ==
  password: Z2VuYQ==
```

#### Copying data from another remote secret securely
If you want to replicate some kind of environment including the secrets used in the "source environment", it might not be a good idea to just download the secrets because that would mean your secrets would leave the controlled environment of the cluster. Another option to copy the secrets without ever revealing their values to any of the users is to take advantage of the remote secret's `dataFrom` capability.

```yaml
apiVersion: appstudio.redhat.com/v1beta1
kind: RemoteSecret
metadata:
    name: my-remote-secret
    namespace: copied-namespace 
spec:
    secret:
        name: pull-secret
        type: kubernetes.io/dockercfg
    targets: []
dataFrom:
  name: my-remote-secret
  namespace: original-namespace
```

This will create a new remote secret called "my-remote-secret" in the "copied-namespace" namespace and will copy the data associated with "my-remote-secret" in the "original-namespace".

Perhaps even better way of creating such an environment copy is to actually copy all the Kubernetes resources, including the remote secrets, from the original "environment", say the "environment" is the "original-namespace" namespace in the example above. Once imported, the remote secrets copied this way will be in the `AwaitingData` state. Then you can patch these remote secrets with either the new data like this:

```yaml
data:
  username: Z2VuYQ==
  password: Z2VuYQ==
```

or copy the data into these new remote secrets from the original location by patching them like this:

```yaml
dataFrom:
  name: my-remote-secret
  namespace: original-namespace
```

The above 2 patches are meant to be saved as a file and used with kubectl like this:

```
kubectl patch remotesecret my-remote-secret -n copied-namespace --type=merge --patch-file=patch.yaml
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

#### Defining the structure of the secrets in the targets

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

#### Defining RemoteSecret with a set of required keys
**Warning:** This feature is in early stages of development. Updating existing RemoteSecret's spec with a new set of
keys might cause unexpected behaviour. If in doubt the safest option is to delete the RemoteSecret and create a new one with the updated spec.

In this example we will create a RemoteSecret which specifies a secret with set of required keys.
This means that secret deployed to targets **must** contain these two keys: `usr` and `pass`.
Note that the secret **may** contain additional keys other than these two, based on the data in the UploadSecret.
```yaml
apiVersion: appstudio.redhat.com/v1beta1
kind: RemoteSecret
metadata:
    name: test-remote-secret
    namespace: default
spec:
    secret:
        name: deployed-secret
        keys: 
        - name: usr
        - name: pass
    targets: []
```

To upload the data for this RemoteSecret we can create the following UploadSecret:
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
    usr: myself
    pass: passwrd
    somekey: somevalue
```

If we failed to provide either the `usr` or `pass` key, the data from UploadSecret would not be saved and an error Event
would be created, such as this one:
```yaml
apiVersion: v1                                                                                                                                                                                                                              
eventTime: null                                                                                                                                                                                                                             
firstTimestamp: null                                                                                                                                                                                                                        
involvedObject:                                                                                                                                                                                                                             
  apiVersion: v1                                                                                                                                                                                                                            
  kind: Secret                                                                                                                                                                                                                              
  name: test-remote-secret-secret                                                                                                                                                                                                           
  namespace: default                                                                                                                                                                                                                        
kind: Event                                                                                                                                                                                                                                 
lastTimestamp: "2023-08-01T13:36:52Z"                                                                                                                                                                                                       
message: 'validation of upload secret failed: the secret data does not contain the                                                                                                                                                          
  required keys: pass '                                                                                                                                                                                                                     
metadata:                                                                                                                                                                                                                                   
  name: test-remote-secret-secret                                                                                                                                                                                                           
  namespace: default                                                                                                                                                                                                                        
reason: cannot process upload secret                                                                                                                                                                                                        
reportingComponent: ""                                                                                                                                                                                                                      
reportingInstance: ""                                                                                                                                                                                                                       
source: {}                                                                                                                                                                                                                                  
type: Error 
```

In the second example below is a RemoteSecret that has a specific type, `kubernetes.io/ssh-auth`, defined in the spec.
Reminder: specifying `Opaque` type is the same as not specifying any type.
```yaml
apiVersion: appstudio.redhat.com/v1beta1
kind: RemoteSecret
metadata:
    name: test-remote-secret
    namespace: default
spec:
    secret:
        type: kubernetes.io/ssh-auth
        name: deployed-secret
        keys: 
        - name: usr
        - name: pass
    targets: []
```

To upload the data for this RemoteSecret, we need to create an UploadSecret which type is `kubernetes.io/ssh-auth`.
Because secrets of type `kubernetes.io/ssh-auth` must have the `ssh-privatekey` key (required by Kubernetes), this means that the UploadSecret
must have at least these three keys: `ssh-privatekey`, `usr`, and `pass`, just like the UploadSecret below:
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
type: kubernetes.io/ssh-auth
stringData:
    usr: myself
    pass: passwrd
    ssh-privatekey: ssh-key...
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

### Security

The remote secret operator places some constraints on the targets where the remote secret can be deployed to. After all, it would be bad if anyone that would be able to modify a remote secret be effectively able to create secrets in any namespace in the cluster.

Therefore, the following limitations are put in place:

. A target can always point to the namespace of the remote secret. This means that anyone that is able to update remote secrets (and therefore modify its targets), is therefore also able to create secrets in the same namespace by using remote secret targets. We feel this is a reasonable thing to do because remote secrets are in a sense just secret "dispatcher" and if, in a certain namespace, you are able to modify former, you should also be able to modify the latter.

. Each target can specify the `clusterCredentialsSecret` - a name of a secret which contains a kubeconfig configuratin for connecting to the target cluster/namespace. This secret needs to live in the same namespace as the remote secret. If you also specify the `apiUrl` on the target of the remote secret, this effectively enables the remote secrets to deploy to a different cluster.

. By default, deploying to a different namespace in the same cluster is disallowed. If you want to enable it, you need to create a service account labeled as `appstudio.redhat.com/remotesecret-auth-sa`. All remote secrets that exist in the namespace that contains such service account will be deployed using that service account to access the target namespaces. This way, one can limit the namespaces to which remote secrets from a certain namespace can be deployed (by only allowing the serviceaccount to access a concrete set of namespaces).

#### Examples

##### Same namespace
Deploying to the same namespace where the remote secret exists is always allowed:

```yaml
apiVersion: appstudio.redhat.com/v1beta1
kind: RemoteSecret
metadata:
    name: test-remote-secret-secret
    namespace: jdoe-workspace
spec:
    secret:
        name: test-remote-secret-secret
    target:
        - namespace: jdoe-workspace
```


##### Same cluster without kubeconfig
Say we are creating our remote secrets in a namespace called `jdoe-workspace`. We want to make sure that users that have access to that workspace can only deploy secrets to the `jdoe-dev` namespace. As the title of this example says, we DON'T want to provide a kubeconfig in each target that would give the remote secret controller permissions to write into the `jdoe-dev` namespace.

For this, we first create a labeled service account in the workspace namespace:

```yaml
apiVersion: v1
kind: ServiceAccount
metadata:
  name: rs-deployer
  namespace: jdoe-workspace
  labels:
    "appstudio.redhat.com/remotesecret-auth-sa": "true"
```

Next, assume that there exists a cluster role that enables CRUD of secrets and service accounts called `secret-setter`. We can use this cluster role to give our above created service account the permission to write into the `jdoe-dev` namespace (of course, there are numerous ways of granting our service account the necessary permissions. This is just one example):

```yaml
apiVersion: rbac.authorization.k8s.io/v1
kind: RoleBinding
metadata:
  name: jdoe-rs-deployer-secret-setter
  namespace: jdoe-dev
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: secret-setter
subjects:
- kind: ServiceAccount
  name: rs-deployer
  namespace: jdoe-workspace
```

With access allowed, we can now create a remote secret that is allowed to deliver from `jdoe-workspace` to `jdoe-dev` but nowhere else (notice the status of the example remote secret where the deployment to a namespace different from `jdoe-dev` fails):

```yaml
apiVersion: appstudio.redhat.com/v1beta1
kind: RemoteSecret
metadata:
    name: test-remote-secret-secret
    namespace: jdoe-workspace
spec:
    secret:
        name: test-remote-secret-secret
    target:
        - namespace: jdoe-dev
        - namespace: other-namespace
status:
  conditions:
  - lastTransitionTime: "..."
    message: ""
    reason: DataFound
    status: "True"
    type: DataObtained
  - lastTransitionTime: "..."
    message: "error while deploying to other-namespace ..."
    reason: PartiallyInjected
    status: "False"
    type: Deployed
  targets:
  - namespace: "jdoe-dev"
    secretName: test-remote-secret-secret 
  - namespace: "other-namespace"
    error: "service account system:serviceaccount:jdoe-workspace:rs-deployer cannot create secrets in namespace other-namespace"
```

##### Another cluster

It is possible to deploy the secrets defined by the remote secret to a competely different cluster using a referenced kubeconfig configuration.

You need to make sure that the kubeconfig configuration either has only a single context or uses the correct context for connecting to the cluster as its current context.

Store this kubeconfig file as a secret in the namespace where your remote secret lives:

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: staging-cluster-kubeconfig
  namespace: jdoe-workspace
data:
  kubeconfig: "... your kubeconfig encoded in base64 ..."
```

Now we can reference this kubeconfig in our remote secret:

```yaml
apiVersion: appstudio.redhat.com/v1beta1
kind: RemoteSecret
metadata:
    name: test-remote-secret-secret
    namespace: jdoe-workspace
spec:
    secret:
        name: test-remote-secret-secret
    target:
    - apiUrl: https://staging-cluster:8443 
      namespace: my-app
      clusterCredentialsSecret: staging-cluster-kubeconfig 
status:
  conditions:
  - lastTransitionTime: "..."
    message: ""
    reason: DataFound
    status: "True"
    type: DataObtained
  - lastTransitionTime: "..."
    reason: Injected
    status: "True"
    type: Deployed
  targets:
  - namespace: "my-app"
    secretName: test-remote-secret-secret 
    clusterCredentialsSecret: staging-cluster-kubeconfig
    apiUrl: https://staging-cluster:8443
```

Note that if you don't specify the `apiUrl` on the target, the current cluster is assumed. Therefore, you can also use kubeconfig-style connections to deploy to the current cluster.

### Partial Updates of the Secret Data

With remote secrets, you can review the set of keys that are present in the secret data (but you cannot retrieve the values which are only ever deployed as secrets in the targets). To be able to amend the keys in a remote secret without knowing the values of all keys in it, one can do a partial update of the data. Using this approach, one can only modify the keys to which the values are known while not touching the pre-existing keys.

The partial update is done using the upload secret as any other secret data manipulation. The upload secret needs to be annotated as providing a partial update and contain only the actual changes to the data as illustrated in the below examples.

Note that the type of the Partial Upload Secret does not need to match the type specified in RemoteSecret's spec
as is the case with Upload Secret (read more in [Delivering the secrets interactively](#delivering-the-secrets-interactively)).
It is therefore easiest to create a Partial Upload Secret with the type `Opaque`. This loosening of a constraint is to avoid
the need to always specify the key-value pairs which are required for the specific type by Kubernetes.
#### Creating new keys in the secret data

In the below example, let's assume there already exists a remote secret called `my-remote-secret` that already has some data uploaded to it. We want to add two new keys to its data, `my-new-key` and `passphrase`, and assign values to them.
```yaml
apiVersion: v1
kind: Secret
metadata:
  name: my-upload-secret
  labels:
    appstudio.redhat.com/upload-secret: remotesecret
  annotations:
    appstudio.redhat.com/remotesecret-name: my-remote-secret
    appstudio.redhat.com/remotesecret-partial-update: "true"
data:
  my-new-key: secret_value
  passphrase: "f0urty 2"
```

#### Updating existing keys in the secret data

Updating existing keys is essentially the same as creating the new ones. You just specify a key value pair where the key already exists in the data of the remote secret. So let's update the value of the `my-new-key` key from the previous example:

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: my-upload-secret
  labels:
    appstudio.redhat.com/upload-secret: remotesecret
  annotations:
    appstudio.redhat.com/remotesecret-name: my-remote-secret
    appstudio.redhat.com/remotesecret-partial-update: "true"
data:
  my-new-key: another_secret_value
```

#### Deleting existing keys from the secret data

Deleting a key is done differently to the update. This is so that you can combine updates and deletions of the keys in one go. So let's say we now want to remove the `my-new-key` because we made a typo, and it should have been `my-secret-key`. Let's also remote the `passphrase` because we don't want it in the secret data of this remote secret anymore.

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: my-upload-secret
  labels:
    appstudio.redhat.com/upload-secret: remotesecret
  annotations:
    appstudio.redhat.com/remotesecret-name: my-remote-secret
    appstudio.redhat.com/remotesecret-partial-update: "true"
    appstudio.redhat.com/remotesecret-deleted-keys: my-new_key, passphrase
data:
  my-secret-key: another_secret_value
```

Note that the deleted keys take precedence over the keys in the data. So if you specify the same key both in the value of the `appstudio.redhat.com/remotesecret-delete-keys` annotation and in the data of the upload secret, the key is deleted from the secret data.
