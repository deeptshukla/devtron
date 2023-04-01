/*
 * Copyright (c) 2020 Devtron Labs
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *    http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 *
 */

package app

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/devtron-labs/devtron/internal/middleware"
	"github.com/devtron-labs/devtron/internal/sql/repository/app"
	chartRepoRepository "github.com/devtron-labs/devtron/pkg/chartRepo/repository"
	repository2 "github.com/devtron-labs/devtron/pkg/cluster/repository"
	"github.com/devtron-labs/devtron/pkg/dockerRegistry"
	"github.com/devtron-labs/devtron/util/argo"
	errors2 "github.com/juju/errors"
	"go.opentelemetry.io/otel"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/argoproj/argo-cd/v2/pkg/apiclient/application"
	"github.com/devtron-labs/devtron/api/bean"
	application2 "github.com/devtron-labs/devtron/client/argocdServer/application"
	"github.com/devtron-labs/devtron/internal/constants"
	"github.com/devtron-labs/devtron/internal/sql/models"
	"github.com/devtron-labs/devtron/internal/sql/repository"
	"github.com/devtron-labs/devtron/internal/sql/repository/chartConfig"
	"github.com/devtron-labs/devtron/internal/sql/repository/helper"
	"github.com/devtron-labs/devtron/internal/sql/repository/pipelineConfig"
	"github.com/devtron-labs/devtron/internal/util"
	"github.com/devtron-labs/devtron/pkg/prometheus"
	"github.com/go-pg/pg"
	"github.com/pkg/errors"
	v1 "github.com/prometheus/client_golang/api/prometheus/v1"
	"go.uber.org/zap"
)

type AppListingService interface {
	FetchAppsByEnvironment(fetchAppListingRequest FetchAppListingRequest, w http.ResponseWriter, r *http.Request, token string) ([]*bean.AppEnvironmentContainer, int, error)
	FetchJobs(fetchJobListingRequest FetchAppListingRequest) ([]*bean.JobContainer, error)
	FetchOverviewCiPipelines(jobId int) ([]*bean.JobListingContainer, error)
	BuildAppListingResponse(fetchAppListingRequest FetchAppListingRequest, envContainers []*bean.AppEnvironmentContainer) ([]*bean.AppContainer, error)
	FetchAllDevtronManagedApps() ([]AppNameTypeIdContainer, error)
	FetchAppDetails(ctx context.Context, appId int, envId int) (bean.AppDetailContainer, error)

	PodCountByAppLabel(appLabel string, namespace string, env string, proEndpoint string) int
	PodListByAppLabel(appLabel string, namespace string, env string, proEndpoint string) map[string]string

	// below 4 functions used for pod level cpu and memory usage
	CpuUsageGroupByPod(namespace string, env string, proEndpoint string) map[string]string
	CpuRequestGroupByPod(namespace string, env string, proEndpoint string) map[string]string
	MemoryUsageGroupByPod(namespace string, env string, proEndpoint string) map[string]string
	MemoryRequestGroupByPod(namespace string, env string, proEndpoint string) map[string]string

	//Currently not in use
	CpuUsageGroupByContainer(podName string, namespace string, env string, proEndpoint string) map[string]string
	CpuRequestGroupByContainer(podName string, namespace string, env string, proEndpoint string) map[string]string
	MemoryUsageGroupByContainer(podName string, namespace string, env string, proEndpoint string) map[string]string
	MemoryRequestGroupByContainer(podName string, namespace string, env string, proEndpoint string) map[string]string

	//Currently not in use (intent to fetch graph data from prometheus)
	CpuUsageGroupByPodGraph(podName string, namespace string, env string, proEndpoint string, r v1.Range) map[string][]interface{}
	MemoryUsageGroupByPodGraph(podName string, namespace string, env string, proEndpoint string, r v1.Range) map[string][]interface{}
	GraphAPI(appId int, envId int) error

	FetchAppTriggerView(appId int) ([]bean.TriggerView, error)
	FetchAppStageStatus(appId int, appType int) ([]bean.AppStageStatus, error)

	FetchOtherEnvironment(ctx context.Context, appId int) ([]*bean.Environment, error)
	RedirectToLinkouts(Id int, appId int, envId int, podName string, containerName string) (string, error)
	ISLastReleaseStopType(appId, envId int) (bool, error)
	ISLastReleaseStopTypeV2(pipelineIds []int) (map[int]bool, error)
	GetReleaseCount(appId, envId int) (int, error)
}

const (
	Initiate              string = "Initiate"
	ScalingReplicaSetDown string = "ScalingReplicaSetDown"
)

type FetchAppListingRequest struct {
	Environments      []int            `json:"environments"`
	Statuses          []string         `json:"statuses"`
	Teams             []int            `json:"teams"`
	AppNameSearch     string           `json:"appNameSearch"`
	SortOrder         helper.SortOrder `json:"sortOrder"`
	SortBy            helper.SortBy    `json:"sortBy"`
	Offset            int              `json:"offset"`
	Size              int              `json:"size"`
	DeploymentGroupId int              `json:"deploymentGroupId"`
	Namespaces        []string         `json:"namespaces"` //{clusterId}_{namespace}
	AppStatuses       []string         `json:"appStatuses"`
	AppIds            []int            `json:"-"` //internal use only
}
type AppNameTypeIdContainer struct {
	AppName string `json:"appName"`
	Type    string `json:"type"`
	AppId   int    `json:"appId"`
}

func (req FetchAppListingRequest) GetNamespaceClusterMapping() (namespaceClusterPair []*repository2.ClusterNamespacePair, clusterIds []int, err error) {
	for _, ns := range req.Namespaces {
		items := strings.Split(ns, "_")
		if len(items) < 1 && len(items) > 2 {
			return nil, nil, fmt.Errorf("invalid namespaceds")
		}
		clusterId, err := strconv.Atoi(items[0])
		if err != nil {
			return nil, nil, fmt.Errorf("invalid clustrer id")
		}
		if len(items) == 2 {
			pair := &repository2.ClusterNamespacePair{
				ClusterId:     clusterId,
				NamespaceName: items[1],
			}
			namespaceClusterPair = append(namespaceClusterPair, pair)

		} else {
			clusterIds = append(clusterIds, clusterId)
		}
	}
	return namespaceClusterPair, clusterIds, nil
}

type AppListingServiceImpl struct {
	Logger                         *zap.SugaredLogger
	application                    application2.ServiceClient
	appRepository                  app.AppRepository
	appListingRepository           repository.AppListingRepository
	appListingViewBuilder          AppListingViewBuilder
	pipelineRepository             pipelineConfig.PipelineRepository
	cdWorkflowRepository           pipelineConfig.CdWorkflowRepository
	linkoutsRepository             repository.LinkoutsRepository
	appLevelMetricsRepository      repository.AppLevelMetricsRepository
	envLevelMetricsRepository      repository.EnvLevelAppMetricsRepository
	pipelineOverrideRepository     chartConfig.PipelineOverrideRepository
	environmentRepository          repository2.EnvironmentRepository
	argoUserService                argo.ArgoUserService
	envOverrideRepository          chartConfig.EnvConfigOverrideRepository
	chartRepository                chartRepoRepository.ChartRepository
	ciPipelineRepository           pipelineConfig.CiPipelineRepository
	dockerRegistryIpsConfigService dockerRegistry.DockerRegistryIpsConfigService
}

