package provider

import (
	"context"
	"crypto/md5"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"text/template"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/segmentio/kubeapply/data"
	"github.com/segmentio/kubeapply/pkg/cluster"
	"github.com/segmentio/kubeapply/pkg/cluster/diff"
	"github.com/segmentio/kubeapply/pkg/cluster/kube"
	"github.com/segmentio/kubeapply/pkg/config"
	"github.com/segmentio/kubeapply/pkg/util"
	log "github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

var (
	kubeConfigTemplate = template.Must(
		template.New("kubeconfig").Parse(
			string(data.MustAsset("pkg/provider/templates/kubeconfig.yaml")),
		),
	)
	cache = newDiffCache()
)

type providerContext struct {
	autoCreateNamespaces bool
	clusterClient        cluster.ClusterClient
	config               config.ClusterConfig
	keepExpanded         bool
	rawClient            kubernetes.Interface
	tempDir              string
}

// Provider is the entrypoint for creating a new kubeapply terraform provider instance.
func Provider(providerCtx *providerContext) *schema.Provider {
	return &schema.Provider{
		ConfigureContextFunc: func(
			ctx context.Context,
			d *schema.ResourceData,
		) (interface{}, diag.Diagnostics) {
			// Use a provider context that's injected in for testing
			// purposes.
			if providerCtx != nil {
				return providerCtx, diag.Diagnostics{}
			}

			return providerConfigure(ctx, d)
		},
		Schema: map[string]*schema.Schema{
			// Basic info about the cluster
			"cluster_name": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "Name of the cluster",
			},
			"region": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "Region",
			},
			"account_name": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "Region",
			},

			// Cluster API and auth information
			"host": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "The hostname (in form of URI) of Kubernetes master.",
			},
			"cluster_ca_certificate": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "PEM-encoded root certificates bundle for TLS authentication.",
			},
			"token": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "Token to authenticate an service account",
			},

			// Whether we should automatically create namespaces
			"auto_create_namespaces": {
				Type:        schema.TypeBool,
				Description: "Automatically create namespaces before each diff",
				Default:     true,
				Optional:    true,
			},
		},
		ResourcesMap: map[string]*schema.Resource{
			"kubeapply_profile": profileResource(),
		},
		DataSourcesMap: map[string]*schema.Resource{},
	}
}

func providerConfigure(
	ctx context.Context,
	d *schema.ResourceData,
) (interface{}, diag.Diagnostics) {
	var diags diag.Diagnostics

	log.Info("Creating provider")

	config := config.ClusterConfig{
		Cluster: d.Get("cluster_name").(string),
		Region:  d.Get("region").(string),
		Env:     d.Get("account_name").(string),
	}

	tempDir, err := ioutil.TempDir("", "kubeconfig")
	if err != nil {
		return nil, diag.FromErr(err)
	}
	kubeConfigPath := filepath.Join(tempDir, "kubeconfig.yaml")

	out, err := os.Create(kubeConfigPath)
	if err != nil {
		return nil, diag.FromErr(err)
	}
	defer out.Close()

	err = kubeConfigTemplate.Execute(
		out,
		struct {
			Server string
			CAData string
			Token  string
		}{
			Server: d.Get("host").(string),
			CAData: d.Get("cluster_ca_certificate").(string),
			Token:  d.Get("token").(string),
		},
	)
	if err != nil {
		return nil, diag.FromErr(err)
	}

	log.Infof("Setting provider kubeconfig path to %s", kubeConfigPath)
	config.KubeConfigPath = kubeConfigPath

	log.Info("Creating cluster client")
	clusterClient, err := cluster.NewKubeClusterClient(
		ctx,
		&cluster.ClusterClientConfig{
			ClusterConfig: &config,
		},
	)
	if err != nil {
		return nil, diag.FromErr(err)
	}

	log.Info("Creating raw kube client")
	kubeClientConfig, err := clientcmd.BuildConfigFromFlags("", kubeConfigPath)
	if err != nil {
		return nil, diag.FromErr(err)
	}

	rawClient, err := kubernetes.NewForConfig(kubeClientConfig)
	if err != nil {
		return nil, diag.FromErr(err)
	}

	providerCtx := providerContext{
		autoCreateNamespaces: d.Get("auto_create_namespaces").(bool),
		config:               config,
		clusterClient:        clusterClient,
		rawClient:            rawClient,
		tempDir:              tempDir,
	}

	return &providerCtx, diags
}

