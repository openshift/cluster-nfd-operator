package client

import (
	"fmt"
	"os"
	"os/user"
	"path/filepath"

	configv1client "github.com/openshift/client-go/config/clientset/versioned/typed/config/v1"
	nfdv1alpha1 "github.com/openshift/cluster-nfd-operator/pkg/apis/nfd/v1alpha1"
	apiextensionsclient "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

// GetConfig creates a *rest.Config for talking to a Kubernetes apiserver.
//
// Config precedence
//
// * KUBECONFIG environment variable pointing at a file
// * In-cluster config if running in cluster
// * $HOME/.kube/config if exists
func GetConfig() (*rest.Config, error) {
	configFromFlags := func(kubeConfig string) (*rest.Config, error) {
		if _, err := os.Stat(kubeConfig); err != nil {
			return nil, fmt.Errorf("Cannot stat kubeconfig '%s'", kubeConfig)
		}
		return clientcmd.BuildConfigFromFlags("", kubeConfig)
	}

	// If an env variable is specified with the config location, use that
	kubeConfig := os.Getenv("KUBECONFIG")
	if len(kubeConfig) > 0 {
		return configFromFlags(kubeConfig)
	}
	// If no explicit location, try the in-cluster config
	if c, err := rest.InClusterConfig(); err == nil {
		return c, nil
	}
	// If no in-cluster config, try the default location in the user's home directory
	if usr, err := user.Current(); err == nil {
		kubeConfig := filepath.Join(usr.HomeDir, ".kube", "config")
		return configFromFlags(kubeConfig)
	}

	return nil, fmt.Errorf("Could not locate a kubeconfig")
}

// GetCfgV1Client returns OpenShift *v1.ConfigV1Client for talking to a Kubernetes apiserver.
func GetCfgV1Client() (*configv1client.ConfigV1Client, error) {
	c, err := GetConfig()
	if err != nil {
		return nil, err
	}

	operatorClient, err := configv1client.NewForConfig(c)
	if err != nil {
		return nil, err
	}

	return operatorClient, nil
}

func GetClientSet() (kubernetes.Interface, error) {
	c, err := GetConfig()
	if err != nil {
		return nil, err
	}

	clientSet, err := kubernetes.NewForConfig(c)
	if err != nil {
		return nil, err
	}

	return clientSet, nil
}

func GetApiClient() (apiextensionsclient.Interface, error) {
	c, err := GetConfig()
	if err != nil {
		return nil, err
	}

	eclient, err := apiextensionsclient.NewForConfig(c)
	if err != nil {
		return nil, err
	}

	return eclient, nil

}

var SchemeGroupVersion = schema.GroupVersion{Group: "nfd.openshift.io", Version: "v1alpha1"}

// CR related
func NewClient() (*NFDV1AlphaClient, error) {
	config, err := GetConfig()
	if err != nil {
		return nil, err
	}

	scheme := runtime.NewScheme()
	SchemeBuilder := runtime.NewSchemeBuilder(addKnownTypes)
	if err := SchemeBuilder.AddToScheme(scheme); err != nil {
		return nil, err
	}

	config.GroupVersion = &SchemeGroupVersion
	config.APIPath = "/apis"
	config.ContentType = runtime.ContentTypeJSON
	config.NegotiatedSerializer = serializer.NegotiatedSerializerWrapper(runtime.SerializerInfo{})
	client, err := rest.RESTClientFor(config)
	if err != nil {
		return nil, err
	}
	return &NFDV1AlphaClient{restClient: client}, nil
}

// CR related
func addKnownTypes(scheme *runtime.Scheme) error {
	scheme.AddKnownTypes(SchemeGroupVersion,
		&nfdv1alpha1.NodeFeatureDiscovery{},
		&nfdv1alpha1.NodeFeatureDiscoveryList{},
	)
	meta_v1.AddToGroupVersion(scheme, SchemeGroupVersion)
	return nil
}

func (c *NFDV1AlphaClient) NodeFeatureDiscoveries(namespace string) NfdConfigInterface {
	return &nfdConfigClient{
		client: c.restClient,
		ns:     namespace,
	}
}

type NFDV1AlphaClient struct {
	restClient rest.Interface
}

type NfdConfigInterface interface {
	Create(obj *nfdv1alpha1.NodeFeatureDiscovery) (*nfdv1alpha1.NodeFeatureDiscovery, error)
	Update(obj *nfdv1alpha1.NodeFeatureDiscovery) (*nfdv1alpha1.NodeFeatureDiscovery, error)
	Delete(name string, options *meta_v1.DeleteOptions) error
	Get(name string) (*nfdv1alpha1.NodeFeatureDiscovery, error)
}

type nfdConfigClient struct {
	client rest.Interface
	ns     string
}

func (c *nfdConfigClient) Create(obj *nfdv1alpha1.NodeFeatureDiscovery) (*nfdv1alpha1.NodeFeatureDiscovery, error) {
	result := &nfdv1alpha1.NodeFeatureDiscovery{}
	err := c.client.Post().
		Namespace(c.ns).Resource("nodefeaturediscoveries").
		Body(obj).Do().Into(result)
	return result, err
}

func (c *nfdConfigClient) Update(obj *nfdv1alpha1.NodeFeatureDiscovery) (*nfdv1alpha1.NodeFeatureDiscovery, error) {
	result := &nfdv1alpha1.NodeFeatureDiscovery{}
	err := c.client.Put().
		Namespace(c.ns).Resource("nodefeaturediscoveries").
		Body(obj).Do().Into(result)
	return result, err
}

func (c *nfdConfigClient) Delete(name string, options *meta_v1.DeleteOptions) error {
	return c.client.Delete().
		Namespace(c.ns).Resource("nodefeaturediscoveries").
		Name(name).Body(options).Do().
		Error()
}

func (c *nfdConfigClient) Get(name string) (*nfdv1alpha1.NodeFeatureDiscovery, error) {
	result := &nfdv1alpha1.NodeFeatureDiscovery{}
	err := c.client.Get().
		Namespace(c.ns).Resource("nodefeaturediscoveries").
		Name(name).Do().Into(result)
	return result, err
}