func NewAppListingServiceImpl(Logger *zap.SugaredLogger, appListingRepository repository.AppListingRepository,
	application application2.ServiceClient, appRepository app.AppRepository,
	appListingViewBuilder AppListingViewBuilder, pipelineRepository pipelineConfig.PipelineRepository,
	linkoutsRepository repository.LinkoutsRepository, appLevelMetricsRepository repository.AppLevelMetricsRepository,
	envLevelMetricsRepository repository.EnvLevelAppMetricsRepository, cdWorkflowRepository pipelineConfig.CdWorkflowRepository,
	pipelineOverrideRepository chartConfig.PipelineOverrideRepository, environmentRepository repository2.EnvironmentRepository,
	argoUserService argo.ArgoUserService, envOverrideRepository chartConfig.EnvConfigOverrideRepository,
	chartRepository chartRepoRepository.ChartRepository, ciPipelineRepository pipelineConfig.CiPipelineRepository,
	dockerRegistryIpsConfigService dockerRegistry.DockerRegistryIpsConfigService) *AppListingServiceImpl {
	serviceImpl := &AppListingServiceImpl{
		Logger:                         Logger,
		appListingRepository:           appListingRepository,
		application:                    application,
		appRepository:                  appRepository,
		appListingViewBuilder:          appListingViewBuilder,
		pipelineRepository:             pipelineRepository,
		linkoutsRepository:             linkoutsRepository,
		appLevelMetricsRepository:      appLevelMetricsRepository,
		envLevelMetricsRepository:      envLevelMetricsRepository,
		cdWorkflowRepository:           cdWorkflowRepository,
		pipelineOverrideRepository:     pipelineOverrideRepository,
		environmentRepository:          environmentRepository,
		argoUserService:                argoUserService,
		envOverrideRepository:          envOverrideRepository,
		chartRepository:                chartRepository,
		ciPipelineRepository:           ciPipelineRepository,
		dockerRegistryIpsConfigService: dockerRegistryIpsConfigService,
	}
	return serviceImpl
}

const AcdInvalidAppErr = "invalid acd app name and env"
const NotDeployed = "Not Deployed"

func (impl AppListingServiceImpl) FetchAllDevtronManagedApps() ([]AppNameTypeIdContainer, error) {
	impl.Logger.Debug("reached at FetchAllDevtronManagedApps:")
	apps := make([]AppNameTypeIdContainer, 0)
	res, err := impl.appRepository.FetchAllActiveDevtronAppsWithAppIdAndName()
	if err != nil {
		impl.Logger.Errorw("failed to fetch devtron apps", "err", err)
		return nil, err
	}
	for _, r := range res {
		appContainer := AppNameTypeIdContainer{
			AppId:   r.Id,
			AppName: r.AppName,
			Type:    "devtron-app",
		}
		apps = append(apps, appContainer)
	}
	res, err = impl.appRepository.FetchAllActiveInstalledAppsWithAppIdAndName()
	if err != nil {
		impl.Logger.Errorw("failed to fetch devtron installed apps", "err", err)
		return nil, err
	}
	for _, r := range res {
		appContainer := AppNameTypeIdContainer{
			AppId:   r.Id,
			AppName: r.AppName,
			Type:    "devtron-installed-app",
		}
		apps = append(apps, appContainer)
	}
	return apps, nil
}
func (impl AppListingServiceImpl) FetchJobs(fetchJobListingRequest FetchAppListingRequest) ([]*bean.JobContainer, error) {

	jobListingFilter := helper.AppListingFilter{
		Teams:         fetchJobListingRequest.Teams,
		AppNameSearch: fetchJobListingRequest.AppNameSearch,
		SortOrder:     fetchJobListingRequest.SortOrder,
		SortBy:        fetchJobListingRequest.SortBy,
		Offset:        fetchJobListingRequest.Offset,
		Size:          fetchJobListingRequest.Size,
		AppStatuses:   fetchJobListingRequest.AppStatuses,
	}
	appIds, err := impl.appRepository.FetchAppIdsWithFilter(jobListingFilter)
	if err != nil {
		impl.Logger.Errorw("error in fetching app ids list", "error", err, jobListingFilter)
		return []*bean.JobContainer{}, err
	}
	jobListingContainers, err := impl.appListingRepository.FetchJobs(appIds, jobListingFilter.AppStatuses, string(jobListingFilter.SortOrder))
	if err != nil {
		impl.Logger.Errorw("error in fetching job list", "error", err, jobListingFilter)
		return []*bean.JobContainer{}, err
	}
	CiPipelineIDs := GetCIPipelineIDs(jobListingContainers)
	JobsLastSucceededOnTime, err := impl.appListingRepository.FetchJobsLastSucceededOn(CiPipelineIDs)
	jobContainers := BuildJobListingResponse(jobListingContainers, JobsLastSucceededOnTime)
	return jobContainers, nil
}

func (impl AppListingServiceImpl) FetchOverviewCiPipelines(jobId int) ([]*bean.JobListingContainer, error) {
	jobCiContainers, err := impl.appListingRepository.FetchOverviewCiPipelines(jobId)
	if err != nil {
		impl.Logger.Errorw("error in fetching job container", "error", err, jobId)
		return []*bean.JobListingContainer{}, err
	}
	return jobCiContainers, nil
}

func (impl AppListingServiceImpl) FetchAppsByEnvironment(fetchAppListingRequest FetchAppListingRequest, w http.ResponseWriter, r *http.Request, token string) ([]*bean.AppEnvironmentContainer, int, error) {
	impl.Logger.Debug("reached at FetchAppsByEnvironment:")
	appListingFilter := helper.AppListingFilter{
		Environments:      fetchAppListingRequest.Environments,
		Statuses:          fetchAppListingRequest.Statuses,
		Teams:             fetchAppListingRequest.Teams,
		AppNameSearch:     fetchAppListingRequest.AppNameSearch,
		SortOrder:         fetchAppListingRequest.SortOrder,
		SortBy:            fetchAppListingRequest.SortBy,
		Offset:            fetchAppListingRequest.Offset,
		Size:              fetchAppListingRequest.Size,
		DeploymentGroupId: fetchAppListingRequest.DeploymentGroupId,
		AppStatuses:       fetchAppListingRequest.AppStatuses,
		AppIds:            fetchAppListingRequest.AppIds,
	}
	_, span := otel.Tracer("appListingRepository").Start(r.Context(), "FetchAppsByEnvironment")
	envContainers, appSize, err := impl.appListingRepository.FetchAppsByEnvironment(appListingFilter)
	span.End()
	if err != nil {
		impl.Logger.Errorw("error in fetching app list", "error", err)
		return []*bean.AppEnvironmentContainer{}, appSize, err
	}
	return envContainers, appSize, nil
}

func (impl AppListingServiceImpl) ISLastReleaseStopType(appId, envId int) (bool, error) {
	override, err := impl.pipelineOverrideRepository.GetLatestRelease(appId, envId)
	if err != nil && !util.IsErrNoRows(err) {
		impl.Logger.Errorw("error in getting last release")
		return false, err
	} else if util.IsErrNoRows(err) {
		return false, nil
	} else {
		return models.DEPLOYMENTTYPE_STOP == override.DeploymentType, nil
	}
}

