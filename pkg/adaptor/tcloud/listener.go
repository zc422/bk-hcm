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

package tcloud

import (
	"errors"
	"fmt"

	"hcm/pkg/adaptor/poller"
	"hcm/pkg/adaptor/types"
	typelb "hcm/pkg/adaptor/types/load-balancer"
	corelb "hcm/pkg/api/core/cloud/load-balancer"
	"hcm/pkg/criteria/errf"
	"hcm/pkg/kit"
	"hcm/pkg/logs"
	"hcm/pkg/tools/converter"

	clb "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/clb/v20180317"
	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common"
)

// CreateListener 创建监听器 reference: https://cloud.tencent.com/document/api/214/30693
// 接口返回成功后，需以返回的 RequestId 为入参，调用 DescribeTaskStatus 接口查询本次任务是否成功
func (t *TCloudImpl) CreateListener(kt *kit.Kit, opt *typelb.TCloudCreateListenerOption) (
	*poller.BaseDoneResult, error) {
	if opt == nil {
		return nil, errf.New(errf.InvalidParameter, "create listener option is required")
	}

	if err := opt.Validate(); err != nil {
		return nil, errf.NewFromErr(errf.InvalidParameter, err)
	}

	client, err := t.clientSet.ClbClient(opt.Region)
	if err != nil {
		return nil, fmt.Errorf("init tencent cloud clb client failed, region: %s, err: %v", opt.Region, err)
	}

	req := t.formatCreateListenerRequest(opt)
	createResp, err := client.CreateListenerWithContext(kt.Ctx, req)
	if err != nil {
		logs.Errorf("create tencent cloud listener instance failed, req: %+v, err: %v, rid: %s", req, err, kt.Rid)
		return nil, err
	}
	respPoller := poller.Poller[*TCloudImpl, map[string]*clb.DescribeTaskStatusResponseParams, poller.BaseDoneResult]{
		Handler: &taskStatusDefaultPollingHandler{opt.Region},
	}

	reqID := createResp.Response.RequestId
	result, err := respPoller.PollUntilDone(t, kt, []*string{reqID}, types.NewLoadBalancerDefaultPollerOption())
	if err != nil {
		return nil, err
	}
	if len(result.SuccessCloudIDs) == 0 {
		return nil, errf.Newf(errf.CloudVendorError, "no any listener being created, TencentCloudSDK RequestId: %s",
			converter.PtrToVal(reqID))
	}
	result.SuccessCloudIDs = converter.PtrToSlice(createResp.Response.ListenerIds)
	return result, nil
}

var _ poller.PollingHandler[*TCloudImpl, map[string]*clb.DescribeTaskStatusResponseParams,
	poller.BaseDoneResult] = new(taskStatusDefaultPollingHandler)

type taskStatusDefaultPollingHandler struct {
	region string
}

// Done 成功状态判断
func (h *taskStatusDefaultPollingHandler) Done(taskStatusMap map[string]*clb.DescribeTaskStatusResponseParams) (
	bool, *poller.BaseDoneResult) {

	result := &poller.BaseDoneResult{
		SuccessCloudIDs: make([]string, 0),
		FailedCloudIDs:  make([]string, 0),
		UnknownCloudIDs: make([]string, 0),
	}

	for taskID, item := range taskStatusMap {
		if item.Status == nil {
			return false, nil
		}

		status := converter.PtrToVal(item.Status)
		switch status {
		case CLBTaskStatusRunning:
			// 还有任务在运行表示没有成功
			return false, nil
		case CLBTaskStatusFail:
			result.FailedCloudIDs = append(result.FailedCloudIDs, taskID)
		case CLBTaskStatusSuccess:
			result.SuccessCloudIDs = append(result.SuccessCloudIDs, taskID)
		}
	}
	return true, result
}

// Poll 返回任务状态
func (h *taskStatusDefaultPollingHandler) Poll(client *TCloudImpl, kt *kit.Kit, requestIDs []*string) (
	map[string]*clb.DescribeTaskStatusResponseParams, error) {

	taskOpt := &typelb.TCloudDescribeTaskStatusOption{Region: h.region}
	result := make(map[string]*clb.DescribeTaskStatusResponseParams)
	// 查询对应异步任务状态
	for _, reqID := range requestIDs {
		taskOpt.TaskId = converter.PtrToVal(reqID)
		if taskOpt.TaskId == "" {
			return nil, errors.New("empty request ID")
		}
		status, err := client.CLBDescribeTaskStatus(kt, taskOpt)
		if err != nil {
			return nil, err
		}

		result[taskOpt.TaskId] = status
	}
	return result, nil
}

