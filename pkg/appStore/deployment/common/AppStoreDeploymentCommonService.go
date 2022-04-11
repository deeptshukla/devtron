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

package appStoreDeploymentCommon

import (
	"context"
	openapi "github.com/devtron-labs/devtron/api/helm-app/openapiClient"
	appStoreBean "github.com/devtron-labs/devtron/pkg/appStore/bean"
	"github.com/devtron-labs/devtron/pkg/appStore/deployment/repository"
	appStoreDiscoverRepository "github.com/devtron-labs/devtron/pkg/appStore/discover/repository"
	"github.com/go-pg/pg"
	"go.uber.org/zap"
	"time"
)

type AppStoreDeploymentCommonService interface {
	GetInstalledAppByClusterNamespaceAndName(clusterId int, namespace string, appName string) (*appStoreBean.InstallAppVersionDTO, error)
	GetInstalledAppByInstalledAppId(installedAppId int) (*appStoreBean.InstallAppVersionDTO, error)
	UpdateApplicationLinkedWithHelm(ctx context.Context, request *openapi.UpdateReleaseRequest) error
}

type AppStoreDeploymentCommonServiceImpl struct {
	logger                               *zap.SugaredLogger
	installedAppRepository               repository.InstalledAppRepository
	appStoreApplicationVersionRepository appStoreDiscoverRepository.AppStoreApplicationVersionRepository
}

func NewAppStoreDeploymentCommonServiceImpl(logger *zap.SugaredLogger, installedAppRepository repository.InstalledAppRepository,
	appStoreApplicationVersionRepository appStoreDiscoverRepository.AppStoreApplicationVersionRepository) *AppStoreDeploymentCommonServiceImpl {
	return &AppStoreDeploymentCommonServiceImpl{
		logger:                               logger,
		installedAppRepository:               installedAppRepository,
		appStoreApplicationVersionRepository: appStoreApplicationVersionRepository,
	}
}

func (impl AppStoreDeploymentCommonServiceImpl) GetInstalledAppByClusterNamespaceAndName(clusterId int, namespace string, appName string) (*appStoreBean.InstallAppVersionDTO, error) {
	installedApp, err := impl.installedAppRepository.GetInstalledApplicationByClusterIdAndNamespaceAndAppName(clusterId, namespace, appName)
	if err != nil {
		if err == pg.ErrNoRows {
			impl.logger.Warnw("no installed apps found", "clusterId", clusterId)
			return nil, nil
		} else {
			impl.logger.Errorw("error while fetching installed apps", "clusterId", clusterId, "error", err)
			return nil, err
		}
	}

	if installedApp.Id > 0 {
		installedAppVersion, err := impl.installedAppRepository.GetInstalledAppVersionByInstalledAppIdAndEnvId(installedApp.Id, installedApp.EnvironmentId)
		if err != nil {
			return nil, err
		}
		return impl.convert(installedApp, installedAppVersion), nil
	}

	return nil, nil
}

func (impl AppStoreDeploymentCommonServiceImpl) GetInstalledAppByInstalledAppId(installedAppId int) (*appStoreBean.InstallAppVersionDTO, error) {
	installedAppVersion, err := impl.installedAppRepository.GetActiveInstalledAppVersionByInstalledAppId(installedAppId)
	if err != nil {
		return nil, err
	}
	installedApp := &installedAppVersion.InstalledApp
	return impl.convert(installedApp, installedAppVersion), nil

	return nil, nil
}

