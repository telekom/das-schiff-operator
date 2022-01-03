module github.com/telekom/das-schiff-operator

go 1.16

require (
	filippo.io/age v1.0.0-beta7
	github.com/go-git/go-billy/v5 v5.3.1
	github.com/go-git/go-git/v5 v5.4.2
	github.com/go-logr/logr v0.4.0
	github.com/infobloxopen/infoblox-go-client v1.1.1-0.20210326040601-b71324be2432
	github.com/onsi/ginkgo v1.16.4
	github.com/onsi/gomega v1.16.0
	github.com/spf13/viper v1.9.0
	go.mozilla.org/sops/v3 v3.7.1
	gopkg.in/yaml.v3 v3.0.0-20210107192922-496545a6307b
	k8s.io/api v0.22.2
	k8s.io/apimachinery v0.22.2
	k8s.io/client-go v0.22.2
	sigs.k8s.io/cluster-api v1.0.1
	sigs.k8s.io/cluster-api-provider-vsphere v1.0.1
	sigs.k8s.io/controller-runtime v0.10.3
)

replace sigs.k8s.io/cluster-api => sigs.k8s.io/cluster-api v1.0.1