func (t *TCloudImpl) formatCreateListenerRequest(opt *typelb.TCloudCreateListenerOption) *clb.CreateListenerRequest {
	req := clb.NewCreateListenerRequest()
	req.LoadBalancerId = converter.ValToPtr(opt.LoadBalancerId)
	req.ListenerNames = append(req.ListenerNames, converter.ValToPtr(opt.ListenerName))
	req.Protocol = converter.ValToPtr(string(opt.Protocol))
	req.Ports = append(req.Ports, converter.ValToPtr(opt.Port))
	req.MultiCertInfo = convCert(opt.Certificate)

	if len(opt.Scheduler) > 0 {
		req.Scheduler = converter.ValToPtr(opt.Scheduler)
	}
	if opt.SessionType != nil {
		req.SessionType = opt.SessionType
	}
	if opt.SessionExpireTime >= 0 {
		req.SessionExpireTime = converter.ValToPtr(opt.SessionExpireTime)
	}
	if opt.SniSwitch >= 0 {
		req.SniSwitch = converter.ValToPtr(int64(opt.SniSwitch))
	}
	if opt.HealthCheck != nil {
		req.HealthCheck = &clb.HealthCheck{
			HealthSwitch:    opt.HealthCheck.HealthSwitch,
			TimeOut:         opt.HealthCheck.TimeOut,
			IntervalTime:    opt.HealthCheck.IntervalTime,
			HealthNum:       opt.HealthCheck.HealthNum,
			UnHealthNum:     opt.HealthCheck.UnHealthNum,
			HttpCode:        opt.HealthCheck.HttpCode,
			HttpCheckPath:   opt.HealthCheck.HttpCheckPath,
			HttpCheckDomain: opt.HealthCheck.HttpCheckDomain,
			HttpCheckMethod: opt.HealthCheck.HttpCheckMethod,
			CheckPort:       opt.HealthCheck.CheckPort,
			ContextType:     opt.HealthCheck.ContextType,
			CheckType:       opt.HealthCheck.CheckType,
			HttpVersion:     opt.HealthCheck.HttpVersion,
			SourceIpType:    opt.HealthCheck.SourceIpType,
		}
	}

	return req
}

func convCert(optCert *corelb.TCloudCertificateInfo) *clb.MultiCertInfo {
	if optCert == nil {
		return nil
	}
	multiCert := &clb.MultiCertInfo{SSLMode: optCert.SSLMode}
	if optCert.CaCloudID != nil {
		multiCert.CertList = append(multiCert.CertList,
			&clb.CertInfo{CertId: optCert.CaCloudID})
	}
	for _, c := range optCert.ClientCloudIDs {
		multiCert.CertList = append(multiCert.CertList,
			&clb.CertInfo{CertId: converter.ValToPtr(c)})
	}
	return multiCert
}

// UpdateListener 更新监听器 reference: https://cloud.tencent.com/document/api/214/30681
func (t *TCloudImpl) UpdateListener(kt *kit.Kit, opt *typelb.TCloudUpdateListenerOption) error {
	if opt == nil {
		return errf.New(errf.InvalidParameter, "update listener option is required")
	}

	if err := opt.Validate(); err != nil {
		return errf.NewFromErr(errf.InvalidParameter, err)
	}

	client, err := t.clientSet.ClbClient(opt.Region)
	if err != nil {
		return fmt.Errorf("init tencent cloud clb client failed, region: %s, err: %v", opt.Region, err)
	}

	req := t.formatUpdateListenerRequest(opt)
	updateResp, err := client.ModifyListenerWithContext(kt.Ctx, req)
	if err != nil {
		logs.Errorf("update tcloud listener failed, err: %v, resp: %+v, opt: %+v, rid: %s",
			err, updateResp, opt, kt.Rid)
		return err
	}

	respPoller := poller.Poller[*TCloudImpl, map[string]*clb.DescribeTaskStatusResponseParams, poller.BaseDoneResult]{
		Handler: &taskStatusDefaultPollingHandler{opt.Region},
	}

	reqID := updateResp.Response.RequestId
	result, err := respPoller.PollUntilDone(t, kt, []*string{reqID}, types.NewLoadBalancerDefaultPollerOption())
	if err != nil {
		return err
	}
	if len(result.SuccessCloudIDs) == 0 {
		return errf.Newf(errf.CloudVendorError,
			"no any listener being updated, TencentCloudSDK RequestId: %s", converter.PtrToVal(reqID))
	}
	return nil
}