func (impl AppListingServiceImpl) ISLastReleaseStopTypeV2(pipelineIds []int) (map[int]bool, error) {
	releaseMap := make(map[int]bool)
	if len(pipelineIds) == 0 {
		return releaseMap, nil
	}
	overrides, err := impl.pipelineOverrideRepository.GetLatestReleaseDeploymentType(pipelineIds)
	if err != nil && !util.IsErrNoRows(err) {
		impl.Logger.Errorw("error in getting last release")
		return releaseMap, err
	} else if util.IsErrNoRows(err) {
		return releaseMap, nil
	}
	for _, override := range overrides {
		if _, ok := releaseMap[override.PipelineId]; !ok {
			isStopType := models.DEPLOYMENTTYPE_STOP == override.DeploymentType
			releaseMap[override.PipelineId] = isStopType
		}
	}
	return releaseMap, nil
}

func (impl AppListingServiceImpl) GetReleaseCount(appId, envId int) (int, error) {
	override, err := impl.pipelineOverrideRepository.GetAllRelease(appId, envId)
	if err != nil && !util.IsErrNoRows(err) {
		impl.Logger.Errorw("error in getting releases")
		return 0, err
	} else if util.IsErrNoRows(err) {
		return 0, nil
	} else {
		return len(override), nil
	}
}

func (impl AppListingServiceImpl) BuildAppListingResponse(fetchAppListingRequest FetchAppListingRequest, envContainers []*bean.AppEnvironmentContainer) ([]*bean.AppContainer, error) {
	start := time.Now()
	appEnvMapping, err := impl.fetchACDAppStatus(fetchAppListingRequest, envContainers)
	middleware.AppListingDuration.WithLabelValues("fetchACDAppStatus", "devtron").Observe(time.Since(start).Seconds())
	if err != nil {
		impl.Logger.Errorw("error in fetching app statuses", "error", err)
		return []*bean.AppContainer{}, err
	}
	start = time.Now()
	appContainerResponses, err := impl.appListingViewBuilder.BuildView(fetchAppListingRequest, appEnvMapping)
	middleware.AppListingDuration.WithLabelValues("buildView", "devtron").Observe(time.Since(start).Seconds())
	return appContainerResponses, err
}
func GetCIPipelineIDs(jobContainers []*bean.JobListingContainer) []int {

	var ciPipelineIDs []int
	for _, jobContainer := range jobContainers {
		ciPipelineIDs = append(ciPipelineIDs, jobContainer.CiPipelineID)
	}
	return ciPipelineIDs
}
func BuildJobListingResponse(jobContainers []*bean.JobListingContainer, JobsLastSucceededOnTime []*bean.CiPipelineLastSucceededTime) []*bean.JobContainer {
	jobContainersMapping := make(map[int]bean.JobContainer)
	var appIds []int

	lastSucceededTimeMapping := make(map[int]time.Time)
	for _, lastSuccessTime := range JobsLastSucceededOnTime {
		lastSucceededTimeMapping[lastSuccessTime.CiPipelineID] = lastSuccessTime.LastSucceededOn
	}

	//Storing the sequence in appIds array
	for _, jobContainer := range jobContainers {
		val, ok := jobContainersMapping[jobContainer.JobId]
		if !ok {
			appIds = append(appIds, jobContainer.JobId)
			val = bean.JobContainer{}
			val.JobId = jobContainer.JobId
			val.JobName = jobContainer.JobName
			val.Description = jobContainer.Description
		}

		if len(val.JobCiPipelines) == 0 {
			val.JobCiPipelines = make([]bean.JobCIPipeline, 0)
		}

		if jobContainer.CiPipelineID != 0 {
			ciPipelineObj := bean.JobCIPipeline{
				CiPipelineId:   jobContainer.CiPipelineID,
				CiPipelineName: jobContainer.CiPipelineName,
				Status:         jobContainer.Status,
				LastRunAt:      jobContainer.StartedOn,
				//LastSuccessAt: jobContainer.LastSuccessAt,
			}
			if lastSuccessAt, ok := lastSucceededTimeMapping[jobContainer.CiPipelineID]; ok {
				ciPipelineObj.LastSuccessAt = lastSuccessAt
			}

			val.JobCiPipelines = append(val.JobCiPipelines, ciPipelineObj)
		}
		jobContainersMapping[jobContainer.JobId] = val

	}

	result := make([]*bean.JobContainer, 0)
	for _, appId := range appIds {
		val := jobContainersMapping[appId]
		result = append(result, &val)
	}

	return result
}

func (impl AppListingServiceImpl) fetchACDAppStatus(fetchAppListingRequest FetchAppListingRequest, existingAppEnvContainers []*bean.AppEnvironmentContainer) (map[string][]*bean.AppEnvironmentContainer, error) {
	appEnvMapping := make(map[string][]*bean.AppEnvironmentContainer)
	for _, env := range existingAppEnvContainers {
		appKey := strconv.Itoa(env.AppId) + "_" + env.AppName
		appEnvMapping[appKey] = append(appEnvMapping[appKey], env)
	}
	return appEnvMapping, nil
}

func (impl AppListingServiceImpl) getAppACDStatus(env bean.AppEnvironmentContainer, w http.ResponseWriter, r *http.Request, token string) (string, error) {
	//not being used  now
	if len(env.AppName) > 0 && len(env.EnvironmentName) > 0 {
		acdAppName := env.AppName + "-" + env.EnvironmentName
		query := &application.ResourcesQuery{
			ApplicationName: &acdAppName,
		}
		ctx, cancel := context.WithCancel(r.Context())
		if cn, ok := w.(http.CloseNotifier); ok {
			go func(done <-chan struct{}, closed <-chan bool) {
				select {
				case <-done:
				case <-closed:
					cancel()
				}
			}(ctx.Done(), cn.CloseNotify())
		}
		defer cancel()
		acdToken, err := impl.argoUserService.GetLatestDevtronArgoCdUserToken()
		if err != nil {
			impl.Logger.Errorw("error in getting acd token", "err", err)
			return "", err
		}
		ctx = context.WithValue(ctx, "token", acdToken)
		impl.Logger.Debugf("Getting status for app %s in env %s", env.AppId, env.EnvironmentId)
		start := time.Now()
		resp, err := impl.application.ResourceTree(ctx, query)
		elapsed := time.Since(start)
		impl.Logger.Debugf("Time elapsed %s in fetching application %s for environment %s", elapsed, env.AppId, env.EnvironmentId)
		if err != nil {
			impl.Logger.Errorw("error fetching resource tree", "error", err)
			err = &util.ApiError{
				Code:            constants.AppDetailResourceTreeNotFound,
				InternalMessage: "app detail fetched, failed to get resource tree from acd",
				UserMessage:     "app detail fetched, failed to get resource tree from acd",
			}
			return "", err
		}
		return resp.Status, nil
	}
	impl.Logger.Error("invalid acd app name and env ", env.AppName, " - ", env.EnvironmentName)
	return "", errors.New(AcdInvalidAppErr)
}

// TODO: Status mapping
func (impl AppListingServiceImpl) adaptStatusForView(status string) string {
	return status
}

