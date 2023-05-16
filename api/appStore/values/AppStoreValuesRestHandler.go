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

package appStoreValues

import (
	"encoding/json"
	"github.com/devtron-labs/devtron/api/restHandler/common"
	appStoreBean "github.com/devtron-labs/devtron/pkg/appStore/bean"
	"github.com/devtron-labs/devtron/pkg/appStore/values/service"
	"github.com/devtron-labs/devtron/pkg/user"
	"github.com/gorilla/mux"
	"go.uber.org/zap"
	"net/http"
	"strconv"
)

type AppStoreValuesRestHandler interface {
	CreateAppStoreVersionValues(w http.ResponseWriter, r *http.Request)
	UpdateAppStoreVersionValues(w http.ResponseWriter, r *http.Request)
	FindValuesById(w http.ResponseWriter, r *http.Request)
	DeleteAppStoreVersionValues(w http.ResponseWriter, r *http.Request)

	FindValuesByAppStoreIdAndReferenceType(w http.ResponseWriter, r *http.Request)
	FetchTemplateValuesByAppStoreId(w http.ResponseWriter, r *http.Request)
	GetSelectedChartMetadata(w http.ResponseWriter, r *http.Request)
	GetChartForLatestDeployment(w http.ResponseWriter, r *http.Request)
}

type AppStoreValuesRestHandlerImpl struct {
	Logger                *zap.SugaredLogger
	userAuthService       user.UserService
	appStoreValuesService service.AppStoreValuesService
}

func NewAppStoreValuesRestHandlerImpl(Logger *zap.SugaredLogger, userAuthService user.UserService,
	appStoreValuesService service.AppStoreValuesService) *AppStoreValuesRestHandlerImpl {
	return &AppStoreValuesRestHandlerImpl{
		Logger:                Logger,
		userAuthService:       userAuthService,
		appStoreValuesService: appStoreValuesService,
	}
}

func (handler AppStoreValuesRestHandlerImpl) CreateAppStoreVersionValues(w http.ResponseWriter, r *http.Request) {
	decoder := json.NewDecoder(r.Body)
	userId, err := handler.userAuthService.GetLoggedInUser(r)
	if userId == 0 || err != nil {
		common.WriteJsonResp(w, err, "Unauthorized User", http.StatusUnauthorized)
		return
	}
	var request appStoreBean.AppStoreVersionValuesDTO
	err = decoder.Decode(&request)
	if err != nil {
		handler.Logger.Errorw("request err, CreateAppStoreVersionValues", "err", err, "payload", request)
		common.WriteJsonResp(w, err, nil, http.StatusBadRequest)
		return
	}
	request.UserId = userId
	handler.Logger.Infow("request payload, CreateAppStoreVersionValues", "payload", request)
	res, err := handler.appStoreValuesService.CreateAppStoreVersionValues(&request)
	if err != nil {
		handler.Logger.Errorw("service err, CreateAppStoreVersionValues", "err", err, "payload", request)
		common.WriteJsonResp(w, err, nil, http.StatusInternalServerError)
		return
	}
	common.WriteJsonResp(w, err, res, http.StatusOK)
}

func (handler AppStoreValuesRestHandlerImpl) UpdateAppStoreVersionValues(w http.ResponseWriter, r *http.Request) {
	decoder := json.NewDecoder(r.Body)
	userId, err := handler.userAuthService.GetLoggedInUser(r)
	if userId == 0 || err != nil {
		common.WriteJsonResp(w, err, "Unauthorized User", http.StatusUnauthorized)
		return
	}
	var request appStoreBean.AppStoreVersionValuesDTO
	err = decoder.Decode(&request)
	if err != nil {
		handler.Logger.Errorw("request err, UpdateAppStoreVersionValues", "err", err, "payload", request)
		common.WriteJsonResp(w, err, nil, http.StatusBadRequest)
		return
	}
	request.UserId = userId
	handler.Logger.Infow("request payload, UpdateAppStoreVersionValues", "payload", request)
	res, err := handler.appStoreValuesService.UpdateAppStoreVersionValues(&request)
	if err != nil {
		handler.Logger.Errorw("service err, UpdateAppStoreVersionValues", "err", err, "payload", request)
		common.WriteJsonResp(w, err, nil, http.StatusInternalServerError)
		return
	}
	common.WriteJsonResp(w, err, res, http.StatusOK)
}

