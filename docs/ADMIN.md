## Installation
TODO Explain production configuration


#### Configuration parameters
The `remote-secret-controller-manager-environment-config` config map contains configuration options that will be applied to  the operator.

| Command argument                                      | Environment variable           | Default                  | Description                                                                                                                                                                                                                        |
|-------------------------------------------------------|--------------------------------|--------------------------|------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| --base-url                                            | BASEURL                        |                          | This is the publicly accessible URL on which the SPI OAuth service is reachable. Note that this is not just a hostname, it is a full URL including a scheme, e.g. "https://acme.com/spi"                                           |
| --config-file                                         | CONFIGFILE                     | /etc/spi/config.yaml     | The location of the configuration file.                                                                                                                                                                                            |
| --instance-id                                         | INSTANCEID                     | spi-1                    | ID of this SPI instance. Used to avoid conflicts when multiple SPI instances uses shared resources (e.g. secretstorage).                                                                                                           |
| --metrics-bind-address                                | METRICSADDR                    | 127.0.0.1:8080           | The address the metric endpoint binds to. Note: While this is the default from the operator binary point of view, the metrics are still available externally through the authorized endpoint provided by kube-rbac-proxy           |
| --pprof-bind-address                                  | PPROFBINDADDRESS               | 0                        | Is the TCP address that the controller should bind to for serving pprof.                                                                                                                                                           |
| --allow-insecure-urls                                 | ALLOWINSECUREURLS              | false                    | Whether it is allowed or not to use insecure (http) URLs in service provider or token storage configurations.                                                                                                                      |
| --health-probe-bind-address HEALTH-PROBE-BIND-ADDRESS | PROBEADDR                      | :8081                    | The address the probe endpoint binds to.                                                                                                                                                                                           |
| --tokenstorage                                        | TOKENSTORAGE                   | vault                    | The type of the token storage. Supported types: 'vault', 'aws', 'memory', 'es'                                                                                                                                                     |
| --vault-host                                          | VAULTHOST                      | http://spi-vault:8200    | Vault host URL. Default is internal kubernetes service.                                                                                                                                                                            |
| --vault-insecure-tls                                  | VAULTINSECURETLS               | false                    | Whether is allowed or not insecure vault tls connection.                                                                                                                                                                           |
| --vault-auth-method                                   | VAULTAUTHMETHOD                | approle                  | Authentication method to Vault token storage. Options: 'kubernetes', 'approle'.                                                                                                                                                    |
| --vault-roleid-filepath                               | VAULTAPPROLEROLEIDFILEPATH     | /etc/spi/role_id         | Used with Vault approle authentication. Filepath with role_id.                                                                                                                                                                     |
| --vault-secretid-filepath                             | VAULTAPPROLESECRETIDFILEPATH   | /etc/spi/secret_id       | Used with Vault approle authentication. Filepath with secret_id.                                                                                                                                                                   |
| --vault-k8s-sa-token-filepath                         | VAULTKUBERNETESSATOKENFILEPATH |                          | Used with Vault kubernetes authentication. Filepath to kubernetes ServiceAccount token. When empty, Vault configuration uses default k8s path. No need to set when running in k8s deployment, useful mostly for local development. |
| --vault-k8s-role                                      | VAULTKUBERNETESROLE            |                          | Used with Vault kubernetes authentication. Vault authentication role set for k8s ServiceAccount.                                                                                                                                   |
| --vault-data-path-prefix                              | VAULTDATAPATHPREFIX            | spi                      | Path prefix in Vault token storage under which all SPI data will be stored. No leading or trailing '/' should be used, it will be trimmed.                                                                                         |
| --aws-config-filepath                                 | AWS_CONFIG_FILE                | /etc/spi/aws/config      | Filepath to AWS configuration file                                                                                                                                                                                                 |
| --aws-credentials-filepath                            | AWS_CREDENTIALS_FILE           | /etc/spi/aws/credentials | Filepath to AWS credentials file                                                                                                                                                                                                   |
| --zap-devel                                           | ZAPDEVEL                       | false                    | Development Mode defaults(encoder=consoleEncoder,logLevel=Debug,stackTraceLevel=Warn) Production Mode defaults(encoder=jsonEncoder,logLevel=Info,stackTraceLevel=Error)                                                            |
| --zap-encoder                                         | ZAPENCODER                     |                          | Zap log encoding (‘json’ or ‘console’)                                                                                                                                                                                             |
| --zap-log-level                                       | ZAPLOGLEVEL                    |                          | Zap Level to configure the verbosity of logging.                                                                                                                                                                                   |
| --zap-stacktrace-level                                | ZAPSTACKTRACELEVEL             |                          | Zap Level at and above which stacktraces are captured.                                                                                                                                                                             |
| --zap-time-encoding                                   | ZAPTIMEENCODING                | iso8601                  | Format of the time in the log. One of 'epoch', 'millis', 'nano', 'iso8601', 'rfc3339' or 'rfc3339nano.                                                                                                                             |
| --leader-elect                                        | ENABLELEADERELECTION           | false                    | Enable leader election for controller manager. Enabling this will ensure there is only one active controller manager.                                                                                                              |
| --metadata-cache-ttl                                  | TOKENMETADATACACHETTL          | 1h                       | The maximum age of the token metadata cache. To reduce the load on the service providers, SPI only refreshes the metadata of the tokens when determined stale by this parameter.                                                   |
| --token-ttl                                           | TOKENLIFETIMEDURATION          | 120h                     | Access token lifetime in hours, minutes or seconds. Examples:  "3h",  "5h30m40s" etc.                                                                                                                                              |
| --binding-ttl                                         | BINDINGLIFETIMEDURATION        | 2h                       | Access token binding lifetime in hours, minutes or seconds. Examples: "3h", "5h30m40s" etc.                                                                                                                                        |
| --access-check-ttl                                    | ACCESSCHECKLIFETIMEDURATION    | 30m                      | Access check lifetime in hours, minutes or seconds.                                                                                                                                                                                |
| --file-request-ttl                                    | FILEREQUESTLIFETIMEDURATION    | 30m                      | File content request lifetime in hours, minutes or seconds.                                                                                                                                                                        |
| --token-match-policy                                  | TOKENMATCHPOLICY               | any                      | The policy to match the token against the binding. Options:  'any', 'exact'."`                                                                                                                                                     |
| --deletion-grace-period                               | DELETIONGRACEPERIOD            | 2s                       | The grace period between a condition for deleting a binding or token is satisfied and the token or binding actually being deleted.                                                                                                 |
| --disable-http2                                       | DISABLEHTTP2                   | true                     | Whether to disable webhook communication over HTTP/2 protocol or not.                                                                                                                                                              |
| --storage-config-json                                 | STORAGECONFIGJSON              |                          | JSON with ESO ClusterSecretStore provider's configuration. Example: '{\"fake\":{}}'                                                                                                                                                |
|

## Token Storage
### Vault

Vault is default token storage. Vault instance is deployed together with SPI components. `make deploy_minikube` or `make deploy_openshift` configures it automatically.
For other deployments, like [infra-deployments](https://github.com/redhat-appstudio/infra-deployments) run `./hack/vault-init.sh` manually.

There are a couple of support scripts to work with Vault
- `./hack/vault-init.sh` - Initialize and configure Vault instance.
  - To change path prefix for the SPI data (default is `spi`), set `SPI_DATA_PATH_PREFIX` environment variable. Value must be without leading and trailing slashes (e.g.: `SPI_DATA_PATH_PREFIX=all/spi/tokens/here`). To configure Vault path prefix in SPI see `--vault-data-path-prefix` SPI property.
- `./hack/vault-generate-template.sh` - generates deployment yamls from [vault-helm](https://github.com/hashicorp/vault-helm). These should be commited in this repository.
- injected in vault pod `/vault/userconfig/scripts/poststart.sh` - unseal vault storage. Runs automatically after pod startup.
- injected in vault pod `/vault/userconfig/scripts/root.sh` - vault login as root with generated root token. Can be used for manual configuration.

### AWS Secrets Manager

To enable AWS Secrets Manager as token storage, set `--tokenstorage=aws`. `make deploy_minikube_aws` or `make deploy_openshift_aws` configures it automatically.

SPI require 2 AWS configuration files, `config` and `credentials`. These can be set with `--aws-config-filepath` and `--aws-credentials-filepath`.

_Note: If you've used AWS cli locally, AWS configuration files should be at `~/.aws/config` and `~/.aws/credentials`. To create the secret, use `./hack/aws-create-credentials-secret.sh`_

### External secret powered storage
Remote Secret operator can be configured to use [external secret powered storage](https://external-secrets.io/latest/introduction/overview/#secretstore). To enable it, set `--tokenstorage=es`.
Additionally to that, `--storage-config-json` must be set to valid JSON with ESO ClusterSecretStore provider's configuration.

AWS example:
```bash
kubectl patch configmap remote-secret-controller-manager-environment-config \
  -n remotesecret\
  --type merge \
  -p '{"data":{"TOKENSTORAGE":"es","STORAGECONFIGJSON":"{\"aws\":{\"region\":\"us-east-1\",\"service\":\"SecretsManager\",\"auth\":{\"secretRef\":{\"accessKeyIDSecretRef\":{\"name\":\"aws-secretsmanager-credentials-eso\",\"namespace\":\"remotesecret\",\"key\":\"aws_access_key_id\"},\"secretAccessKeySecretRef\":{\"namespace\":\"remotesecret\",\"name\":\"aws-secretsmanager-credentials-eso\",\"key\":\"aws_secret_access_key\"}}}}}"}}'
```
In this example we are using AWS Secrets Manager as a secret store. It is configured to use us-east-1 region and credentials from `aws-secretsmanager-credentials-eso` secret. This secret must be created in namespace `remotesecret`.

Vault example:
```bash
kubectl patch configmap remote-secret-controller-manager-environment-config \
-n remotesecret\
--type merge \
-p '{"data":{"TOKENSTORAGE":"es","STORAGECONFIGJSON":"{\"vault\":{\"server\":\"http://vault.spi-vault.svc.cluster.local:8200\",\"path\":\"spi\",\"version\":\"v2\",\"auth\":{\"appRole\":{\"path\":\"approle\",\"roleId\":\"'"$VAULT_APP_ROLE_ID"'\",\"secretRef\":{\"name\":\"vault-approle-remote-secret-operator\",\"key\":\"secret_id\",\"namespace\":\"remotesecret\"}}}}}"}}'
```
In this example we are using Vault as a secret store. It is configured to use `http://vault.spi-vault.svc.cluster.local:8200` as a server, `spi` as a path, `v2` as a version and `approle` as an authentication method. AppRole authentication method is configured to use `vault-approle-remote-secret-operator` secret to get `secret_id` and `role_id` values. This secret must be created in namespace `remotesecret`.

## [Service Level Objectives monitoring](#service-level-objectives-monitoring)

 There is a defined list of Service Level Objectives (SLO-s), for which RemoteSecret operator should collect indicator metrics, 
 and expose them on its monitoring framework. 
 TODO
