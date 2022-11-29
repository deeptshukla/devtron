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

package router

import (
	"github.com/devtron-labs/devtron/api/restHandler"
	"github.com/devtron-labs/devtron/api/restHandler/app"
	"github.com/gorilla/mux"
)

type WebhookRouter interface {
	intWebhookRouter(configRouter *mux.Router)
}

type WebhookRouterImpl struct {
	gitWebhookRestHandler   restHandler.GitWebhookRestHandler
	pipelineRestHandler     app.PipelineConfigRestHandler
	externalCiRestHandler   restHandler.ExternalCiRestHandler
	pubSubClientRestHandler restHandler.PubSubClientRestHandler
}

func NewWebhookRouterImpl(gitWebhookRestHandler restHandler.GitWebhookRestHandler,
	pipelineRestHandler app.PipelineConfigRestHandler, externalCiRestHandler restHandler.ExternalCiRestHandler,
	pubSubClientRestHandler restHandler.PubSubClientRestHandler) *WebhookRouterImpl {
	return &WebhookRouterImpl{
		gitWebhookRestHandler:   gitWebhookRestHandler,
		pipelineRestHandler:     pipelineRestHandler,
		externalCiRestHandler:   externalCiRestHandler,
		pubSubClientRestHandler: pubSubClientRestHandler,
	}
}

func (router *WebhookRouterImpl) intWebhookRouter(configRouter *mux.Router) {
	configRouter.Path("/git").HandlerFunc(router.gitWebhookRestHandler.HandleGitWebhook).Methods("POST")
	configRouter.Path("/ci/workflow").HandlerFunc(router.pipelineRestHandler.HandleWorkflowWebhook).Methods("POST")
	configRouter.Path("/msg/nats").HandlerFunc(router.pubSubClientRestHandler.PublishEventsToNats).Methods("POST")
	configRouter.Path("/ext-ci/{externalCiId}").HandlerFunc(router.externalCiRestHandler.HandleExternalCiWebhook).Methods("POST")
}