func (handler AppStoreValuesRestHandlerImpl) FindValuesById(w http.ResponseWriter, r *http.Request) {
	userId, err := handler.userAuthService.GetLoggedInUser(r)
	if userId == 0 || err != nil {
		common.WriteJsonResp(w, err, "Unauthorized User", http.StatusUnauthorized)
		return
	}
	vars := mux.Vars(r)
	referenceId, err := strconv.Atoi(vars["referenceId"])
	if err != nil || referenceId == 0 {
		handler.Logger.Errorw("request err, FindValuesById", "err", err, "referenceId", referenceId)
		common.WriteJsonResp(w, err, nil, http.StatusBadRequest)
		return
	}
	kind := vars["kind"]
	if len(kind) == 0 || (kind != appStoreBean.REFERENCE_TYPE_DEPLOYED && kind != appStoreBean.REFERENCE_TYPE_DEFAULT && kind != appStoreBean.REFERENCE_TYPE_TEMPLATE && kind != appStoreBean.REFERENCE_TYPE_EXISTING) {
		handler.Logger.Error(err)
		common.WriteJsonResp(w, err, nil, http.StatusBadRequest)
		return
	}
	handler.Logger.Infow("request payload, FindValuesById", "referenceId", referenceId, "kind", kind)
	res, err := handler.appStoreValuesService.FindValuesByIdAndKind(referenceId, kind)
	if err != nil {
		handler.Logger.Errorw("service err, FindValuesById", "err", err, "payload", referenceId, "kind", kind)
		common.WriteJsonResp(w, err, nil, http.StatusInternalServerError)
		return
	}
	common.WriteJsonResp(w, err, res, http.StatusOK)
}

func (handler AppStoreValuesRestHandlerImpl) DeleteAppStoreVersionValues(w http.ResponseWriter, r *http.Request) {
	userId, err := handler.userAuthService.GetLoggedInUser(r)
	if userId == 0 || err != nil {
		common.WriteJsonResp(w, err, nil, http.StatusUnauthorized)
		return
	}
	vars := mux.Vars(r)
	appStoreValueId, err := strconv.Atoi(vars["appStoreValueId"])
	if err != nil {
		handler.Logger.Errorw("request err, DeleteAppStoreVersionValues", "err", err, "appStoreValueId", appStoreValueId)
		common.WriteJsonResp(w, err, nil, http.StatusBadRequest)
		return
	}
	handler.Logger.Infow("request payload, DeleteAppStoreVersionValues", "appStoreValueId", appStoreValueId)

	res, err := handler.appStoreValuesService.DeleteAppStoreVersionValues(appStoreValueId)
	if err != nil {
		handler.Logger.Errorw("service err, DeleteAppStoreVersionValues", "err", err, "appStoreValueId", appStoreValueId)
		common.WriteJsonResp(w, err, nil, http.StatusInternalServerError)
		return
	}
	common.WriteJsonResp(w, err, res, http.StatusOK)
}

func (handler AppStoreValuesRestHandlerImpl) FindValuesByAppStoreIdAndReferenceType(w http.ResponseWriter, r *http.Request) {
	userId, err := handler.userAuthService.GetLoggedInUser(r)
	if userId == 0 || err != nil {
		common.WriteJsonResp(w, err, "Unauthorized User", http.StatusUnauthorized)
		return
	}

	vars := mux.Vars(r)
	appStoreVersionId, err := strconv.Atoi(vars["appStoreId"])
	if err != nil {
		handler.Logger.Errorw("request err, FindValuesByAppStoreIdAndReferenceType", "err", err, "appStoreVersionId", appStoreVersionId)
		common.WriteJsonResp(w, err, nil, http.StatusBadRequest)
		return
	}
	handler.Logger.Infow("request payload, FindValuesByAppStoreIdAndReferenceType", "appStoreVersionId", appStoreVersionId)
	res, err := handler.appStoreValuesService.FindValuesByAppStoreIdAndReferenceType(appStoreVersionId, appStoreBean.REFERENCE_TYPE_TEMPLATE)
	if err != nil {
		handler.Logger.Errorw("service err, FindValuesByAppStoreIdAndReferenceType", "err", err, "appStoreVersionId", appStoreVersionId)
		common.WriteJsonResp(w, err, nil, http.StatusInternalServerError)
		return
	}
	common.WriteJsonResp(w, err, res, http.StatusOK)
}