func (impl AppListingServiceImpl) FetchAppDetails(ctx context.Context, appId int, envId int) (bean.AppDetailContainer, error) {
	appDetailContainer, err := impl.appListingRepository.FetchAppDetail(ctx, appId, envId)
	if err != nil {
		impl.Logger.Errorw("error in fetching app detail", "error", err)
		return bean.AppDetailContainer{}, err
	}

	appDetailContainer, err = impl.updateEnvMetricsData(ctx, appId, appDetailContainer)
	if err != nil {
		return appDetailContainer, err
	}

	_, span := otel.Tracer("orchestrator").Start(ctx, "linkoutsRepository.FetchLinkoutsByAppIdAndEnvId")
	linkoutsModel, err := impl.linkoutsRepository.FetchLinkoutsByAppIdAndEnvId(appId, envId)
	span.End()
	if err != nil && err != pg.ErrNoRows {
		impl.Logger.Errorw("error in fetching linkouts", "error", err)
		return bean.AppDetailContainer{}, err
	}
	var linkouts []bean.LinkOuts
	for _, linkout := range linkoutsModel {
		linkouts = append(linkouts, bean.LinkOuts{Id: linkout.Id, Name: linkout.Name})
	}

	appDetailContainer.LinkOuts = linkouts
	appDetailContainer.AppId = appId

	_, span = otel.Tracer("orchestrator").Start(ctx, "environmentRepository.FindById")
	envModel, err := impl.environmentRepository.FindById(envId)
	span.End()
	if err != nil {
		impl.Logger.Errorw("error in fetching environment", "error", err)
		return bean.AppDetailContainer{}, err
	}
	clusterId := envModel.ClusterId
	appDetailContainer.K8sVersion = envModel.Cluster.K8sVersion
	appDetailContainer.ClusterId = clusterId
	appDetailContainer.ClusterName = envModel.Cluster.ClusterName

	// set ifIpsAccess provided and relevant data
	appDetailContainer.IsExternalCi = true
	appDetailContainer, err = impl.setIpAccessProvidedData(ctx, appDetailContainer, clusterId)
	if err != nil {
		return appDetailContainer, err
	}

	return appDetailContainer, nil
}

func (impl AppListingServiceImpl) updateEnvMetricsData(ctx context.Context, appId int, appDetailContainer bean.AppDetailContainer) (bean.AppDetailContainer, error) {
	var appMetrics bool
	var infraMetrics bool
	_, span := otel.Tracer("orchestrator").Start(ctx, "appLevelMetricsRepository.FindByAppId")
	appLevelMetrics, err := impl.appLevelMetricsRepository.FindByAppId(appId)
	span.End()
	if err != nil && err != pg.ErrNoRows {
		impl.Logger.Errorw("error in app metrics app level flag", "error", err)
		return bean.AppDetailContainer{}, err
	} else if appLevelMetrics != nil {
		appMetrics = appLevelMetrics.AppMetrics
		infraMetrics = appLevelMetrics.InfraMetrics
	}
	i := 0
	for _, env := range appDetailContainer.Environments {
		var envLevelMetrics *bool
		var envLevelInfraMetrics *bool
		_, span := otel.Tracer("orchestrator").Start(ctx, "appLevelMetricsRepository.FindByAppIdAndEnvId")
		envLevelAppMetrics, err := impl.envLevelMetricsRepository.FindByAppIdAndEnvId(appId, env.EnvironmentId)
		span.End()
		if err != nil && err != pg.ErrNoRows {
			impl.Logger.Errorw("error in app metrics env level flag", "error", err)
			return bean.AppDetailContainer{}, err
		}

		if envLevelAppMetrics != nil && envLevelAppMetrics.Id != 0 && envLevelAppMetrics.AppMetrics != nil {
			envLevelMetrics = envLevelAppMetrics.AppMetrics
		} else {
			envLevelMetrics = &appMetrics
		}
		if envLevelAppMetrics != nil && envLevelAppMetrics.Id != 0 && envLevelAppMetrics.InfraMetrics != nil {
			envLevelInfraMetrics = envLevelAppMetrics.InfraMetrics
		} else {
			envLevelInfraMetrics = &infraMetrics
		}
		appDetailContainer.Environments[i].AppMetrics = envLevelMetrics
		appDetailContainer.Environments[i].InfraMetrics = envLevelInfraMetrics
		i++
	}
	return appDetailContainer, nil
}

func (impl AppListingServiceImpl) setIpAccessProvidedData(ctx context.Context, appDetailContainer bean.AppDetailContainer, clusterId int) (bean.AppDetailContainer, error) {
	ciPipelineId := appDetailContainer.CiPipelineId
	if ciPipelineId > 0 {
		_, span := otel.Tracer("orchestrator").Start(ctx, "ciPipelineRepository.FindById")
		ciPipeline, err := impl.ciPipelineRepository.FindById(ciPipelineId)
		span.End()
		if err != nil && err != pg.ErrNoRows {
			impl.Logger.Errorw("error in fetching ciPipeline", "ciPipelineId", ciPipelineId, "error", err)
			return bean.AppDetailContainer{}, err
		}

		if ciPipeline != nil && ciPipeline.CiTemplate != nil && len(*ciPipeline.CiTemplate.DockerRegistryId) > 0 {
			dockerRegistryId := ciPipeline.CiTemplate.DockerRegistryId
			appDetailContainer.DockerRegistryId = *dockerRegistryId
			if !ciPipeline.IsExternal || ciPipeline.ParentCiPipeline != 0 {
				appDetailContainer.IsExternalCi = false
			}

			_, span = otel.Tracer("orchestrator").Start(ctx, "dockerRegistryIpsConfigService.IsImagePullSecretAccessProvided")
			// check ips access provided to this docker registry for that cluster
			ipsAccessProvided, err := impl.dockerRegistryIpsConfigService.IsImagePullSecretAccessProvided(*dockerRegistryId, clusterId)
			span.End()
			if err != nil {
				impl.Logger.Errorw("error in checking if docker registry ips access provided", "dockerRegistryId", dockerRegistryId, "clusterId", clusterId, "error", err)
				return bean.AppDetailContainer{}, err
			}
			appDetailContainer.IpsAccessProvided = ipsAccessProvided
		}
	}
	return appDetailContainer, nil
}

// Return only a integer value pod count, aggregated of all the pod inside a app
// (includes all the pods running different cd pipeline for same app)
func (impl AppListingServiceImpl) PodCountByAppLabel(appLabel string, namespace string, env string, proEndpoint string) int {
	if appLabel == "" || namespace == "" || proEndpoint == "" || env == "" {
		impl.Logger.Warnw("not a complete data found for prometheus call", "missing", "AppName or namespace or prometheus url or env")
		return 0
	}

	prometheusAPI, err := prometheus.ContextByEnv(env, proEndpoint)
	if err != nil {
		impl.Logger.Errorw("error in getting prometheus api client:", "error", err)
		return 0
	}

	podCountQuery := "count(kube_pod_labels{label_app='" + appLabel + "', namespace='" + namespace + "'})"
	out, _, err := prometheusAPI.Query(context.Background(), podCountQuery, time.Now())
	if err != nil {
		impl.Logger.Errorw("pod count query failed in prometheus:", "error", err)
		return 0
	}
	response := make(map[string]interface{})
	response["data"] = out
	resJson, err := json.Marshal(response)
	if err != nil {
		impl.Logger.Errorw("pod count data marshal failed:", "error", err)
		return 0
	}

	podCount := 0
	resultMap := make(map[string]interface{})
	err = json.Unmarshal([]byte(resJson), &resultMap)
	if err != nil {
		impl.Logger.Errorw("pod count data unmarshal failed: ", "error", err)
		return 0
	}
	for _, value := range resultMap {
		data := value.([]interface{})

		for _, item := range data {

			ito := item
			for k, v := range ito.(map[string]interface{}) {
				if k == "value" {
					vArr := v.([]interface{})
					//t := (vArr[1].(string))
					feetInt, err := strconv.Atoi(vArr[1].(string))
					if err != nil {
						feetInt = 0
						impl.Logger.Errorw("casting error", "err", err)
					}
					podCount = feetInt
				}
			}
		}
	}
	return podCount
}