func (t *TCloudImpl) formatUpdateListenerRequest(opt *typelb.TCloudUpdateListenerOption) *clb.ModifyListenerRequest {
	req := clb.NewModifyListenerRequest()
	req.LoadBalancerId = converter.ValToPtr(opt.LoadBalancerId)
	req.ListenerId = converter.ValToPtr(opt.ListenerId)
	req.MultiCertInfo = convCert(opt.Certificate)

	if len(opt.ListenerName) > 0 {
		req.ListenerName = converter.ValToPtr(opt.ListenerName)
	}
	if len(opt.Scheduler) > 0 {
		req.Scheduler = converter.ValToPtr(opt.Scheduler)
	}
	if len(opt.SessionType) > 0 {
		req.SessionType = converter.ValToPtr(opt.SessionType)
	}
	if opt.SessionExpireTime >= 0 {
		req.SessionExpireTime = converter.ValToPtr(opt.SessionExpireTime)
	}
	if opt.SniSwitch >= 0 {
		req.SniSwitch = converter.ValToPtr(int64(opt.SniSwitch))
	}

	if opt.HealthCheck != nil {
		req.HealthCheck = &clb.HealthCheck{
			HealthSwitch:    opt.HealthCheck.HealthSwitch,
			TimeOut:         opt.HealthCheck.TimeOut,
			IntervalTime:    opt.HealthCheck.IntervalTime,
			HealthNum:       opt.HealthCheck.HealthNum,
			UnHealthNum:     opt.HealthCheck.UnHealthNum,
			HttpCode:        opt.HealthCheck.HttpCode,
			HttpCheckPath:   opt.HealthCheck.HttpCheckPath,
			HttpCheckDomain: opt.HealthCheck.HttpCheckDomain,
			HttpCheckMethod: opt.HealthCheck.HttpCheckMethod,
			CheckPort:       opt.HealthCheck.CheckPort,
			ContextType:     opt.HealthCheck.ContextType,
			CheckType:       opt.HealthCheck.CheckType,
			HttpVersion:     opt.HealthCheck.HttpVersion,
			SourceIpType:    opt.HealthCheck.SourceIpType,
		}
	}

	return req
}

// DeleteListener 删除监听器 reference: https://cloud.tencent.com/document/api/214/41504
// 本接口返回成功后需以返回的 RequestID 为入参，调用 DescribeTaskStatus 接口查询本次任务是否成功
func (t *TCloudImpl) DeleteListener(kt *kit.Kit, opt *typelb.TCloudDeleteListenerOption) error {
	if opt == nil {
		return errf.New(errf.InvalidParameter, "delete listener option is required")
	}

	if err := opt.Validate(); err != nil {
		return errf.NewFromErr(errf.InvalidParameter, err)
	}

	client, err := t.clientSet.ClbClient(opt.Region)
	if err != nil {
		return fmt.Errorf("init tencent cloud clb client failed, region: %s, err: %v", opt.Region, err)
	}

	req := clb.NewDeleteLoadBalancerListenersRequest()
	req.LoadBalancerId = common.StringPtr(opt.LoadBalancerId)
	req.ListenerIds = common.StringPtrs(opt.CloudIDs)
	deleteResp, err := client.DeleteLoadBalancerListenersWithContext(kt.Ctx, req)
	if err != nil {
		logs.Errorf("delete tcloud listener failed(RequestID:%s), opt: %+v, err: %v, rid: %s",
			deleteResp.Response.RequestId, opt, err, kt.Rid)
		return err
	}

	respPoller := poller.Poller[*TCloudImpl, map[string]*clb.DescribeTaskStatusResponseParams, poller.BaseDoneResult]{
		Handler: &taskStatusDefaultPollingHandler{opt.Region},
	}

	reqID := deleteResp.Response.RequestId
	result, err := respPoller.PollUntilDone(t, kt, []*string{reqID}, types.NewLoadBalancerDefaultPollerOption())
	if err != nil {
		return err
	}
	if len(result.SuccessCloudIDs) == 0 {
		return errf.Newf(errf.CloudVendorError,
			"no any listener being deleted, TencentCloudSDK RequestId: %s", converter.PtrToVal(reqID))
	}
	return nil
}