func (handler AppStoreValuesRestHandlerImpl) FetchTemplateValuesByAppStoreId(w http.ResponseWriter, r *http.Request) {
	userId, err := handler.userAuthService.GetLoggedInUser(r)
	if userId == 0 || err != nil {
		common.WriteJsonResp(w, err, "Unauthorized User", http.StatusUnauthorized)
		return
	}
	vars := mux.Vars(r)
	appStoreId, err := strconv.Atoi(vars["appStoreId"])
	if err != nil {
		handler.Logger.Errorw("request err, FetchTemplateValuesByAppStoreId", "err", err, "appStoreId", appStoreId)
		common.WriteJsonResp(w, err, nil, http.StatusBadRequest)
		return
	}

	v := r.URL.Query()
	var installedAppVersionId int
	installedAppVersionIds := v.Get("installedAppVersionId")
	if len(installedAppVersionIds) > 0 {
		installedAppVersionId, err = strconv.Atoi(installedAppVersionIds)
		if err != nil {
			handler.Logger.Errorw("request err, FetchTemplateValuesByAppStoreId", "err", err, "installedAppVersionIds", installedAppVersionIds)
			common.WriteJsonResp(w, err, nil, http.StatusBadRequest)
			return
		}
	}

	handler.Logger.Infow("request payload, FetchTemplateValuesByAppStoreId", "appStoreId", appStoreId)
	res, err := handler.appStoreValuesService.FindValuesByAppStoreId(appStoreId, installedAppVersionId)
	if err != nil {
		handler.Logger.Errorw("service err, FetchTemplateValuesByAppStoreId", "err", err, "appStoreId", appStoreId)
		common.WriteJsonResp(w, err, nil, http.StatusInternalServerError)
		return
	}
	common.WriteJsonResp(w, err, res, http.StatusOK)
}

func (handler AppStoreValuesRestHandlerImpl) GetSelectedChartMetadata(w http.ResponseWriter, r *http.Request) {
	userId, err := handler.userAuthService.GetLoggedInUser(r)
	if userId == 0 || err != nil {
		common.WriteJsonResp(w, err, "Unauthorized User", http.StatusUnauthorized)
		return
	}
	decoder := json.NewDecoder(r.Body)
	var request service.ChartMetaDataRequestWrapper
	err = decoder.Decode(&request)
	if err != nil {
		handler.Logger.Errorw("request err, GetSelectedChartMetadata", "err", err, "request", request)
		common.WriteJsonResp(w, err, nil, http.StatusBadRequest)
		return
	}
	handler.Logger.Infow("request payload, GetSelectedChartMetadata", "request", request)
	res, err := handler.appStoreValuesService.GetSelectedChartMetaData(&request)
	if err != nil {
		handler.Logger.Errorw("service err, GetSelectedChartMetadata", "err", err, "request", request)
		common.WriteJsonResp(w, err, nil, http.StatusInternalServerError)
		return
	}
	common.WriteJsonResp(w, err, res, http.StatusOK)
}

func (handler AppStoreValuesRestHandlerImpl) GetChartForLatestDeployment(w http.ResponseWriter, r *http.Request) {
	userId, err := handler.userAuthService.GetLoggedInUser(r)
	if userId == 0 || err != nil {
		common.WriteJsonResp(w, err, "Unauthorized User", http.StatusUnauthorized)
		return
	}
	vars := mux.Vars(r)
	installedAppId, err := strconv.Atoi(vars["installed-app-id"])
	if err != nil {
		handler.Logger.Errorw("request err, FetchAppDetailsForInstalledAppV2", "err", err, "installedAppId", installedAppId)
		common.WriteJsonResp(w, err, nil, http.StatusBadRequest)
		return
	}
	envId, err := strconv.Atoi(vars["env-id"])
	if err != nil {
		handler.Logger.Errorw("request err, FetchAppDetailsForInstalledAppV2", "err", err, "installedAppId", installedAppId, "envId", envId)
		common.WriteJsonResp(w, err, nil, http.StatusBadRequest)
		return
	}
	//token := r.Header.Get("token")

}