// Returns map of running pod names
func (impl AppListingServiceImpl) PodListByAppLabel(appLabel string, namespace string, env string, proEndpoint string) map[string]string {
	response := make(map[string]interface{})
	podList := make(map[string]string)
	resultMap := make(map[string]interface{})
	if appLabel == "" || namespace == "" || proEndpoint == "" || env == "" {
		impl.Logger.Warnw("not a complete data found for prometheus call", "missing", "AppName or namespace or prometheus url or env")
		return podList
	}

	prometheusAPI, err := prometheus.ContextByEnv(env, proEndpoint)
	if err != nil {
		impl.Logger.Errorw("error in getting prometheus api client:", "error", err)
		return podList
	}

	podCountQuery := "kube_pod_labels{label_app='" + appLabel + "', namespace='" + namespace + "'}"
	out, _, err := prometheusAPI.Query(context.Background(), podCountQuery, time.Now())
	if err != nil {
		impl.Logger.Errorw("pod list query failed in prometheus:", "error", err)
		return podList
	}

	response["data"] = out
	resJson, err := json.Marshal(response)
	if err != nil {
		impl.Logger.Errorw("pod count data unmarshal failed:", "error", err)
		return podList
	}

	err = json.Unmarshal([]byte(resJson), &resultMap)
	if err != nil {
		impl.Logger.Errorw("pod count data unmarshal failed:", "error", err)
		return podList
	}
	for _, value := range resultMap {
		if value != nil {
			data := value.([]interface{})

			for _, item := range data {

				ito := item
				for k, v := range ito.(map[string]interface{}) {
					if k == "metric" {
						vMap := v.(map[string]interface{})
						key := vMap["pod"].(string)
						podList[key] = "1"
					}
					if k == "value" {
					}
				}
			}
		}
	}
	return podList
}

func (impl AppListingServiceImpl) CpuUsageGroupByPod(namespace string, env string, proEndpoint string) map[string]string {
	impl.Logger.Debug("executing cpuUsageGroupByPod:")
	cpuUsageMetric := make(map[string]string)

	if namespace == "" || proEndpoint == "" || env == "" {
		impl.Logger.Warnw("not a complete data found for prometheus call", "missing", "AppName or namespace or prometheus url or env")
		return cpuUsageMetric
	}

	prometheusAPI, err := prometheus.ContextByEnv(env, proEndpoint)
	if err != nil {
		impl.Logger.Errorw("error in getting prometheus api client:", "error", err)
		return cpuUsageMetric
	}

	query := "sum(rate (container_cpu_usage_seconds_total{image!='',pod_name!='',container_name!='POD',namespace='" + namespace + "'}[1m])) by (pod_name)"
	out, _, err := prometheusAPI.Query(context.Background(), query, time.Now())
	if err != nil {
		impl.Logger.Errorw("error in getting CpuUsageGroupByPod:", "error", err)
		return cpuUsageMetric
	}

	response := make(map[string]interface{})
	response["data"] = out
	resJson, err := json.Marshal(response)
	if err != nil {
		impl.Logger.Errorw("error in marshal CpuUsageGroupByPod:", "error", err)
		return cpuUsageMetric
	}

	resultMap := make(map[string]interface{})
	err = json.Unmarshal([]byte(resJson), &resultMap)
	if err != nil {
		impl.Logger.Errorw("error in unmarshal CpuUsageGroupByPod:", "error", err)
		return cpuUsageMetric
	}

	for _, value := range resultMap {
		data := value.([]interface{})
		for _, item := range data {
			ito := item
			temp := ""
			for k, v := range ito.(map[string]interface{}) {
				if k == "metric" {
					vMap := v.(map[string]interface{})
					key := vMap["pod_name"].(string)
					cpuUsageMetric[key] = "1.0"
					temp = key
				}
				if k == "value" {
					vArr := v.([]interface{})
					if _, ok := cpuUsageMetric[temp]; ok {
						cpuUsageMetric[temp] = vArr[1].(string)
					}
				}
			}
		}
	}
	return cpuUsageMetric
}

func (impl AppListingServiceImpl) CpuRequestGroupByPod(namespace string, env string, proEndpoint string) map[string]string {
	impl.Logger.Debug("executing cpuUsageGroupByPod:")
	cpuRequestMetric := make(map[string]string)

	if namespace == "" || proEndpoint == "" || env == "" {
		impl.Logger.Warnw("not a complete data found for prometheus call", "missing", "AppName or namespace or prometheus url or env")
		return cpuRequestMetric
	}

	prometheusAPI, err := prometheus.ContextByEnv(env, proEndpoint)
	if err != nil {
		impl.Logger.Errorw("error in getting prometheus api client:", "error", err)
		return cpuRequestMetric
	}

	query := "sum(kube_pod_container_resource_requests_cpu_cores{namespace='" + namespace + "'}) by (pod)"
	out, _, err := prometheusAPI.Query(context.Background(), query, time.Now())
	if err != nil {
		impl.Logger.Errorw("error in prometheus query:", "error", err)
		return cpuRequestMetric
	}

	response := make(map[string]interface{})
	response["data"] = out
	resJson, err := json.Marshal(response)
	if err != nil {
		impl.Logger.Errorw("error in marshal:", "error", err)
		return cpuRequestMetric
	}

	resultMap := make(map[string]interface{})
	err = json.Unmarshal([]byte(resJson), &resultMap)
	if err != nil {
		impl.Logger.Errorw("error in unmarshal:", "error", err)
		return cpuRequestMetric
	}
	for _, value := range resultMap {
		data := value.([]interface{})

		for _, item := range data {

			ito := item
			temp := ""
			for k, v := range ito.(map[string]interface{}) {
				if k == "metric" {
					vMap := v.(map[string]interface{})
					key := vMap["pod"].(string)
					cpuRequestMetric[key] = "1"
					temp = key
				}
				if k == "value" {
					vArr := v.([]interface{})
					if _, ok := cpuRequestMetric[temp]; ok {
						cpuRequestMetric[temp] = vArr[1].(string)
					}
				}
			}
		}
	}
	return cpuRequestMetric
}

