package gkeupstreamrefresh

import (
	"context"
	"reflect"

	v1 "github.com/rancher/gke-operator/pkg/apis/gke.cattle.io/v1"
	apimgmtv3 "github.com/rancher/rancher/pkg/apis/management.cattle.io/v3"
	v3 "github.com/rancher/rancher/pkg/generated/controllers/management.cattle.io/v3"
	mgmtv3 "github.com/rancher/rancher/pkg/generated/norman/management.cattle.io/v3"
	"github.com/rancher/rancher/pkg/settings"
	"github.com/rancher/rancher/pkg/wrangler"
	wranglerv1 "github.com/rancher/wrangler/pkg/generated/controllers/core/v1"
	"github.com/robfig/cron"
	"github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/runtime"
)

const (
	isGKEIndexer = "clusters.management.cattle.io/is-gke"
)

var (
	gkeUpstreamRefresher *gkeRefreshController
)

func init() {
	// possible settings controller, which references refresh
	// cron job, will run prior to StartgkeUpstreamCronJob.
	// This ensure the CronJob will not be nil
	gkeUpstreamRefresher = &gkeRefreshController{
		refreshCronJob: cron.New(),
	}
}

type gkeRefreshController struct {
	refreshCronJob *cron.Cron
	secretsCache   wranglerv1.SecretCache
	clusterClient  v3.ClusterClient
	clusterCache   v3.ClusterCache
}

func StartGKEUpstreamCronJob(wContext *wrangler.Context) {
	gkeUpstreamRefresher.secretsCache = wContext.Core.Secret().Cache()
	gkeUpstreamRefresher.clusterClient = wContext.Mgmt.Cluster()
	gkeUpstreamRefresher.clusterCache = wContext.Mgmt.Cluster().Cache()

	gkeUpstreamRefresher.clusterCache.AddIndexer(isGKEIndexer, func(obj *apimgmtv3.Cluster) ([]string, error) {
		if obj.Spec.GKEConfig == nil {
			return []string{}, nil
		}
		return []string{"true"}, nil
	})

	schedule, err := cron.ParseStandard(settings.GKEUpstreamRefreshCron.Get())
	if err != nil {
		logrus.Errorf("Error parsing GKE upstream cluster refresh cron. Upstream state will not be refreshed: %v", err)
		return
	}
	gkeUpstreamRefresher.refreshCronJob.Schedule(schedule, cron.FuncJob(gkeUpstreamRefresher.refreshAllUpstreamStates))
	gkeUpstreamRefresher.refreshCronJob.Start()
}

func (e *gkeRefreshController) refreshAllUpstreamStates() {
	logrus.Debugf("refreshing gke clusters' upstream states")
	clusters, err := e.clusterCache.GetByIndex(isGKEIndexer, "true")
	if err != nil {
		logrus.Error("error trying to refresh gke clusters' upstream states")
		return
	}

	for _, cluster := range clusters {
		if _, err := e.refreshClusterUpstreamSpec(cluster); err != nil {
			logrus.Errorf("error refreshing gke cluster [%s] upstream state", cluster.Name)
		}
	}
}

func (e *gkeRefreshController) refreshClusterUpstreamSpec(cluster *mgmtv3.Cluster) (*mgmtv3.Cluster, error) {
	if cluster == nil || cluster.DeletionTimestamp != nil {
		return nil, nil
	}

	if cluster.Spec.GKEConfig == nil {
		return cluster, nil
	}

	logrus.Infof("checking cluster [%s] upstream state for changes", cluster.Name)

	if cluster.Status.GKEStatus.UpstreamSpec == nil {
		logrus.Infof("initial upstream spec for cluster [%s] has not been set by gke cluster handler yet, skipping", cluster.Name)
		return cluster, nil
	}

	upstreamSpec, err := GetComparableUpstreamSpec(e.secretsCache, cluster)
	if err != nil {
		return cluster, err
	}

	if !reflect.DeepEqual(cluster.Status.GKEStatus.UpstreamSpec, upstreamSpec) {
		logrus.Infof("updating cluster [%s], upstream change detected", cluster.Name)
		cluster = cluster.DeepCopy()
		cluster.Status.GKEStatus.UpstreamSpec = upstreamSpec
		cluster, err = e.clusterClient.Update(cluster)
		if err != nil {
			return cluster, err
		}
	}

	if !reflect.DeepEqual(cluster.Spec.GKEConfig, cluster.Status.AppliedSpec.GKEConfig) {
		logrus.Infof("cluster [%s] currently updating, skipping spec sync", cluster.Name)
		return cluster, nil
	}

	// check for changes between GKE spec on cluster and the GKE spec on the GKEClusterConfig object

	specMap, err := runtime.DefaultUnstructuredConverter.ToUnstructured(cluster.Spec.GKEConfig)
	if err != nil {
		return cluster, err
	}

	upstreamSpecMap, err := runtime.DefaultUnstructuredConverter.ToUnstructured(upstreamSpec)
	if err != nil {
		return cluster, err
	}

	var updateGKEConfig bool
	for key, value := range upstreamSpecMap {
		if specMap[key] == nil {
			continue
		}
		if reflect.DeepEqual(specMap[key], value) {
			continue
		}
		updateGKEConfig = true
		specMap[key] = value
	}

	if !updateGKEConfig {
		logrus.Infof("cluster [%s] matches upstream, skipping spec sync", cluster.Name)
		return cluster, nil
	}

	if err = runtime.DefaultUnstructuredConverter.FromUnstructured(specMap, cluster.Spec.GKEConfig); err != nil {
		return cluster, err
	}

	return e.clusterClient.Update(cluster)
}

func GetComparableUpstreamSpec(secretsCache wranglerv1.SecretCache, cluster *mgmtv3.Cluster) (*v1.GKEClusterConfigSpec, error) {
	ctx := context.Background()
	upstreamSpec, err := controller.buildUpstreamClusterState(ctx, secretsCache, *cluster.Spec.GKEConfig)
	if err != nil {
		return nil, err
	}

	upstreamSpec.DisplayName = cluster.Spec.GKEConfig.ClusterName
	upstreamSpec.ResourceLocation = cluster.Spec.GKEConfig.ResourceLocation
	upstreamSpec.AzureCredentialSecret = cluster.Spec.GKEConfig.AzureCredentialSecret
	upstreamSpec.Imported = cluster.Spec.GKEConfig.Imported

	return upstreamSpec, nil
}
