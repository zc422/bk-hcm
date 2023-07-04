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

package common

import (
	"fmt"

	typecvm "hcm/pkg/adaptor/types/cvm"
	"hcm/pkg/api/cloud-server/cvm"
	hcproto "hcm/pkg/api/hc-service/cvm"
	"hcm/pkg/tools/converter"
)

// ConvTCloudCvmCreateReq conv cvm create req.
func ConvTCloudCvmCreateReq(req *cscvm.TCloudCvmCreateReq) *hcproto.TCloudBatchCreateReq {
	// 自动续费&续费周期
	instanceChargePrepaid := &typecvm.TCloudInstanceChargePrepaid{
		Period: &req.InstanceChargePaidPeriod,
		// 默认通知但不自动续费
		RenewFlag: typecvm.NotifyAndManualRenew,
	}
	if req.AutoRenew != nil && *req.AutoRenew {
		instanceChargePrepaid.RenewFlag = typecvm.NotifyAndAutoRenew
	}

	// 数据盘
	dataDisk := make([]typecvm.TCloudDataDisk, 0)
	for _, d := range req.DataDisk {
		for i := int64(0); i < d.DiskCount; i++ {
			dataDisk = append(dataDisk, typecvm.TCloudDataDisk{
				DiskSizeGB: &d.DiskSizeGB,
				DiskType:   d.DiskType,
			})
		}
	}

	createReq := &hcproto.TCloudBatchCreateReq{
		AccountID:             req.AccountID,
		Region:                req.Region,
		Name:                  req.Name,
		Zone:                  req.Zone,
		InstanceType:          req.InstanceType,
		CloudImageID:          req.CloudImageID,
		Password:              req.Password,
		RequiredCount:         req.RequiredCount,
		CloudSecurityGroupIDs: req.CloudSecurityGroupIDs,
		CloudVpcID:            req.CloudVpcID,
		CloudSubnetID:         req.CloudSubnetID,
		InstanceChargeType:    req.InstanceChargeType,
		InstanceChargePrepaid: instanceChargePrepaid,
		SystemDisk: &typecvm.TCloudSystemDisk{
			DiskType:   req.SystemDisk.DiskType,
			DiskSizeGB: &req.SystemDisk.DiskSizeGB,
		},
		DataDisk:         dataDisk,
		PublicIPAssigned: req.PublicIPAssigned,
	}

	return createReq
}

// ConvAwsCvmCreateReq conv cvm create req.
func ConvAwsCvmCreateReq(req *cscvm.AwsCvmCreateReq) *hcproto.AwsBatchCreateReq {
	blockDeviceMapping := make([]typecvm.AwsBlockDeviceMapping, 0)
	// 系统盘
	deviceName := "/dev/sda1"
	blockDeviceMapping = append(blockDeviceMapping, typecvm.AwsBlockDeviceMapping{
		DeviceName: &deviceName,
		Ebs: &typecvm.AwsEbs{
			VolumeSizeGB: req.SystemDisk.DiskSizeGB,
			VolumeType:   req.SystemDisk.DiskType,
		},
	})

	diskStartIndex := 'a'
	// 数据盘
	for _, d := range req.DataDisk {
		for i := int64(0); i < d.DiskCount; i++ {
			blockDeviceMapping = append(blockDeviceMapping, typecvm.AwsBlockDeviceMapping{
				DeviceName: converter.ValToPtr("/dev/sd" + string(diskStartIndex+1)),
				Ebs: &typecvm.AwsEbs{
					VolumeSizeGB: d.DiskSizeGB,
					VolumeType:   d.DiskType,
				},
			})
		}
	}

	createReq := &hcproto.AwsBatchCreateReq{
		AccountID:             req.AccountID,
		Region:                req.Region,
		Zone:                  req.Zone,
		Name:                  req.Name,
		InstanceType:          req.InstanceType,
		CloudImageID:          req.CloudImageID,
		CloudSubnetID:         req.CloudSubnetID,
		PublicIPAssigned:      req.PublicIPAssigned,
		CloudSecurityGroupIDs: req.CloudSecurityGroupIDs,
		BlockDeviceMapping:    blockDeviceMapping,
		Password:              req.Password,
		RequiredCount:         req.RequiredCount,
	}

	return createReq
}