func (impl AppListingServiceImpl) MemoryUsageGroupByPod(namespace string, env string, proEndpoint string) map[string]string {
	impl.Logger.Debug("executing memoryUsageGroupByPod")
	memoryUsageMetric := make(map[string]string)

	if namespace == "" || proEndpoint == "" || env == "" {
		impl.Logger.Warnw("not a complete data found for prometheus call", "missing", "AppName or namespace or prometheus url or env")
		return memoryUsageMetric
	}

	prometheusAPI, err := prometheus.ContextByEnv(env, proEndpoint)
	if err != nil {
		impl.Logger.Errorw("error in getting prometheus api client:", "error", err)
		return memoryUsageMetric
	}

	query := "sum(container_memory_usage_bytes{container_name!='POD', container_name!='', namespace='" + namespace + "'}) by (pod_name)"
	out, _, err := prometheusAPI.Query(context.Background(), query, time.Now())
	if err != nil {
		impl.Logger.Errorw("error in prometheus query:", "error", err)
		return memoryUsageMetric
	}
	response := make(map[string]interface{})
	response["data"] = out
	resJson, err := json.Marshal(response)
	if err != nil {
		impl.Logger.Errorw("error in marshal:", "error", err)
		return memoryUsageMetric
	}
	resultMap := make(map[string]interface{})
	err = json.Unmarshal([]byte(resJson), &resultMap)
	if err != nil {
		impl.Logger.Errorw("error in unmarshal:", "error", err)
		return memoryUsageMetric
	}
	for _, value := range resultMap {
		data := value.([]interface{})
		for _, item := range data {

			ito := item
			temp := ""
			for k, v := range ito.(map[string]interface{}) {
				if k == "metric" {
					vMap := v.(map[string]interface{})
					key := vMap["pod_name"].(string)
					memoryUsageMetric[key] = "1"
					temp = key
				}
				if k == "value" {
					vArr := v.([]interface{})
					if _, ok := memoryUsageMetric[temp]; ok {
						memoryUsageMetric[temp] = vArr[1].(string)
					}
				}
			}
		}
	}
	return memoryUsageMetric
}

func (impl AppListingServiceImpl) MemoryRequestGroupByPod(namespace string, env string, proEndpoint string) map[string]string {
	impl.Logger.Debug("executing memoryRequestGroupByPod")
	memoryRequestMetric := make(map[string]string)
	if namespace == "" || proEndpoint == "" || env == "" {
		impl.Logger.Warnw("not a complete data found for prometheus call", "missing", "AppName or namespace or prometheus url or env")
		return memoryRequestMetric
	}

	prometheusAPI, err := prometheus.ContextByEnv(env, proEndpoint)
	if err != nil {
		impl.Logger.Errorw("error in getting prometheus api client:", "error", err)
		return memoryRequestMetric
	}

	query := "sum(kube_pod_container_resource_requests_memory_bytes{container!='POD', container!='', namespace='" + namespace + "'}) by (pod)"
	out, _, err := prometheusAPI.Query(context.Background(), query, time.Now())
	if err != nil {
		impl.Logger.Errorw("error in prometheus query:", "error", err)
		return memoryRequestMetric
	}

	response := make(map[string]interface{})
	response["data"] = out
	resJson, err := json.Marshal(response)
	if err != nil {
		impl.Logger.Errorw("error in marshal:", "error", err)
		return memoryRequestMetric
	}

	resultMap := make(map[string]interface{})
	err = json.Unmarshal([]byte(resJson), &resultMap)
	if err != nil {
		impl.Logger.Errorw("error in unmarshal:", "error", err)
		return memoryRequestMetric
	}

	for _, value := range resultMap {
		data := value.([]interface{})
		for _, item := range data {
			ito := item
			temp := ""
			for k, v := range ito.(map[string]interface{}) {
				if k == "metric" {
					vMap := v.(map[string]interface{})
					key := vMap["pod"].(string)
					memoryRequestMetric[key] = "1"
					temp = key
				}
				if k == "value" {
					vArr := v.([]interface{})
					if _, ok := memoryRequestMetric[temp]; ok {
						memoryRequestMetric[temp] = vArr[1].(string)
					}
				}
			}
		}
	}
	return memoryRequestMetric
}

// Deprecated: Currently not in use
func (impl AppListingServiceImpl) CpuUsageGroupByContainer(podName string, namespace string, env string, proEndpoint string) map[string]string {
	impl.Logger.Debug("executing cpuUsageGroupByPod:")
	prometheusAPI, err := prometheus.ContextByEnv(env, proEndpoint)
	cpuUsageMetric := make(map[string]string)

	if err != nil {
		impl.Logger.Errorw("error in getting prometheus api client:", "error", err)
		return cpuUsageMetric
	}

	query := "sum(rate(container_cpu_usage_seconds_total{image!='', pod_name='" + podName + "',container_name!='POD', namespace='" + podName + "'}[1m])) by (container_name)"
	out, _, err := prometheusAPI.Query(context.Background(), query, time.Now())
	if err != nil {
		impl.Logger.Errorw("error in prometheus query:", "error", err)
		return cpuUsageMetric
	}
	response := make(map[string]interface{})
	response["data"] = out
	resJson, err := json.Marshal(response)
	if err != nil {
		impl.Logger.Errorw("error in marshal:", "error", err)
		return cpuUsageMetric
	}

	resultMap := make(map[string]interface{})
	err = json.Unmarshal([]byte(resJson), &resultMap)
	if err != nil {
		impl.Logger.Errorw("error in unmarshal:", "error", err)
		return cpuUsageMetric
	}

	for _, value := range resultMap {
		data := value.([]interface{})

		for _, item := range data {
			ito := item
			temp := ""
			for k, v := range ito.(map[string]interface{}) {
				if k == "metric" {
					vMap := v.(map[string]interface{})
					key := vMap["pod_name"].(string)
					cpuUsageMetric[key] = "1"
					temp = key
				}
				if k == "value" {
					vArr := v.([]interface{})
					if _, ok := cpuUsageMetric[temp]; ok {
						cpuUsageMetric[temp] = vArr[1].(string)
					}
				}
			}
		}
	}

	return cpuUsageMetric
}

// Deprecated: Currently not in use
func (impl AppListingServiceImpl) CpuRequestGroupByContainer(podName string, namespace string, env string, proEndpoint string) map[string]string {
	impl.Logger.Debug("executing cpuUsageGroupByPod:")
	prometheusAPI, err := prometheus.ContextByEnv(env, proEndpoint)
	cpuRequestMetric := make(map[string]string)

	if err != nil {
		impl.Logger.Errorw("error in getting prometheus api client:", "error", err)
		return cpuRequestMetric
	}

	query := "sum(kube_pod_container_resource_requests_cpu_cores{namespace='" + namespace + "',pod='" + podName + "'}) by (container)"
	out, _, err := prometheusAPI.Query(context.Background(), query, time.Now())
	if err != nil {
		impl.Logger.Errorw("error in prometheus query:", "error", err)
		return cpuRequestMetric
	}

	response := make(map[string]interface{})
	response["data"] = out
	resJson, err := json.Marshal(response)
	if err != nil {
		impl.Logger.Errorw("error in marshal:", "error", err)
		return cpuRequestMetric
	}

	resultMap := make(map[string]interface{})
	err = json.Unmarshal([]byte(resJson), &resultMap)
	if err != nil {
		impl.Logger.Errorw("error in unmarshal:", "error", err)
		return cpuRequestMetric
	}

	for _, value := range resultMap {
		data := value.([]interface{})
		for _, item := range data {
			ito := item
			temp := ""
			for k, v := range ito.(map[string]interface{}) {
				if k == "metric" {
					vMap := v.(map[string]interface{})
					key := vMap["pod"].(string)
					cpuRequestMetric[key] = "1"
					temp = key
				}
				if k == "value" {
					vArr := v.([]interface{})
					if _, ok := cpuRequestMetric[temp]; ok {
						cpuRequestMetric[temp] = vArr[1].(string)
					}
				}
			}
		}
	}
	return cpuRequestMetric
}

