module github.com/kanzifucius/promethues-operator-slos

go 1.13

require (
	github.com/cespare/xxhash v1.1.0 // indirect
	github.com/ghodss/yaml v1.0.0
	github.com/go-logr/logr v0.1.0
	github.com/onsi/ginkgo v1.12.1
	github.com/onsi/gomega v1.10.1
	github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring v0.44.1
	github.com/prometheus/common v0.4.1
	github.com/prometheus/prometheus v2.5.0+incompatible
	github.com/stretchr/testify v1.4.0
	go.uber.org/zap v1.10.0
	k8s.io/apimachinery v0.18.6
	k8s.io/client-go v0.18.6
	sigs.k8s.io/controller-runtime v0.6.3
)
