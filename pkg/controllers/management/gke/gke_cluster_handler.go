package gke

import (
	"context"
	"time"

	"github.com/rancher/rancher/pkg/catalog/manager"
	v3 "github.com/rancher/rancher/pkg/generated/controllers/management.cattle.io/v3"
	corev1 "github.com/rancher/rancher/pkg/generated/norman/core/v1"
	mgmtv3 "github.com/rancher/rancher/pkg/generated/norman/management.cattle.io/v3"
	projectv3 "github.com/rancher/rancher/pkg/generated/norman/project.cattle.io/v3"
	"github.com/rancher/rancher/pkg/namespace"
	"github.com/rancher/rancher/pkg/systemaccount"
	"github.com/rancher/rancher/pkg/types/config"
	typeDialer "github.com/rancher/rancher/pkg/types/config/dialer"
	"github.com/rancher/rancher/pkg/wrangler"
	wranglerv1 "github.com/rancher/wrangler/pkg/generated/controllers/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/rest"
)

const (
	systemNS            = "cattle-system"
	aksAPIGroup         = "gke.cattle.io"
	aksV1               = "gke.cattle.io/v1"
	aksOperatorTemplate = "system-library-rancher-aks-operator"
	aksOperator         = "rancher-aks-operator"
	localCluster        = "local"
	enqueueTime         = time.Second * 5
	importedAnno        = "gke.cattle.io/imported"
)

type gkeOperatorValues struct {
	HTTPProxy  string `json:"httpProxy,omitempty"`
	HTTPSProxy string `json:"httpsProxy,omitempty"`
	NoProxy    string `json:"noProxy,omitempty"`
}

type gkeOperatorController struct {
	clusterEnqueueAfter  func(name string, duration time.Duration)
	secretsCache         wranglerv1.SecretCache
	templateCache        v3.CatalogTemplateCache
	projectCache         v3.ProjectCache
	appLister            projectv3.AppLister
	appClient            projectv3.AppInterface
	nsClient             corev1.NamespaceInterface
	clusterClient        v3.ClusterClient
	catalogManager       manager.CatalogManager
	systemAccountManager *systemaccount.Manager
	dynamicClient        dynamic.NamespaceableResourceInterface
	clientDialer         typeDialer.Factory
}

func Register(ctx context.Context, wContext *wrangler.Context, mgmtCtx *config.ManagementContext) {
	gkeClusterConfigResource := schema.GroupVersionResource{
		Group:    aksAPIGroup,
		Version:  "v1",
		Resource: "aksclusterconfigs",
	}

	gkeCCDynamicClient := mgmtCtx.DynamicClient.Resource(gkeClusterConfigResource)
	e := &gkeOperatorController{
		clusterEnqueueAfter:  wContext.Mgmt.Cluster().EnqueueAfter,
		secretsCache:         wContext.Core.Secret().Cache(),
		templateCache:        wContext.Mgmt.CatalogTemplate().Cache(),
		projectCache:         wContext.Mgmt.Project().Cache(),
		appLister:            mgmtCtx.Project.Apps("").Controller().Lister(),
		appClient:            mgmtCtx.Project.Apps(""),
		nsClient:             mgmtCtx.Core.Namespaces(""),
		clusterClient:        wContext.Mgmt.Cluster(),
		catalogManager:       mgmtCtx.CatalogManager,
		systemAccountManager: systemaccount.NewManager(mgmtCtx),
		dynamicClient:        gkeCCDynamicClient,
		clientDialer:         mgmtCtx.Dialer,
	}

	wContext.Mgmt.Cluster().OnChange(ctx, "aks-operator-controller", e.onClusterChange)
}

func (g *gkeOperatorController) onClusterChange(key string, cluster *mgmtv3.Cluster) (*mgmtv3.Cluster, error) {
	if cluster == nil || cluster.DeletionTimestamp != nil {
		return cluster, nil
	}

	if cluster.Spec.GKEConfig == nil {
		return cluster, nil
	}

	return cluster, nil
}

func (g *gkeOperatorController) updateAKSClusterConfig(cluster *mgmtv3.Cluster, aksClusterConfigDynamic *unstructured.Unstructured, spec map[string]interface{}) (*mgmtv3.Cluster, error) {
	return nil, nil
}

func (g *gkeOperatorController) generateAndSetServiceAccount(cluster *mgmtv3.Cluster) (*mgmtv3.Cluster, error) {
	return nil, nil
}

// buildGKECCCreateObject returns an object that can be used with the kubernetes dynamic client to
// create an GKEClusterConfig that matches the spec contained in the cluster's GKEConfig.
func buildGKECCCreateObject(cluster *mgmtv3.Cluster) (*unstructured.Unstructured, error) {
	return nil, nil
}

// recordAppliedSpec sets the cluster's current spec as its appliedSpec
func (g *gkeOperatorController) recordAppliedSpec(cluster *mgmtv3.Cluster) (*mgmtv3.Cluster, error) {
	return nil, nil
}

func (g *gkeOperatorController) deployAKSOperator() error {
	template, err := g.templateCache.Get(namespace.GlobalNamespace, aksOperatorTemplate)
	if err != nil {
		return err
	}

	if template.APIVersion != "" {
		return nil
	}
	return nil
}

func (g *gkeOperatorController) getKubeConfig(cluster *mgmtv3.Cluster) (*rest.Config, error) {
	return nil, nil
}