// Deprecated: Currently not in use
func (impl AppListingServiceImpl) MemoryUsageGroupByContainer(podName string, namespace string, env string, proEndpoint string) map[string]string {
	impl.Logger.Debug("executing memoryUsageGroupByPod")
	prometheusAPI, err := prometheus.ContextByEnv(env, proEndpoint)
	memoryUsageMetric := make(map[string]string)

	if err != nil {
		impl.Logger.Errorw("error in getting prometheus api client:", "error", err)
		return memoryUsageMetric
	}

	query := "sum(container_memory_usage_bytes{container_name!='POD', container_name!='',pod_name='" + podName + "', namespace='" + namespace + "'}) by (container_name)"
	out, _, err := prometheusAPI.Query(context.Background(), query, time.Now())
	if err != nil {
		impl.Logger.Errorw("error in prometheus query:", "error", err)
		return memoryUsageMetric
	}
	response := make(map[string]interface{})
	response["data"] = out
	resJson, err := json.Marshal(response)
	if err != nil {
		impl.Logger.Errorw("error in marshal:", "error", err)
		return memoryUsageMetric
	}

	resultMap := make(map[string]interface{})
	err = json.Unmarshal([]byte(resJson), &resultMap)
	if err != nil {
		impl.Logger.Errorw("error in unmarshal:", "error", err)
		return memoryUsageMetric
	}
	for _, value := range resultMap {
		data := value.([]interface{})
		for _, item := range data {
			ito := item
			temp := ""
			for k, v := range ito.(map[string]interface{}) {
				if k == "metric" {
					vMap := v.(map[string]interface{})
					key := vMap["pod_name"].(string)
					memoryUsageMetric[key] = "1"
					temp = key
				}
				if k == "value" {
					vArr := v.([]interface{})
					if _, ok := memoryUsageMetric[temp]; ok {
						memoryUsageMetric[temp] = vArr[1].(string)
					}
				}
			}
		}
	}
	return memoryUsageMetric
}

// Deprecated: Currently not in use
func (impl AppListingServiceImpl) MemoryRequestGroupByContainer(podName string, namespace string, env string, proEndpoint string) map[string]string {
	impl.Logger.Debug("executing memoryRequestGroupByPod")
	prometheusAPI, err := prometheus.ContextByEnv(env, proEndpoint)
	memoryRequestMetric := make(map[string]string)
	if err != nil {
		impl.Logger.Errorw("error in getting prometheus api client:", "error", err)
		return memoryRequestMetric
	}

	query := "sum(kube_pod_container_resource_requests_memory_bytes{container!='POD', container!='',pod='" + podName + "', namespace='" + namespace + "'}) by (container)"
	out, _, err := prometheusAPI.Query(context.Background(), query, time.Now())
	if err != nil {
		impl.Logger.Errorw("error in prometheus query:", "error", err)
		return memoryRequestMetric
	}

	response := make(map[string]interface{})
	response["data"] = out
	resJson, err := json.Marshal(response)
	if err != nil {
		impl.Logger.Errorw("error in marshal:", "error", err)
		return memoryRequestMetric
	}

	resultMap := make(map[string]interface{})
	err = json.Unmarshal([]byte(resJson), &resultMap)
	if err != nil {
		impl.Logger.Errorw("error in unmarshal:", "error", err)
		return memoryRequestMetric
	}
	for _, value := range resultMap {
		data := value.([]interface{})
		for _, item := range data {
			ito := item
			temp := ""
			for k, v := range ito.(map[string]interface{}) {
				if k == "metric" {
					vMap := v.(map[string]interface{})
					key := vMap["pod"].(string)
					memoryRequestMetric[key] = "1"
					temp = key
				}
				if k == "value" {
					vArr := v.([]interface{})
					if _, ok := memoryRequestMetric[temp]; ok {
						memoryRequestMetric[temp] = vArr[1].(string)
					}
				}
			}
		}
	}
	return memoryRequestMetric
}

// Deprecated: Currently not in use (intent to fetch graph data from prometheus)
func (impl AppListingServiceImpl) CpuUsageGroupByPodGraph(podName string, namespace string, env string, proEndpoint string, r v1.Range) map[string][]interface{} {
	impl.Logger.Debug("executing CpuUsageGroupByPodGraph:")
	prometheusAPI, err := prometheus.ContextByEnv(env, proEndpoint)
	cpuUsageMetric := make(map[string][]interface{})

	if err != nil {
		impl.Logger.Errorw("error in getting prometheus api client:", "error", err)
		return cpuUsageMetric
	}

	query := "sum(rate(container_cpu_usage_seconds_total{namespace='" + namespace + "', container_name!='POD'}[1m])) by (pod_name)"
	time1 := time.Now()
	r1 := v1.Range{
		Start: time1.Add(-time.Hour),
		End:   time1,
		Step:  time.Minute,
	}
	out, _, err := prometheusAPI.QueryRange(context.Background(), query, r1)
	if err != nil {
		impl.Logger.Errorw("error in prometheus query:", "error", err)
		return cpuUsageMetric
	}

	response := make(map[string]interface{})
	response["data"] = out
	resJson, err := json.Marshal(response)
	if err != nil {
		impl.Logger.Errorw("error in marshal:", "error", err)
		return cpuUsageMetric
	}
	resultMap := make(map[string]interface{})
	err = json.Unmarshal([]byte(resJson), &resultMap)
	if err != nil {
		impl.Logger.Errorw("error in unmarshal:", "error", err)
		return cpuUsageMetric
	}
	for _, value := range resultMap {
		data := value.([]interface{})
		for _, item := range data {
			ito := item
			temp := ""
			for k, v := range ito.(map[string]interface{}) {
				if k == "metric" {
					vMap := v.(map[string]interface{})
					key := vMap["pod_name"].(string)
					cpuUsageMetric[key] = nil
					temp = key
				}
				if k == "values" {
					vArr := v.([]interface{})
					if _, ok := cpuUsageMetric[temp]; ok {
						cpuUsageMetric[temp] = vArr
					}
				}
			}
		}
	}
	return cpuUsageMetric
}

