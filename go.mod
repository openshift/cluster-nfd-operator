module github.com/openshift/cluster-nfd-operator

go 1.15

require (
	github.com/go-logr/logr v0.3.0 // indirect
	github.com/go-logr/zapr v0.3.0 // indirect
	github.com/go-openapi/spec v0.19.3
	github.com/golang/glog v0.0.0-20160126235308-23def4e6c14b
	github.com/markbates/inflect v1.0.4 // indirect
	github.com/openshift/api v0.0.0-20200827090112-c05698d102cf
	github.com/openshift/client-go v0.0.0-20200827190008-3062137373b5
	github.com/operator-framework/operator-sdk v0.4.1-0.20190129222657-43d37ce85826
	// Kubernetes 1.19
	k8s.io/api v0.19.0
	k8s.io/apiextensions-apiserver v0.19.0
	k8s.io/apimachinery v0.19.0
	k8s.io/client-go v0.19.0
	k8s.io/klog/v2 v2.4.0 // indirect
	k8s.io/kube-openapi v0.0.0-20200805222855-6aeccd4b50c6 // kube-openapi release-1.9 branch
	sigs.k8s.io/controller-runtime v0.6.3
)

replace k8s.io/client-go => k8s.io/client-go v0.19.0