// CreateRule 创建7层规则接口 reference: https://cloud.tencent.com/document/api/214/30691
// 接口返回成功后，需以返回的 RequestId 为入参，调用 DescribeTaskStatus 接口查询本次任务是否成功
func (t *TCloudImpl) CreateRule(kt *kit.Kit, opt *typelb.TCloudCreateRuleOption) (
	*poller.BaseDoneResult, error) {
	if opt == nil {
		return nil, errf.New(errf.InvalidParameter, "create rule option is required")
	}

	if err := opt.Validate(); err != nil {
		return nil, errf.NewFromErr(errf.InvalidParameter, err)
	}

	client, err := t.clientSet.ClbClient(opt.Region)
	if err != nil {
		return nil, fmt.Errorf("init tencent cloud clb client failed, region: %s, err: %v", opt.Region, err)
	}

	req := t.formatCreateRuleRequest(opt)
	createResp, err := client.CreateRuleWithContext(kt.Ctx, req)
	if err != nil {
		logs.Errorf("create tencent cloud rule instance failed, req: %+v, err: %v, rid: %s", req, err, kt.Rid)
		return nil, err
	}
	respPoller := poller.Poller[*TCloudImpl, map[string]*clb.DescribeTaskStatusResponseParams, poller.BaseDoneResult]{
		Handler: &taskStatusDefaultPollingHandler{opt.Region},
	}

	reqID := createResp.Response.RequestId
	result, err := respPoller.PollUntilDone(t, kt, []*string{reqID}, types.NewLoadBalancerDefaultPollerOption())
	if err != nil {
		return nil, err
	}
	if len(result.SuccessCloudIDs) == 0 {
		return nil, errf.Newf(errf.CloudVendorError,
			"no any rule being created, TencentCloudSDK RequestId: %s", converter.PtrToVal(reqID))
	}
	result.SuccessCloudIDs = converter.PtrToSlice(createResp.Response.LocationIds)
	return result, nil
}

func (t *TCloudImpl) formatCreateRuleRequest(opt *typelb.TCloudCreateRuleOption) *clb.CreateRuleRequest {
	req := clb.NewCreateRuleRequest()
	req.LoadBalancerId = converter.ValToPtr(opt.LoadBalancerId)
	req.ListenerId = converter.ValToPtr(opt.ListenerId)

	req.Rules = make([]*clb.RuleInput, 0)
	for _, item := range opt.Rules {
		tmpRule := &clb.RuleInput{
			Domain:            item.Domain,
			Url:               item.Url,
			SessionExpireTime: item.SessionExpireTime,
			Scheduler:         item.Scheduler,
		}

		if item.HealthCheck != nil {
			tmpRule.HealthCheck = &clb.HealthCheck{
				HealthSwitch:    item.HealthCheck.HealthSwitch,
				TimeOut:         item.HealthCheck.TimeOut,
				IntervalTime:    item.HealthCheck.IntervalTime,
				HealthNum:       item.HealthCheck.HealthNum,
				UnHealthNum:     item.HealthCheck.UnHealthNum,
				HttpCode:        item.HealthCheck.HttpCode,
				HttpCheckPath:   item.HealthCheck.HttpCheckPath,
				HttpCheckDomain: item.HealthCheck.HttpCheckDomain,
				HttpCheckMethod: item.HealthCheck.HttpCheckMethod,
				CheckPort:       item.HealthCheck.CheckPort,
				ContextType:     item.HealthCheck.ContextType,
				CheckType:       item.HealthCheck.CheckType,
				HttpVersion:     item.HealthCheck.HttpVersion,
				SourceIpType:    item.HealthCheck.SourceIpType,
			}
		}
		tmpRule.MultiCertInfo = convCert(item.Certificate)

		req.Rules = append(req.Rules, tmpRule)
	}
	return req
}

