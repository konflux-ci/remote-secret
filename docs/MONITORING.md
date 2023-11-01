## Monitoring and Alerting
This document covers the monitoring and alerting capabilities of the Remote Secret operator.

### Installation of the monitoring stack
For the minikube installations, a `./hack/deploy-minikube-monitoring` script can be used to deploy the monitoring stack. This script will deploy the Prometheus and Grafana with Go runtime and RemoteSecret operator metrics dashboards.


### Available prometheus metrics list

The RemoteSecret operator provides the following metrics:

| Metric name                   | Description                                                                    | Labels                |
|-------------------------------|--------------------------------------------------------------------------------|-----------------------|
| `data_upload_rejected_total`  | The number of remote secret data uploads rejected by the webhook or controller | `operation`, `reason` |
| `vault_request_count_total`   | The request counts to Vault categorized by HTTP method status code             | `method`, `status`    |
| `vault_response_time_seconds` | The response time of Vault requests categorized by HTTP method and status code | `method`, `status`    |