func (impl AppStoreDeploymentCommonServiceImpl) UpdateApplicationLinkedWithHelm(ctx context.Context, request *openapi.UpdateReleaseRequest) error {

	dbConnection := impl.installedAppRepository.GetConnection()
	tx, err := dbConnection.Begin()
	if err != nil {
		return err
	}
	// Rollback tx on error.
	defer tx.Rollback()
	// update same chart or upgrade its version only
	installedAppVersionModel, err := impl.installedAppRepository.GetInstalledAppVersion(request.InstalledAppVersionId)
	if err != nil {
		impl.logger.Errorw("error while fetching chart installed version", "error", err)
		return err
	}
	var installedAppVersion *repository.InstalledAppVersions
	if installedAppVersionModel.AppStoreApplicationVersionId != request.AppStoreVersion {
		// upgrade to new version of same chart
		installedAppVersionModel.Active = false
		installedAppVersionModel.UpdatedOn = time.Now()
		installedAppVersionModel.UpdatedBy = int32(request.UserId)
		_, err = impl.installedAppRepository.UpdateInstalledAppVersion(installedAppVersionModel, tx)
		if err != nil {
			impl.logger.Errorw("error while fetching from db", "error", err)
			return err
		}
		appStoreAppVersion, err := impl.appStoreApplicationVersionRepository.FindById(request.AppStoreVersion)
		if err != nil {
			impl.logger.Errorw("fetching error", "err", err)
			return err
		}
		installedAppVersion = &repository.InstalledAppVersions{
			InstalledAppId:               installedAppVersionModel.InstalledAppId,
			AppStoreApplicationVersionId: request.AppStoreVersion,
			ValuesYaml:                   request.GetValuesYaml(),
		}
		installedAppVersion.CreatedBy = int32(request.UserId)
		installedAppVersion.UpdatedBy = int32(request.UserId)
		installedAppVersion.CreatedOn = time.Now()
		installedAppVersion.UpdatedOn = time.Now()
		installedAppVersion.Active = true
		installedAppVersion.ReferenceValueId = request.ReferenceValueId
		installedAppVersion.ReferenceValueKind = request.ReferenceValueKind
		_, err = impl.installedAppRepository.CreateInstalledAppVersion(installedAppVersion, tx)
		if err != nil {
			impl.logger.Errorw("error while fetching from db", "error", err)
			return err
		}
		installedAppVersion.AppStoreApplicationVersion = *appStoreAppVersion

		//update requirements yaml in chart

		request.InstalledAppVersionId = installedAppVersion.Id
	} else {
		installedAppVersion = installedAppVersionModel
	}

	//DB operation
	installedAppVersion.ValuesYaml = request.GetValuesYaml()
	installedAppVersion.UpdatedOn = time.Now()
	installedAppVersion.UpdatedBy = int32(request.UserId)
	installedAppVersion.ReferenceValueId = request.ReferenceValueId
	installedAppVersion.ReferenceValueKind = request.ReferenceValueKind
	_, err = impl.installedAppRepository.UpdateInstalledAppVersion(installedAppVersion, tx)
	if err != nil {
		impl.logger.Errorw("error while fetching from db", "error", err)
		return err
	}

	//STEP 8: finish with return response
	err = tx.Commit()
	if err != nil {
		impl.logger.Errorw("error while committing transaction to db", "error", err)
		return err
	}
	return nil
}

//converts db object to bean
func (impl AppStoreDeploymentCommonServiceImpl) convert(chart *repository.InstalledApps, installedAppVersion *repository.InstalledAppVersions) *appStoreBean.InstallAppVersionDTO {
	chartVersionApp := installedAppVersion.AppStoreApplicationVersion
	chartRepo := installedAppVersion.AppStoreApplicationVersion.AppStore.ChartRepo
	return &appStoreBean.InstallAppVersionDTO{
		EnvironmentId:         chart.EnvironmentId,
		Id:                    chart.Id,
		AppId:                 chart.AppId,
		AppOfferingMode:       chart.App.AppOfferingMode,
		ClusterId:             chart.Environment.ClusterId,
		Namespace:             chart.Environment.Namespace,
		AppName:               chart.App.AppName,
		EnvironmentName:       chart.Environment.Name,
		InstalledAppId:        chart.Id,
		InstalledAppVersionId: installedAppVersion.Id,
		AppStoreVersion:       installedAppVersion.AppStoreApplicationVersionId,
		ReferenceValueId:      installedAppVersion.ReferenceValueId,
		ReferenceValueKind:    installedAppVersion.ReferenceValueKind,
		InstallAppVersionChartDTO: &appStoreBean.InstallAppVersionChartDTO{
			AppStoreChartId: chartVersionApp.AppStore.Id,
			ChartName:       chartVersionApp.Name,
			ChartVersion:    chartVersionApp.Version,
			InstallAppVersionChartRepoDTO: &appStoreBean.InstallAppVersionChartRepoDTO{
				RepoName: chartRepo.Name,
				RepoUrl:  chartRepo.Url,
				UserName: chartRepo.UserName,
				Password: chartRepo.Password,
			},
		},
	}
}