// UpdateRule 更新7层规则接口 reference: https://cloud.tencent.com/document/api/214/30679
// 接口返回成功后，需以返回的 RequestId 为入参，调用 DescribeTaskStatus 接口查询本次任务是否成功
func (t *TCloudImpl) UpdateRule(kt *kit.Kit, opt *typelb.TCloudUpdateRuleOption) error {
	if opt == nil {
		return errf.New(errf.InvalidParameter, "update rule option is required")
	}

	if err := opt.Validate(); err != nil {
		return errf.NewFromErr(errf.InvalidParameter, err)
	}

	client, err := t.clientSet.ClbClient(opt.Region)
	if err != nil {
		return fmt.Errorf("init tencent cloud clb client failed, region: %s, err: %v", opt.Region, err)
	}

	req := clb.NewModifyRuleRequest()
	req.LoadBalancerId = converter.ValToPtr(opt.LoadBalancerId)
	req.ListenerId = converter.ValToPtr(opt.ListenerId)
	if len(opt.Url) > 0 {
		req.Url = converter.ValToPtr(opt.Url)
	}
	if len(opt.Scheduler) > 0 {
		req.Scheduler = converter.ValToPtr(opt.Scheduler)
	}
	if opt.SessionExpireTime >= 0 {
		req.SessionExpireTime = converter.ValToPtr(opt.SessionExpireTime)
	}
	if len(opt.ForwardType) > 0 {
		req.ForwardType = converter.ValToPtr(opt.ForwardType)
	}
	if opt.HealthCheck != nil {
		req.HealthCheck = &clb.HealthCheck{
			HealthSwitch:    opt.HealthCheck.HealthSwitch,
			TimeOut:         opt.HealthCheck.TimeOut,
			IntervalTime:    opt.HealthCheck.IntervalTime,
			HealthNum:       opt.HealthCheck.HealthNum,
			UnHealthNum:     opt.HealthCheck.UnHealthNum,
			HttpCode:        opt.HealthCheck.HttpCode,
			HttpCheckPath:   opt.HealthCheck.HttpCheckPath,
			HttpCheckDomain: opt.HealthCheck.HttpCheckDomain,
			HttpCheckMethod: opt.HealthCheck.HttpCheckMethod,
			CheckPort:       opt.HealthCheck.CheckPort,
			ContextType:     opt.HealthCheck.ContextType,
			CheckType:       opt.HealthCheck.CheckType,
			HttpVersion:     opt.HealthCheck.HttpVersion,
			SourceIpType:    opt.HealthCheck.SourceIpType,
		}
	}
	updateResp, err := client.ModifyRuleWithContext(kt.Ctx, req)
	if err != nil {
		logs.Errorf("update tencent cloud rule instance failed, req: %+v, err: %v, rid: %s", req, err, kt.Rid)
		return err
	}
	respPoller := poller.Poller[*TCloudImpl, map[string]*clb.DescribeTaskStatusResponseParams, poller.BaseDoneResult]{
		Handler: &taskStatusDefaultPollingHandler{opt.Region},
	}

	reqID := updateResp.Response.RequestId
	result, err := respPoller.PollUntilDone(t, kt, []*string{reqID}, types.NewLoadBalancerDefaultPollerOption())
	if err != nil {
		return err
	}
	if len(result.SuccessCloudIDs) == 0 {
		return errf.Newf(errf.CloudVendorError,
			"no any rule being updated, TencentCloudSDK RequestId: %s", converter.PtrToVal(reqID))
	}
	return nil
}

