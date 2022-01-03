# Das Schiffs Operator

Das Schiff is a GitOps focused Kubernetes as a Service Engine/Distribution used within [Deutsche Telekom Technik](https://de.wikipedia.org/wiki/Telekom_Deutschland#Deutsche_Telekom_Technik_GmbH). Das Schiff's architecture is open source as well at [telekom/das-schiff](https://github.com/telekom/das-schiff).

This operator contains custom controllers that are used for Das Schiff. It's built using the [Operator SDK](https://sdk.operatorframework.io/).

## Modules

While it's not properly organized, it technically concists of several somewhat independent modules. Currently they are just separate controllers using shared libraries from `pkg`.

### CAPV IPAM integration with Infoblox

We're using Cluster API and the cluster-api-provider-vsphere to deploy clusters to various vSphere instances. As we want to use Infoblox for IP address management, we've built a custom controller that integrates CAPV with Infoblox.

CAPV waits with deployment until VSphereMachine resources are complete, which requires all network interfaces to be configured correctly. We use this behavior to dynamically assign IP addresses to the created machines. The `./controllers/vspheremachine_ipam_controller.go` obvserves VSphereMachines and assigns addresses from Infoblox as specified by several annotations:

```yaml
ipam.schiff.telekom.de/InfobloxNetworkView: "DEFAULT" # separation feature of Infoblox
ipam.schiff.telekom.de/NetworkName: "net1" # used to identify the interface
ipam.schiff.telekom.de/Subnet: "10.10.42.0/24" # identifies the network
ipam.schiff.telekom.de/DNSZone: "example.com" # controller creates host entries in this zone
```

### Sops Encrypted Kubeconfig Backups to Git

We're very Git focused and store lots of things in an internal GitLab instance. While we use OIDC based SSO to access our clusters, we also need an emergency backdoor in case the SSO system is unavailable. CAPI stores the admin Kubeconfigs of created clusters in the management cluster. In order to not be dependant on the availability of those clusters, the `./controllers/kubeadmincontrollplane_controller.go` creates backups of those configs to a GitLab repository.

Since we can't store those configs in plain text for obvious security reasons we're encrypting them with [mozilla/sops](https://github.com/mozilla/sops) using [age](https://github.com/FiloSottile/age) keys.

## Building

To build you'll need to have a working Go development environment that's at least the version specified in the `./go.mod` file.
Then simply run `make` to build the operator.

Afterwards you can also build the Docker image using `docker build .`

## Running for Development

The repository includes a Tilt config. See https://tilt.dev for more information about that tool.

## Testing

The `envtest` package of https://github.com/kubernetes-sigs/controller-runtime is used for testing. It requres a set of tools to be located in `/usr/local/kubebuilder/bin`. To install them run the following command

```bash
sudo hack/setup-envtest.sh
```

You can then execute all tests by running:

```
make test
```