// ConvGcpCvmCreateReq conv cvm create req.
func ConvGcpCvmCreateReq(req *cscvm.GcpCvmCreateReq) *hcproto.GcpBatchCreateReq {

	dataDisk := make([]typecvm.GcpDataDisk, 0)
	// 数据盘
	for _, d := range req.DataDisk {
		for i := int64(0); i < d.DiskCount; i++ {
			dataDisk = append(dataDisk, typecvm.GcpDataDisk{
				DiskType:   d.DiskType,
				SizeGb:     d.DiskSizeGB,
				Mode:       d.Mode,
				AutoDelete: *d.AutoDelete,
			})
		}
	}
	description := ""
	if req.Memo != nil {
		description = *req.Memo
	}

	createReq := &hcproto.GcpBatchCreateReq{
		AccountID:     req.AccountID,
		NamePrefix:    req.Name,
		Region:        req.Region,
		Zone:          req.Zone,
		InstanceType:  req.InstanceType,
		CloudImageID:  req.CloudImageID,
		Password:      req.Password,
		RequiredCount: req.RequiredCount,
		CloudVpcID:    req.CloudVpcID,
		CloudSubnetID: req.CloudSubnetID,
		Description:   description,
		SystemDisk: &typecvm.GcpOsDisk{
			DiskType: req.SystemDisk.DiskType,
			SizeGb:   req.SystemDisk.DiskSizeGB,
		},
		DataDisk: dataDisk,
	}

	return createReq
}

// ConvAzureCvmCreateReq conv cvm create req.
func ConvAzureCvmCreateReq(req *cscvm.AzureCvmCreateReq) *hcproto.AzureBatchCreateReq {

	dataDisk := make([]typecvm.AzureDataDisk, 0)
	index := 1
	for _, d := range req.DataDisk {
		for i := int64(0); i < d.DiskCount; i++ {
			dataDisk = append(dataDisk, typecvm.AzureDataDisk{
				Name:   fmt.Sprintf("data%d", index),
				SizeGB: int32(d.DiskSizeGB),
				Type:   d.DiskType,
			})
			index += 1
		}
	}

	zones := make([]string, 0)
	if len(req.Zone) != 0 {
		zones = append(zones, req.Zone)
	}

	createReq := &hcproto.AzureBatchCreateReq{
		AccountID:            req.AccountID,
		ResourceGroupName:    req.ResourceGroupName,
		Region:               req.Region,
		Name:                 req.Name,
		Zones:                zones,
		InstanceType:         req.InstanceType,
		CloudImageID:         req.CloudImageID,
		Username:             req.Username,
		Password:             req.Password,
		CloudSubnetID:        req.CloudSubnetID,
		CloudSecurityGroupID: req.CloudSecurityGroupIDs[0],
		OSDisk: &typecvm.AzureOSDisk{
			Name:   "disk-" + req.Name,
			SizeGB: int32(req.SystemDisk.DiskSizeGB),
			Type:   req.SystemDisk.DiskType,
		},
		DataDisk:      dataDisk,
		RequiredCount: req.RequiredCount,
	}

	return createReq
}

// ConvHuaWeiCvmCreateReq conv cvm create req.
func ConvHuaWeiCvmCreateReq(req *cscvm.HuaWeiCvmCreateReq) *hcproto.HuaWeiBatchCreateReq {
	dataVolumes := make([]typecvm.HuaWeiVolume, 0)
	for _, d := range req.DataDisk {
		for i := int64(0); i < d.DiskCount; i++ {
			dataVolumes = append(dataVolumes, typecvm.HuaWeiVolume{
				VolumeType: d.DiskType,
				SizeGB:     int32(d.DiskSizeGB),
			})
		}
	}

	// 计费
	periodType := typecvm.Month
	periodNum := int32(req.InstanceChargePaidPeriod)
	if periodNum > 9 {
		periodType = typecvm.Year
		periodNum = int32(req.InstanceChargePaidPeriod / 12)
	}

	createReq := &hcproto.HuaWeiBatchCreateReq{
		AccountID:             req.AccountID,
		Region:                req.Region,
		Name:                  req.Name,
		Zone:                  req.Zone,
		InstanceType:          req.InstanceType,
		CloudImageID:          req.CloudImageID,
		Password:              req.Password,
		RequiredCount:         int32(req.RequiredCount),
		CloudSecurityGroupIDs: req.CloudSecurityGroupIDs,
		CloudVpcID:            req.CloudVpcID,
		CloudSubnetID:         req.CloudSubnetID,
		Description:           req.Memo,
		RootVolume: &typecvm.HuaWeiVolume{
			VolumeType: req.SystemDisk.DiskType,
			SizeGB:     int32(req.SystemDisk.DiskSizeGB),
		},
		DataVolume: dataVolumes,
		InstanceCharge: &typecvm.HuaWeiInstanceCharge{
			ChargingMode: req.InstanceChargeType,
			PeriodType:   &periodType,
			PeriodNum:    &periodNum,
			IsAutoRenew:  req.AutoRenew,
		},
	}

	return createReq
}
