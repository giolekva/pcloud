# ACME webhook for Gandi (cert-manager-webhook-gandi)
`cert-manager-webhook-gandi` is an ACME webhook for [cert-manager]. It provides an ACME (read: Let's Encrypt) webhook for [cert-manager], which allows to use a `DNS-01` challenge with [Gandi]. This allows to provide Let's Encrypt certificates to [Kubernetes] for service protocols other than HTTP and furthermore to request wildcard certificates. Internally it uses the [Gandi LiveDNS API] to communicate with Gandi.

Quoting the [ACME DNS-01 challenge]:

> This challenge asks you to prove that you control the DNS for your domain name by putting a specific value in a TXT record under that domain name. It is harder to configure than HTTP-01, but can work in scenarios that HTTP-01 can’t. It also allows you to issue wildcard certificates. After Let’s Encrypt gives your ACME client a token, your client will create a TXT record derived from that token and your account key, and put that record at _acme-challenge.<YOUR_DOMAIN>. Then Let’s Encrypt will query the DNS system for that record. If it finds a match, you can proceed to issue a certificate!


## Building
Build the container image `cert-manager-webhook-gandi:latest`:

    make build


## Image
Ready made images are hosted on Docker Hub ([image tags]). Use at your own risk:

    bwolf/cert-manager-webhook-gandi


### Release History
Refer to the [CHANGELOG](CHANGELOG.md) file.


## Compatibility
This webhook has been tested with [cert-manager] v1.5.4 and Kubernetes v1.22.2 on `amd64`. In theory it should work on other hardware platforms as well but no steps have been taken to verify this. Please drop me a note if you had success.


## Testing with Minikube
1. Build this webhook in Minikube:

        minikube start --memory=4G --more-options
        eval $(minikube docker-env)
        make build
        docker images | grep webhook

2. Install [cert-manager] with [Helm]:

        helm repo add jetstack https://charts.jetstack.io

        helm install cert-manager jetstack/cert-manager \
            --namespace cert-manager \
            --create-namespace \
            --set installCRDs=true \
            --version v1.5.4 \
            --set 'extraArgs={--dns01-recursive-nameservers=8.8.8.8:53\,1.1.1.1:53}'

        kubectl get pods --namespace cert-manager --watch

   **Note**: refer to Name servers in the official [documentation][setting-nameservers-for-dns01-self-check] according the `extraArgs`.

   **Note**: ensure that the custom CRDS of cert-manager match the major version of the cert-manager release by comparing the URL of the CRDS with the helm info of the charts app version:

            helm search repo jetstack

   Example output:

            NAME                    CHART VERSION   APP VERSION     DESCRIPTION
            jetstack/cert-manager   v1.5.4          v1.5.4          A Helm chart for cert-manager

   Check the state and ensure that all pods are running fine (watch out for any issues regarding the `cert-manager-webhook-` pod and its volume mounts):

            kubectl describe pods -n cert-manager | less


3. Create the secret to keep the Gandi API key in the cert-manager namespace:

        kubectl create secret generic gandi-credentials \
            --namespace cert-manager --from-literal=api-token='<GANDI-API-KEY>'

   *The `Secret` must reside in the same namespace as `cert-manager`.*

4. Deploy this webhook (add `--dry-run` to try it and `--debug` to inspect the rendered manifests; Set `logLevel` to 6 for verbose logs):

   *The `features.apiPriorityAndFairness` argument must be removed or set to `false` for Kubernetes older than 1.20.*

        helm install cert-manager-webhook-gandi \
            --namespace cert-manager \
            --set features.apiPriorityAndFairness=true \
            --set image.repository=cert-manager-webhook-gandi \
            --set image.tag=latest \
            --set logLevel=2 \
            ./deploy/cert-manager-webhook-gandi

   To deploy using the image from Docker Hub (for example using the `0.2.0` tag):

        helm install cert-manager-webhook-gandi \
            --namespace cert-manager \
            --set features.apiPriorityAndFairness=true \
            --set image.tag=0.2.0 \
            --set logLevel=2 \
            ./deploy/cert-manager-webhook-gandi

   To deploy using the Helm repository (for example using the `v0.2.0` version):

        helm install cert-manager-webhook-gandi \
            --repo https://bwolf.github.io/cert-manager-webhook-gandi \
            --version v0.2.0 \
            --namespace cert-manager \
            --set features.apiPriorityAndFairness=true \
            --set logLevel=2

   Check the logs

            kubectl get pods -n cert-manager --watch
            kubectl logs -n cert-manager cert-manager-webhook-gandi-XYZ

6. Create a staging issuer (email addresses with the suffix `example.com` are forbidden).

   See [letsencrypt-staging-issuer.yaml](examples/issuers/letsencrypt-staging-issuer.yaml)

   Don't forget to replace email `invalid@example.com`.

   Check status of the Issuer:

        kubectl describe issuer letsencrypt-staging

   You can deploy a ClusterIssuer instead : see [letsencrypt-staging-clusterissuer.yaml](examples/issuers/letsencrypt-staging-clusterissuer.yaml)

   *Note*: The production Issuer is [similar][ACME documentation].

7. Issue a [Certificate] for your domain: see [certif-example-com.yaml](examples/certificates/certif-example-com.yaml)

   Replace `your-domain` and `your.domain` in the [certif-example-com.yaml](examples/certificates/certif-example-com.yaml)

   Create the Certificate:

        kubectl apply -f ./examples/certificates/certif-example-com.yaml

   Check the status of the Certificate:

        kubectl describe certificate example-com

   Display the details like the common name and subject alternative names:

        kubectl get secret example-com-tls -o yaml

   If you deployed a ClusterIssuer : use [certif-example-com-clusterissuer.yaml](examples/certificates/certif-example-com-clusterissuer.yaml)

8. Issue a wildcard Certificate for your domain: see [certif-wildcard-example-com.yaml](examples/certificates/certif-wildcard-example-com.yaml)

   Replace `your-domain` and `your.domain` in the [certif-wildcard-example-com.yaml](examples/certificates/certif-wildcard-example-com.yaml)

   Create the Certificate:

        kubectl apply -f ./examples/certificates/certif-wildcard-example-com.yaml

   Check the status of the Certificate:

        kubectl describe certificate wildcard-example-com

   Display the details like the common name and subject alternative names:

        kubectl get secret wildcard-example-com-tls -o yaml

   If you deployed a ClusterIssuer : use [certif-wildcard-example-com-clusterissuer.yaml](examples/certificates/certif-wildcard-example-com-clusterissuer.yaml)

9. Uninstall this webhook:

        helm uninstall cert-manager-webhook-gandi --namespace cert-manager
        kubectl delete gandi-credentials --namespace cert-manager

10. Uninstalling cert-manager:
    This is out of scope here. Refer to the official [documentation][cert-manager-uninstall].


## Development
**Note**: If some tool (IDE or build process) fails resolving a dependency, it may be the cause that a indirect dependency uses `bzr` for versioning. In such a case it may help to put the `bzr` binary into `$PATH` or `$GOPATH/bin`.


## Release process (automated with [GitHub actions](.github/workflows/main.yml))
- Changes in the Go code result in the build of a Docker image and the release of a new Helm chart
- Changes at Helm chart level only, result in the release of a new Chart without building a new Docker image
- All other changes are pushed to master
- All versions are to be documented in [CHANGELOG](CHANGELOG.md)

**Note**: All changes to the Go code or Helm chart must go with a version tag `vX.X.X` to trigger the GitHub workflow

**Note**: Any Helm chart release results in the creation of a [GitHub release](https://github.com/bwolf/cert-manager-webhook-gandi/releases)

## Conformance test
Please note that the test is not a typical unit or integration test. Instead it invokes the web hook in a Kubernetes-like environment which asks the web hook to really call the DNS provider (.i.e. Gandi). It attempts to create an `TXT` entry like `cert-manager-dns01-tests.example.com`, verifies the presence of the entry via Google DNS. Finally it removes the entry by calling the cleanup method of web hook.

As said above, the conformance test is run against the real Gandi API. Therefore you *must* have a Gandi account, a domain and an API key.

``` shell
cp testdata/gandi/api-key.yaml.sample testdata/gandi/api-key.yaml
echo -n $YOUR_GANDI_API_KEY | base64 | pbcopy # or xclip
$EDITOR testdata/gandi/api-key.yaml
TEST_ZONE_NAME=example.com. make test
make clean
```


[ACME DNS-01 challenge]: https://letsencrypt.org/docs/challenge-types/#dns-01-challenge
[ACME documentation]: https://cert-manager.io/docs/configuration/acme/
[Certificate]: https://cert-manager.io/docs/usage/certificate/
[cert-manager]: https://cert-manager.io/
[Gandi]: https://gandi.net/
[Gandi LiveDNS API]: https://api.gandi.net/docs/livedns/
[Helm]: https://helm.sh
[image tags]: https://hub.docker.com/r/bwolf/cert-manager-webhook-gandi
[Kubernetes]: https://kubernetes.io/
[setting-nameservers-for-dns01-self-check]: https://cert-manager.io/docs/configuration/acme/dns01/#setting-nameservers-for-dns01-self-check
[cert-manager-uninstall]: https://cert-manager.io/docs/installation/uninstall/kubernetes/
