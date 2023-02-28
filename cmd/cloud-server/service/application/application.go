/*
 * TencentBlueKing is pleased to support the open source community by making
 * 蓝鲸智云 - 混合云管理平台 (BlueKing - Hybrid Cloud Management System) available.
 * Copyright (C) 2022 THL A29 Limited,
 * a Tencent company. All rights reserved.
 * Licensed under the MIT License (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at http://opensource.org/licenses/MIT
 * Unless required by applicable law or agreed to in writing,
 * software distributed under the License is distributed on
 * an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
 * either express or implied. See the License for the
 * specific language governing permissions and limitations under the License.
 *
 * We undertake not to change the open source license (MIT license) applicable
 *
 * to the current version of the project delivered to anyone in the future.
 */

package application

import (
	"fmt"
	"strings"

	"hcm/cmd/cloud-server/service/capability"
	"hcm/pkg/api/core"
	dataproto "hcm/pkg/api/data-service"
	"hcm/pkg/client"
	"hcm/pkg/criteria/enumor"
	"hcm/pkg/cryptography"
	"hcm/pkg/iam/auth"
	"hcm/pkg/rest"
	"hcm/pkg/runtime/filter"
	"hcm/pkg/thirdparty/esb"
)

func InitApplicationService(c *capability.Capability, bkHcmUrl string, platformManagers []string) {
	svc := &applicationSvc{
		client:           c.ApiClient,
		authorizer:       c.Authorizer,
		cipher:           c.Cipher,
		esbClient:        c.EsbClient,
		bkHcmUrl:         bkHcmUrl,
		platformManagers: platformManagers,
	}
	h := rest.NewHandler()
	h.Add("CreateForAddAccount", "POST", "/applications/types/add_account", svc.CreateForAddAccount)
	h.Add("List", "POST", "/applications/list", svc.List)
	h.Add("Get", "GET", "/applications/{application_id}", svc.Get)
	h.Add("Cancel", "PATCH", "/applications/{application_id}/cancel", svc.Cancel)
	h.Add("Approve", "POST", "/applications/approve", svc.Approve)

	h.Load(c.WebService)
}

type applicationSvc struct {
	client           *client.ClientSet
	authorizer       auth.Authorizer
	cipher           cryptography.Crypto
	esbClient        esb.Client
	bkHcmUrl         string
	platformManagers []string
}

func (a *applicationSvc) getCallbackUrl() string {
	return fmt.Sprintf("%s/api/v1/cloud/applications/approve", strings.TrimRight(a.bkHcmUrl, "/"))
}

func (a *applicationSvc) getApprovalProcessServiceID(cts *rest.Contexts, applicationType enumor.ApplicationType) (int64, error) {
	result, err := a.client.DataService().Global.ApprovalProcess.List(
		cts.Kit.Ctx,
		cts.Kit.Header(),
		&dataproto.ApprovalProcessListReq{
			Filter: &filter.Expression{
				Op: filter.And,
				Rules: []filter.RuleFactory{
					filter.AtomRule{
						Field: "application_type",
						Op:    filter.Equal.Factory(),
						Value: string(applicationType),
					},
				},
			},
			Page: &core.BasePage{
				Count: false,
				Start: 0,
				Limit: 1,
			},
		},
	)
	if err != nil {
		return 0, err
	}
	if result.Details == nil || len(result.Details) != 1 {
		return 0, fmt.Errorf("approval process of [%s] not init", applicationType)
	}

	return result.Details[0].ServiceID, nil
}