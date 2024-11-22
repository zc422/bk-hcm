### 描述

- 该接口提供版本：v9.9.9+。
- 该接口所需权限：业务访问。
- 该接口功能描述：业务下同步指定账号下指定资源。

### URL

POST /api/v1/cloud/bizs/{bk_biz_id}/vendors/{vendor}/accounts/{account_id}/resources/{res}/sync_by_cond

### 输入参数

| 参数名称       | 参数类型   | 必选 | 描述                            |
|------------|--------|----|-------------------------------|
| bk_biz_id  | int    | 是  | 同步业务                          |
| vendor     | string | 是  | 云厂商，目前支持腾讯云(tcloud)           |
| account_id | string | 是  | 账号ID                          |
| res        | string | 是  | 资源名称 目前仅支持负载均衡(load_balancer) |

#### vendor=tcloud

| 参数名称        | 参数类型                | 必选 | 描述               |
|-------------|---------------------|----|------------------|
| regions     | []string            | 是  | 指定资源同步地域，最少1，最大5 |
| cloud_ids   | []string            | 否  | 资源id，数量上限20      |
| tag_filters | map[string][]string | 否  | 指定同步标签过滤器标签，上限5  |

### 调用示例

#### vendor=tcloud

1. 同步账号`00000001`在`ap-guangzhou`地域下，id 为`id-abcdefg`的资源

```json
{
  "account_id": "00000001",
  "regions": [
    "ap-guangzhou"
  ],
  "cloud_ids": [
    "id-abcdefg"
  ]
}
```

2. 同步`ap-guangzhou`地域下，标签key为`biz`value为`1234`的资源

```json
{
  "regions": [
    "ap-guangzhou"
  ],
  "tag_filters": {
    "biz": [
      "1234"
    ]
  }
}
```

### 响应示例

```json
{
  "code": 0,
  "message": "ok"
}
```

### 响应参数说明

| 参数名称    | 参数类型   | 描述   |
|---------|--------|------|
| code    | int32  | 状态码  |
| message | string | 请求信息 |