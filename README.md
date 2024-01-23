# remote-secret
[![Code Coverage Report](https://github.com/redhat-appstudio/remote-secret/actions/workflows/codecov.yaml/badge.svg)](https://github.com/redhat-appstudio/remote-secret/actions/workflows/codecov.yaml)
[![codecov](https://codecov.io/gh/redhat-appstudio/remote-secret/branch/main/graph/badge.svg?token=wwN9l7BE12)](https://codecov.io/gh/redhat-appstudio/remote-secret)

A Kubernetes controller/operator that manages secrets on multiple targets.
## Table of contents

- [About](#About)
- [Terminology](#terminology)
- [Architecture](#architecture)
- [Administration Guide](docs/ADMIN.md)
    - [Installation](docs/ADMIN.md#installation)
    - [Configuration parameters](docs/ADMIN.md#Configuration-parameters)
    - [Vault](docs/ADMIN.md#vault)
- [User Guide](docs/USER.md)
- [Contributing](docs/DEVELOP.md)
- [License](LICENSE)
- 

### About
TODO

### Terminology

- **Upload Secret**: A short-lived Kubernetes `Secret` used to deliver confidential data to permanent storage and link it to the `RemoteSecret` CR. The Upload Secret is not a CRD.
- `SecretData`: An object stored in permanent SecretStorage. Valid SecretData is always linked to a RemoteSecret CR.
- `RemoteSecret`: A CRD that appears during upload and links `SecretData` + `DeploymentTarget(s)` + K8s `Secret`. `RemoteSecret` is linked to one (or zero) SecretData and manages its deleting/updating.
- K8s `Secret`: What appears at the output and is used by consumers.
- `SecretId`: A unique identifier of SecretData in permanent SecretStorage.
- `SecretStorage`: A database eligible for storing `SecretData` (such as HashiCorp Vault, AWS Secret Manager). That is an internal mechanism. Only spi-operator will be able to access it directly.

### Architecture


The proposed solution is to create a new Kubernetes Custom Resource (CR) called `RemoteSecret`. It serves as a representation of the Kubernetes Secret that is stored in permanent storage, which is also referred to as `SecretStorage`. This Custom Resource includes references to targets, like Kubernetes namespaces, that may also contain the required data to connect to a remote Kubernetes. To perform an upload to permanent storage, a temporary **Upload Secret** is utilized, which is represented as a regular Kubernetes Secret with special labels and annotations that the SPI controller recognizes. Different `SecretStorage` implementations, like AWS Secret Manager or HashiCorp Vault, can be used. It is simpler to create the RemoteSecret first and then use the linked **Upload Secret** to upload secret data. However, in simple cases, the **Upload Secret**  can be used to perform both uploading and the creation of RemoteSecret in a single action. It's worth noting that the **Upload Secret** is not a core component of the framework, but rather a convenient way of creating secrets. In the future, it is possible that new methods of uploading SecretData to RemoteSecret may be added.

The design specifically allows for separating the upload of the secret data (using the `UploadSecret`) and delivering the secret to the target. The list of targets of the `RemoteSecret` can be initially empty and can be updated as time evolves.

Apart from the remote secret supporting the delivery of the secret, it is also optionally able to link this secret to the service accounts in the target. These service accounts are either managed by the remote secret (i.e. share its lifecycle) or are required to pre-exist in the target cluster and namespace.

The remote secret is a template of a single secret that can be delivered to multiple targets. Because the target is identified by the namespace (and the URL of the cluster, if provided), there can be at most one secret delivered to a certain namespace by a single remote secret.