// UpdateDomainAttr 更新域名属性 reference: https://cloud.tencent.com/document/api/214/38092
// 接口返回成功后，需以返回的 RequestId 为入参，调用 DescribeTaskStatus 接口查询本次任务是否成功
func (t *TCloudImpl) UpdateDomainAttr(kt *kit.Kit, opt *typelb.TCloudUpdateDomainAttrOption) error {
	if opt == nil {
		return errf.New(errf.InvalidParameter, "update rule option is required")
	}

	if err := opt.Validate(); err != nil {
		return errf.NewFromErr(errf.InvalidParameter, err)
	}

	client, err := t.clientSet.ClbClient(opt.Region)
	if err != nil {
		return fmt.Errorf("init tencent cloud clb client failed, region: %s, err: %v", opt.Region, err)
	}

	req := clb.NewModifyDomainAttributesRequest()
	req.LoadBalancerId = converter.ValToPtr(opt.LoadBalancerId)
	req.ListenerId = converter.ValToPtr(opt.ListenerId)
	if len(opt.Domain) > 0 {
		req.Domain = converter.ValToPtr(opt.Domain)
	}
	if len(opt.NewDomain) > 0 {
		req.NewDomain = converter.ValToPtr(opt.NewDomain)
	}
	if opt.DefaultServer != nil {
		req.DefaultServer = opt.DefaultServer
	}
	if len(opt.NewDefaultServerDomain) > 0 {
		req.NewDefaultServerDomain = converter.ValToPtr(opt.NewDefaultServerDomain)
	}
	updateResp, err := client.ModifyDomainAttributesWithContext(kt.Ctx, req)
	if err != nil {
		logs.Errorf("update tencent cloud domain attr instance failed, req: %+v, err: %v, rid: %s", req, err, kt.Rid)
		return err
	}
	respPoller := poller.Poller[*TCloudImpl, map[string]*clb.DescribeTaskStatusResponseParams, poller.BaseDoneResult]{
		Handler: &taskStatusDefaultPollingHandler{opt.Region},
	}

	reqID := updateResp.Response.RequestId
	result, err := respPoller.PollUntilDone(t, kt, []*string{reqID}, types.NewLoadBalancerDefaultPollerOption())
	if err != nil {
		return err
	}
	if len(result.SuccessCloudIDs) == 0 {
		return errf.Newf(errf.CloudVendorError,
			"no any domain attributes being updated, TencentCloudSDK RequestId: %s", converter.PtrToVal(reqID))
	}
	return nil
}

// DeleteRule 删除监听器 reference: https://cloud.tencent.com/document/api/214/30688
// 本接口返回成功后需以返回的 RequestID 为入参，调用 DescribeTaskStatus 接口查询本次任务是否成功
func (t *TCloudImpl) DeleteRule(kt *kit.Kit, opt *typelb.TCloudDeleteRuleOption) error {
	if opt == nil {
		return errf.New(errf.InvalidParameter, "delete rule option is required")
	}

	if err := opt.Validate(); err != nil {
		return errf.NewFromErr(errf.InvalidParameter, err)
	}

	client, err := t.clientSet.ClbClient(opt.Region)
	if err != nil {
		return fmt.Errorf("init tencent cloud clb client failed, region: %s, err: %v", opt.Region, err)
	}

	req := clb.NewDeleteRuleRequest()
	req.LoadBalancerId = common.StringPtr(opt.LoadBalancerId)
	req.LocationIds = common.StringPtrs(opt.CloudIDs)
	if len(opt.Domain) > 0 {
		req.Domain = converter.ValToPtr(opt.Domain)
	}
	if len(opt.Url) > 0 {
		req.Url = converter.ValToPtr(opt.Url)
	}
	if len(opt.NewDefaultServerDomain) > 0 {
		req.NewDefaultServerDomain = converter.ValToPtr(opt.NewDefaultServerDomain)
	}
	deleteResp, err := client.DeleteRuleWithContext(kt.Ctx, req)
	if err != nil {
		logs.Errorf("delete tcloud rule failed(RequestID:%s), opt: %+v, err: %v, rid: %s",
			deleteResp.Response.RequestId, opt, err, kt.Rid)
		return err
	}

	respPoller := poller.Poller[*TCloudImpl, map[string]*clb.DescribeTaskStatusResponseParams, poller.BaseDoneResult]{
		Handler: &taskStatusDefaultPollingHandler{opt.Region},
	}

	reqID := deleteResp.Response.RequestId
	result, err := respPoller.PollUntilDone(t, kt, []*string{reqID}, types.NewLoadBalancerDefaultPollerOption())
	if err != nil {
		return err
	}
	if len(result.SuccessCloudIDs) == 0 {
		return errf.Newf(errf.CloudVendorError,
			"no any rule being deleted, TencentCloudSDK RequestId: %s", converter.PtrToVal(reqID))
	}
	return nil
}
