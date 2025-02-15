/*
 * TencentBlueKing is pleased to support the open source community by making
 * 蓝鲸智云 - 混合云管理平台 (BlueKing - Hybrid Cloud Management System) available.
 * Copyright (C) 2024 THL A29 Limited,
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

package bill

import (
	"errors"

	"hcm/pkg/criteria/enumor"
	"hcm/pkg/criteria/validator"

	"github.com/shopspring/decimal"
)

// BatchBillAdjustmentItemCreateReq batch create request
type BatchBillAdjustmentItemCreateReq struct {
	RootAccountID string        `json:"root_account_id" validate:"required"`
	Vendor        enumor.Vendor `json:"vendor" validate:"required"`

	Items []BillAdjustmentItemCreateReq `json:"items" validate:"required,min=1,max=100,dive,required"`
}

// Validate ...
func (r *BatchBillAdjustmentItemCreateReq) Validate() error {
	return validator.Validate.Struct(r)
}

// BillAdjustmentItemCreateReq create request
type BillAdjustmentItemCreateReq struct {
	RootAccountID string                    `json:"root_account_id" validate:"omitempty"`
	MainAccountID string                    `json:"main_account_id" validate:"required"`
	ProductID     int64                     `json:"product_id" validate:"omitempty"`
	BkBizID       int64                     `json:"bk_biz_id" validate:"omitempty"`
	BillYear      int                       `json:"bill_year" validate:"omitempty"`
	BillMonth     int                       `json:"bill_month" validate:"omitempty"`
	Type          enumor.BillAdjustmentType `json:"type" validate:"required"`
	Currency      string                    `json:"currency" validate:"required"`
	Cost          decimal.Decimal           `json:"cost" validate:"required"`
	RmbCost       decimal.Decimal           `json:"rmb_cost" validate:"required"`
	Memo          *string                   `json:"memo,omitempty"`
}

// Validate ...
func (r *BillAdjustmentItemCreateReq) Validate() error {

	if r.ProductID < 0 && r.BkBizID < 0 {
		return errors.New("both product_id and bk_biz_id are invalid")
	}
	return validator.Validate.Struct(r)
}

// BillAdjustmentItemUpdateReq update request
type BillAdjustmentItemUpdateReq struct {
	MainAccountID string                    `json:"main_account_id"`
	ProductID     int64                     `json:"product_id" validate:"omitempty"`
	BkBizID       int64                     `json:"bk_biz_id" validate:"omitempty"`
	Type          enumor.BillAdjustmentType `json:"type"`
	Currency      string                    `json:"currency"`
	Cost          *decimal.Decimal          `json:"cost"`
	RmbCost       *decimal.Decimal          `json:"rmb_cost"`
	Memo          *string                   `json:"memo"`
}

// Validate ...
func (r *BillAdjustmentItemUpdateReq) Validate() error {
	return validator.Validate.Struct(r)
}