// Deprecated: Currently not in use (intent to fetch graph data from prometheus)
func (impl AppListingServiceImpl) MemoryUsageGroupByPodGraph(podName string, namespace string, env string, proEndpoint string, r v1.Range) map[string][]interface{} {
	impl.Logger.Debug("executing MemoryUsageGroupByPodGraph")
	prometheusAPI, err := prometheus.ContextByEnv(env, proEndpoint)
	memoryUsageMetric := make(map[string][]interface{})

	if err != nil {
		impl.Logger.Errorw("error in getting prometheus api client:", "error", err)
		return memoryUsageMetric
	}

	query := "sum(container_memory_usage_bytes{namespace='" + namespace + "', container_name!='POD', container_name!=''}) by (pod_name)"
	time1 := time.Now()
	r1 := v1.Range{
		Start: time1.Add(-time.Hour),
		End:   time1,
		Step:  time.Minute,
	}
	out, _, err := prometheusAPI.QueryRange(context.Background(), query, r1)
	if err != nil {
		impl.Logger.Errorw("error in prometheus query:", "error", err)
		return memoryUsageMetric
	}
	response := make(map[string]interface{})
	response["data"] = out
	resJson, err := json.Marshal(response)
	if err != nil {
		impl.Logger.Errorw("error in marshal:", "error", err)
		return memoryUsageMetric
	}
	resultMap := make(map[string]interface{})
	err = json.Unmarshal([]byte(resJson), &resultMap)
	if err != nil {
		impl.Logger.Errorw("error in unmarshal:", "error", err)
		return memoryUsageMetric
	}
	for _, value := range resultMap {
		data := value.([]interface{})
		for _, item := range data {
			ito := item
			temp := ""
			for k, v := range ito.(map[string]interface{}) {
				if k == "metric" {
					vMap := v.(map[string]interface{})
					key := vMap["pod_name"].(string)
					memoryUsageMetric[key] = nil
					temp = key
				}
				if k == "values" {
					vArr := v.([]interface{})
					if _, ok := memoryUsageMetric[temp]; ok {
						memoryUsageMetric[temp] = vArr
					}
				}
			}
		}
	}
	return memoryUsageMetric
}

// Deprecated: Currently not in use (intent to fetch graph data from prometheus)
func (impl AppListingServiceImpl) GraphAPI(appId int, envId int) error {
	impl.Logger.Debug("reached at GraphAPI:")
	/*
		appDetailView, err := impl.appListingRepository.FetchAppDetail(appId, envId)
		if err != nil {
			impl.Logger.Errorw("Exception", err)
			return err
		}

		//calculating cpu and memory usage percent
		appLabel := appDetailView.AppName
		namespace := appDetailView.Namespace
		proEndpoint := appDetailView.PrometheusEndpoint
		env := appDetailView.EnvironmentName
		podList := impl.PodListByAppLabel(appLabel, namespace, env, proEndpoint)

		//TODO - Pod List By Label- Release

		time1 := time.Time{}
		r1 := v1.Range{
			Start: time1.Add(-time.Minute),
			End:   time1,
			Step:  time.Minute,
		}
		podName := "prometheus-monitoring-prometheus-oper-prometheus-0"
		impl.CpuUsageGroupByPodGraph(podName, namespace, env, proEndpoint, r1)
		//data := impl.MemoryUsageGroupByPodGraph(podName, "monitoring", env, proEndpoint, r1)

		for fKey, _ := range podList {
			fmt.Println(fKey)
		}
	*/
	return nil
}

func (impl AppListingServiceImpl) FetchAppTriggerView(appId int) ([]bean.TriggerView, error) {
	return impl.appListingRepository.FetchAppTriggerView(appId)
}

func (impl AppListingServiceImpl) FetchAppStageStatus(appId int, appType int) ([]bean.AppStageStatus, error) {
	appStageStatuses, err := impl.appListingRepository.FetchAppStageStatus(appId, appType)
	return appStageStatuses, err
}

func (impl AppListingServiceImpl) FetchOtherEnvironment(ctx context.Context, appId int) ([]*bean.Environment, error) {
	newCtx, span := otel.Tracer("appListingRepository").Start(ctx, "FetchOtherEnvironment")
	envs, err := impl.appListingRepository.FetchOtherEnvironment(appId)
	span.End()
	if err != nil && !util.IsErrNoRows(err) {
		impl.Logger.Errorw("err", err)
		return envs, err
	}
	appLevelAppMetrics := false  //default value
	appLevelInfraMetrics := true //default val
	newCtx, span = otel.Tracer("appLevelMetricsRepository").Start(newCtx, "FindByAppId")
	appLevelMetrics, err := impl.appLevelMetricsRepository.FindByAppId(appId)
	span.End()
	if err != nil && !util.IsErrNoRows(err) {
		impl.Logger.Errorw("error in fetching app metrics", "err", err)
		return envs, err
	} else if util.IsErrNoRows(err) {
		//populate default val
		appLevelAppMetrics = false  //default value
		appLevelInfraMetrics = true //default val
	} else {
		appLevelAppMetrics = appLevelMetrics.AppMetrics
		appLevelInfraMetrics = appLevelMetrics.InfraMetrics
	}
	newCtx, span = otel.Tracer("chartRepository").Start(newCtx, "FindLatestChartForAppByAppId")
	chart, err := impl.chartRepository.FindLatestChartForAppByAppId(appId)
	span.End()
	if err != nil && err != pg.ErrNoRows {
		impl.Logger.Errorw("error in fetching latest chart", "err", err)
		return envs, err
	}
	for _, env := range envs {
		newCtx, span = otel.Tracer("envOverrideRepository").Start(newCtx, "FindLatestChartForAppByAppIdAndEnvId")
		envOverride, err := impl.envOverrideRepository.FindLatestChartForAppByAppIdAndEnvId(appId, env.EnvironmentId)
		span.End()
		if err != nil && !errors2.IsNotFound(err) {
			impl.Logger.Errorw("error in fetching latest chart by appId and envId", "err", err, "appId", appId, "envId", env.EnvironmentId)
			return envs, err
		}
		if envOverride != nil && envOverride.Chart != nil {
			env.ChartRefId = envOverride.Chart.ChartRefId
		} else {
			env.ChartRefId = chart.ChartRefId
		}
		if env.AppMetrics == nil {
			env.AppMetrics = &appLevelAppMetrics
		}
		if env.InfraMetrics == nil {
			env.InfraMetrics = &appLevelInfraMetrics
		}
	}
	return envs, nil
}

func arrContains(s []string, e string) bool {
	for _, a := range s {
		if a == e {
			return true
		}
	}
	return false
}

func (impl AppListingServiceImpl) RedirectToLinkouts(Id int, appId int, envId int, podName string, containerName string) (string, error) {
	linkout, err := impl.linkoutsRepository.FetchLinkoutById(Id)
	if err != nil {
		impl.Logger.Errorw("Exception", err)
		return "", err
	}
	link := linkout.Link
	if len(podName) > 0 && len(containerName) > 0 {
		link = strings.ReplaceAll(link, "{appName}", linkout.AppName)
		link = strings.ReplaceAll(link, "{envName}", linkout.EnvName)
		link = strings.ReplaceAll(link, "{podName}", podName)
		link = strings.ReplaceAll(link, "{containerName}", containerName)
	} else if len(podName) > 0 {
		link = strings.ReplaceAll(link, "{appName}", linkout.AppName)
		link = strings.ReplaceAll(link, "{envName}", linkout.EnvName)
		link = strings.ReplaceAll(link, "{podName}", podName)
	} else if len(containerName) > 0 {
		link = strings.ReplaceAll(link, "{appName}", linkout.AppName)
		link = strings.ReplaceAll(link, "{envName}", linkout.EnvName)
		link = strings.ReplaceAll(link, "{containerName}", containerName)
	} else {
		link = strings.ReplaceAll(link, "{appName}", linkout.AppName)
		link = strings.ReplaceAll(link, "{envName}", linkout.EnvName)
	}

	return link, nil
}
