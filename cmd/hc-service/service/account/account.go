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

// Package account defines account service.
package account

import (
	"hcm/cmd/hc-service/service/capability"
	"hcm/pkg/adaptor"
	"hcm/pkg/adaptor/types"
	proto "hcm/pkg/api/hc-service"
	"hcm/pkg/criteria/errf"
	"hcm/pkg/rest"
)

// InitAccountService initial the account service
func InitAccountService(cap *capability.Capability) {
	a := &account{
		ad: cap.Adaptor,
	}

	h := rest.NewHandler()
	h.Add("TCloudAccountCheck", "POST", "/vendors/tcloud/accounts/check", a.TCloudAccountCheck)
	h.Add("AwsAccountCheck", "POST", "/vendors/aws/accounts/check", a.AwsAccountCheck)
	h.Add("HuaWeiAccountCheck", "POST", "/vendors/huawei/accounts/check", a.HuaWeiAccountCheck)
	// h.Add("GcpAccountCheck", "POST", "/vendors/gcp/accounts/check", a.GcpAccountCheck)
	// h.Add("AzureAccountCheck", "POST", "/vendors/azure/accounts/check", a.AzureAccountCheck)

	h.Load(cap.WebService)
}

type account struct {
	ad *adaptor.Adaptor
}

// TCloudAccountCheck authentication information and permissions.
func (a account) TCloudAccountCheck(cts *rest.Contexts) (interface{}, error) {
	req := new(proto.TCloudAccountCheckReq)
	if err := cts.DecodeInto(req); err != nil {
		return nil, errf.New(errf.DecodeRequestFailed, err.Error())
	}

	if err := req.Validate(); err != nil {
		return nil, errf.Newf(errf.InvalidParameter, err.Error())
	}

	err := a.ad.TCloud().AccountCheck(
		cts.Kit,
		&types.BaseSecret{CloudSecretID: req.CloudSecretID, CloudSecretKey: req.CloudSecretKey},
		&types.TCloudAccountInfo{CloudMainAccountID: req.CloudMainAccountID, CloudSubAccountID: req.CloudSubAccountID},
	)

	return nil, err
}

// AwsAccountCheck authentication information and permissions.
func (a account) AwsAccountCheck(cts *rest.Contexts) (interface{}, error) {
	req := new(proto.AwsAccountCheckReq)
	if err := cts.DecodeInto(req); err != nil {
		return nil, errf.New(errf.DecodeRequestFailed, err.Error())
	}

	if err := req.Validate(); err != nil {
		return nil, errf.Newf(errf.InvalidParameter, err.Error())
	}

	err := a.ad.Aws().AccountCheck(
		cts.Kit,
		&types.BaseSecret{CloudSecretID: req.CloudSecretID, CloudSecretKey: req.CloudSecretKey},
		&types.AwsAccountInfo{CloudAccountID: req.CloudAccountID, CloudIamUsername: req.CloudIamUsername},
	)

	return nil, err
}

// HuaWeiAccountCheck authentication information and permissions.
func (a account) HuaWeiAccountCheck(cts *rest.Contexts) (interface{}, error) {
	req := new(proto.HuaWeiAccountCheckReq)
	if err := cts.DecodeInto(req); err != nil {
		return nil, errf.New(errf.DecodeRequestFailed, err.Error())
	}

	if err := req.Validate(); err != nil {
		return nil, errf.Newf(errf.InvalidParameter, err.Error())
	}

	err := a.ad.HuaWei().AccountCheck(
		cts.Kit,
		&types.BaseSecret{CloudSecretID: req.CloudSecretID, CloudSecretKey: req.CloudSecretKey},
		&types.HuaWeiAccountInfo{
			CloudMainAccountName: req.CloudMainAccountName,
			CloudSubAccountID:    req.CloudSubAccountID,
			CloudSubAccountName:  req.CloudSubAccountName,
			// TODO: 产品上华为云账号就没有录入IamUserID和IamUsername，是否必须呢？如果必须，需要产品支持
			// IamUserCID: 	 req.IamUserID
			// IamUserName:     req.CloudIamUsername,
		},
	)

	return nil, err
}
