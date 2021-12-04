# PCloud Installation Instructions

## Overview
PCloud is multitenant by default and its services are split into two **infrastructure** and **application** categories.
Infrastructure services are shared among all PCloud instances installed on the same cluster of servers, while application services are installed separately for each instance.

PCloud runs on top of Kubernetes, bellow are listed all namespaces needed for healthy setup with services running inside them.

Infrastructure:
* **pcloud-networking-metallb**: runs [metallb](https://metallb.universe.tf) so we can use LoadBalancer type services when running PCloud on on-prem hardware.
* **pcloud-ingress-public**: runs [ingress-nginx](https://github.com/kubernetes/ingress-nginx) to expose services outside of the cluster.
* **pcloud-cert-manager**: runs [cert-manager](https://cert-manager.io) to automate generating SSL certificates for services.
* **pcloud-nebula-controller**: runs [nebula](https://github.com/slackhq/nebula) [controller](https://github.com/giolekva/pcloud/tree/main/core/nebula/controller) to automate signing certificates for VPN mesh network nodes.
* **pcloud-oauth2-manager**: runs [Ory Hydra Maester](https://github.com/ory/hydra-maester) to automate generating registering OAuth2 clients.
* **pcloud-mail-gateway**: runs instance of [maddy](https://maddy.email) as a SMTP MX gateway routing incoming emails to proper PCloud instances and routing outgoing emails to the *world*.
* **longhorn-system**: runs [Longhorn](https://longhorn.io) to support persistent volumens.
* **pcloud-kubed**: runs [kubed](https://github.com/kubeops/config-syncer) to automatically sync wildcard and root level domain certificates across all namespaces of one PCloud instance.

Applications, assuming PCloud instance is named **example**:
* **example-ingress-private**: runs another instance of ingress-nginx to handle traffic coming from private mesh network.
* **example-core-auth**: runs [Ory Kratos](https://www.ory.sh/kratos/docs/) for user authentication, [Ory Hydra](https://www.ory.sh/hydra/docs/) to handle OAuth2 and OpenID connect, and [PCloud serlfservice UI](https://github.com/giolekva/pcloud/tree/main/core/auth/ui) which provides registration/authentication UI.
* **example-app-pihole**: runs [Pi-hole](https://pi-hole.net) to provide DNS based network wide Ad Blocker for private mesh network nodes.x
* **example-app-maddy**: runs another instance of maddy to provide SMTP and IMAP services to users.
* **example-app-vaultwarden**: runs [Vaultwarden](https://github.com/dani-garcia/vaultwarden) which is alternate implementation of Bitwarden Server API to provide users secure password manager.
* **example-app-matrix**: runs [Matrix](https://matrix.org) [Synapse](https://github.com/matrix-org/synapse) homeserver.
* **example**: holds information about above mentioned namespaces.

## Setup instructions
### Prerequisites:
* Installation scripts expect working Kubernetes cluster.
* Helm 3: https://helm.sh/docs/intro/install/
* Helm diff plugin: https://github.com/databus23/helm-diff#install
* Helm secrets plugin: https://github.com/jkroepke/helm-secrets/wiki/Installation
* Mozilla SOPS: https://github.com/mozilla/sops#1download
* helmfile: https://github.com/roboll/helmfile#installation

### Infrastructure:
First update mail-gateway domains section in `pcloud/helmfile/infra/helmfile.yaml`
* **name**: domain name
* **namespace**: **bobo**-app-maddy
* **mx**: mail.<same domain name as above>
* **certificateissuer**: **bobo**-public

Run:
```shell
cd pcloud/helmfile/infra/
helmfile -e prod apply --skip-diff-on-install
```

### Applications:
First configure `.sops.yaml` file in pcloud/helmfile/apps directory.

Then create `secrets.bobo.yaml` file with for keys:
* **gandiAPIToken**: your Gandi API token
* **piholeOAuth2ClientSecret**: generate using https://github.com/oauth2-proxy/oauth2-proxy/blob/master/docs/docs/configuration/overview.md#generating-a-cookie-secret
* **piholeOAuth2CookieSecret**: ditto
* **matrixOAuth2ClientSecret**: ditto

Next add new environment in `pcloud/helmfile/apps/helmfile.yaml`
```yaml
  bobo:
    secrets:
    - secrets.bobo.yaml
    values:
    - pcloudEnvName: pcloud
    - id: bobo
    - namespacePrefix: bobo-
    - domain: <repeat primary domain from infra/helmfile.yaml>
    - contactEmail: <your email address>
    - certManagerNamespace: pcloud-cert-manager
    - mxHostname: <repeat primary domain mx hostname from infra/helmfile.yaml>
    - mailGatewayAddress: "tcp://maddy.pcloud-mail-gateway.svc.cluster.local:587"
    - matrixStorageSize: 100Gi
    - publicIP: <clusters public IP>
    - lighthouseMainIP: 111.0.0.1
    - lighthouseMainPort: 4242
    - lighthouseAuthUIIP: 111.0.0.2
```

Run **(you might have to run this step multiple times :)**
```shell
cd pcloud/helmfile/apps/
helmfile -e bobo apply --skip-diff-on-install
```