type expandResult struct {
	expandedDir string
	manifests   []kube.Manifest
	resources   map[string]string
	totalHash   string
}

func (p *providerContext) expand(
	ctx context.Context,
	path string,
	params map[string]interface{},
) (*expandResult, error) {
	config := p.config
	config.Parameters = map[string]interface{}{}

	for key, value := range params {
		config.Parameters[key] = value
	}

	timeStamp := time.Now().UnixNano()
	expandedDir := filepath.Join(
		p.tempDir,
		"expanded",
		fmt.Sprintf("%d", timeStamp),
	)
	if err := util.RecursiveCopy(path, expandedDir); err != nil {
		return nil, err
	}

	if err := util.ApplyTemplate(expandedDir, config, true, true); err != nil {
		return nil, err
	}

	manifests, err := kube.GetManifests([]string{expandedDir})
	if err != nil {
		return nil, err
	}

	resources := map[string]string{}
	for _, manifest := range manifests {
		resources[manifest.ID] = manifest.Hash
	}

	return &expandResult{
		expandedDir: expandedDir,
		manifests:   manifests,
		resources:   resources,
		totalHash:   p.manifestsHash(manifests),
	}, nil
}

func (p *providerContext) diff(
	ctx context.Context,
	path string,
) ([]diff.Result, error) {
	return p.clusterClient.DiffStructured(ctx, []string{path}, false)
}

func (p *providerContext) apply(
	ctx context.Context,
	path string,
) ([]byte, error) {
	return p.clusterClient.Apply(ctx, []string{path}, false)
}

func (p *providerContext) createNamespaces(
	ctx context.Context,
	manifests []kube.Manifest,
) error {
	if !p.autoCreateNamespaces {
		log.Info("Not auto-creating namespaces since auto_create_namespaces is false")
		return nil
	}

	manifestNamespacesMap := map[string]struct{}{}

	for _, manifest := range manifests {
		if manifest.Head.Metadata.Namespace != "" {
			manifestNamespacesMap[manifest.Head.Metadata.Namespace] = struct{}{}
		}
	}

	apiNamespaces, err := p.rawClient.CoreV1().Namespaces().List(ctx, metav1.ListOptions{})
	if err != nil {
		return err
	}

	apiNamespacesMap := map[string]struct{}{}
	for _, namespace := range apiNamespaces.Items {
		apiNamespacesMap[namespace.Name] = struct{}{}
	}

	for namespace := range manifestNamespacesMap {
		if _, ok := apiNamespacesMap[namespace]; !ok {
			log.Infof("Namespace %s is in manifest but not API, creating", namespace)
			_, err = p.rawClient.CoreV1().Namespaces().Create(
				ctx,
				&corev1.Namespace{
					ObjectMeta: metav1.ObjectMeta{
						Name: namespace,
					},
				},
				metav1.CreateOptions{},
			)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func (p *providerContext) manifestsHash(
	manifests []kube.Manifest,
) string {
	hash := md5.New()

	hash.Write([]byte(p.config.Cluster))
	hash.Write([]byte(p.config.Region))
	hash.Write([]byte(p.config.Env))

	for _, manifest := range manifests {
		hash.Write([]byte(manifest.Hash))
	}

	return fmt.Sprintf("%x", hash.Sum(nil))
}

func (p *providerContext) cleanExpanded(
	result *expandResult,
) error {
	if p.keepExpanded {
		return nil
	}
	return os.RemoveAll(result.expandedDir)
}
