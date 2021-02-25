module github.com/openshift/cluster-nfd-operator

go 1.16

require (
	github.com/go-logr/logr v0.3.0
	github.com/onsi/ginkgo v1.14.1
	github.com/onsi/gomega v1.10.2
	github.com/openshift/custom-resource-status v0.0.0-20210221154447-420d9ecf2a00
	golang.org/x/tools v0.0.0-20201014231627-1610a49f37af // indirect
	google.golang.org/grpc v1.30.0 // indirect
	k8s.io/api v0.20.4
	k8s.io/apimachinery v0.20.4
	k8s.io/client-go v0.20.4
	k8s.io/component-base v0.20.4 // indirect
	k8s.io/cri-api v0.20.4
	k8s.io/klog v1.0.0
	sigs.k8s.io/controller-runtime v0.8.0
)
