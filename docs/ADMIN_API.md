# Admin API 文档

Pages 管理接口文档。API 基础 URL: `/_api`

## 认证

所有 Admin API 接口都需要 **HTTP Basic Auth** 认证。

**默认凭据**

| 字段 | 值 |
|------|----|
| 用户名 | `admin` |
| 密码 | `admin` |

这些凭据在配置文件 `config.toml` 中定义。

**修改认证凭据**

可通过以下方式修改：

1. **编辑 config.toml**
   ```toml
   [server]
   admin_user = "newuser"
   admin_pass = "newpass"
   ```

2. **使用环境变量**（覆盖 config.toml 中的值）
   ```bash
   export PAGES_ADMIN_USER=newuser
   export PAGES_ADMIN_PASS=newpass
   ```

**认证方式**

在请求头中添加 `Authorization` 字段：

```
Authorization: Basic YWRtaW46YWRtaW4=
```

其中 `YWRtaW46YWRtaW4=` 是 `admin:admin` 的 Base64 编码。

或使用 curl 的 `-u` 参数（推荐）：

```bash
curl -u admin:admin http://localhost:1323/_api/sites
```

所有接口返回统一的 JSON 响应格式：

```json
{
  "success": true,
  "message": "操作说明（可选）",
  "data": {}
}
```

### 响应字段

| 字段 | 类型 | 说明 |
|------|------|------|
| success | boolean | 请求是否成功 |
| message | string | 错误或提示信息（失败时返回） |
| data | object | 响应数据（成功时返回） |

## 站点数据模型

站点对象结构：

```json
{
  "id": "default",
  "username": "default",
  "domain": "localhost",
  "index": "index.html",
  "enabled": true,
  "created_at": "2025-12-06T16:27:48.7214506+08:00",
  "updated_at": "2025-12-06T16:27:48.7214506+08:00"
}
```

### 站点字段说明

| 字段 | 类型 | 说明 |
|------|------|------|
| id | string | 站点唯一标识 |
| username | string | 租户用户名（多租户支持） |
| domain | string | 绑定域名 |
| index | string | 首页文件名（默认 index.html） |
| enabled | boolean | 是否启用 |
| created_at | string | 创建时间（ISO 8601） |
| updated_at | string | 更新时间（ISO 8601） |

## 接口列表

### 1. 获取站点列表

列出所有站点。

**请求**

```http
GET /_api/sites
```

**响应示例**

```json
{
  "success": true,
  "data": {
    "sites": [
      {
        "id": "default",
        "username": "default",
        "domain": "localhost",
        "index": "index.html",
        "enabled": true,
        "created_at": "2025-12-06T16:27:48.7214506+08:00",
        "updated_at": "2025-12-06T16:27:48.7214506+08:00"
      },
      {
        "id": "example",
        "username": "default",
        "domain": "example.localhost",
        "index": "index.html",
        "enabled": true,
        "created_at": "2025-12-06T16:27:48.7214506+08:00",
        "updated_at": "2025-12-06T16:27:48.7214506+08:00"
      }
    ],
    "total": 2
  }
}
```

**状态码**

- `200 OK` - 成功获取列表
- `500 Internal Server Error` - 服务器错误

---

### 2. 创建站点

创建新站点，路径将自动生成为 `data/sites/{username}/{id}`。

**请求**

```http
POST /_api/sites
Content-Type: application/json
```

**请求体**

```json
{
  "id": "test",
  "domain": "test.localhost",
  "username": "user1",
  "index": "index.html"
}
```

### 请求字段

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | string | 是 | 站点唯一标识 |
| domain | string | 是 | 绑定域名 |
| username | string | 否 | 租户用户名（默认为 "default"） |
| index | string | 否 | 首页文件名（默认为 "index.html"） |

**响应示例**

```json
{
  "success": true,
  "message": "站点创建成功",
  "data": {
    "id": "test",
    "username": "user1",
    "domain": "test.localhost",
    "index": "index.html",
    "enabled": true,
    "created_at": "2025-12-06T16:27:48.7214506+08:00",
    "updated_at": "2025-12-06T16:27:48.7214506+08:00"
  }
}
```

**状态码**

- `201 Created` - 站点创建成功
- `400 Bad Request` - 请求参数错误或缺少必填字段
- `409 Conflict` - 站点已存在

---

### 3. 获取单个站点

获取指定租户下的站点详情。

**请求**

```http
GET /_api/sites/:username/:id
```

**路径参数**

| 参数 | 类型 | 说明 |
|------|------|------|
| username | string | 租户用户名 |
| id | string | 站点 ID |

**响应示例**

```json
{
  "success": true,
  "data": {
    "id": "default",
    "username": "default",
    "domain": "localhost",
    "index": "index.html",
    "enabled": true,
    "created_at": "2025-12-06T16:27:48.7214506+08:00",
    "updated_at": "2025-12-06T16:27:48.7214506+08:00"
  }
}
```

**状态码**

- `200 OK` - 成功获取站点
- `404 Not Found` - 站点不存在

---

### 4. 更新站点

更新指定租户下的站点配置信息。

**请求**

```http
PUT /_api/sites/:username/:id
Content-Type: application/json
```

**路径参数**

| 参数 | 类型 | 说明 |
|------|------|------|
| username | string | 租户用户名 |
| id | string | 站点 ID |

**请求体**

```json
{
  "domain": "newdomain.localhost",
  "index": "home.html",
  "enabled": false
}
```

### 请求字段

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| domain | string | 否 | 新的绑定域名 |
| index | string | 否 | 新的首页文件名 |
| enabled | boolean | 否 | 是否启用站点 |

**响应示例**

```json
{
  "success": true,
  "message": "站点更新成功",
  "data": {
    "id": "blog",
    "username": "tenant1",
    "domain": "newdomain.localhost",
    "index": "home.html",
    "enabled": false,
    "created_at": "2025-12-06T16:27:48.7214506+08:00",
    "updated_at": "2025-12-06T16:27:48.7214506+08:00"
  }
}
```

**状态码**

- `200 OK` - 更新成功
- `400 Bad Request` - 请求参数错误
- `404 Not Found` - 站点不存在
- `500 Internal Server Error` - 服务器错误

---

### 5. 删除站点

删除指定租户下的站点。

**请求**

```http
DELETE /_api/sites/:username/:id
```

**路径参数**

| 参数 | 类型 | 说明 |
|------|------|------|
| username | string | 租户用户名 |
| id | string | 站点 ID |

**响应示例**

```json
{
  "success": true,
  "message": "站点删除成功"
}
```

**状态码**

- `200 OK` - 删除成功
- `404 Not Found` - 站点不存在

---

### 6. 热重载

重新加载所有站点配置。

**请求**

```http
POST /_api/reload
```

**响应示例**

```json
{
  "success": true,
  "message": "重载成功，当前 2 个站点已生效",
  "data": {
    "sites_count": 2,
    "reloaded_at": "2025-12-06T16:27:48.7214506+08:00"
  }
}
```

**状态码**

- `200 OK` - 重载成功
- `500 Internal Server Error` - 服务器错误

---

### 7. 健康检查

检查服务器健康状态。

**请求**

```http
GET /_api/health
```

**响应示例**

```json
{
  "success": true,
  "data": {
    "status": "healthy",
    "sites_count": 2,
    "timestamp": "2025-12-06T16:27:48.7214506+08:00"
  }
}
```

**状态码**

- `200 OK` - 服务器健康

---

### 8. 一键部署站点

上传 zip 或 tar.gz 压缩包，自动清空并替换站点根目录。

**请求**

```http
POST /_api/sites/:username/:id/deploy
Content-Type: multipart/form-data
```

**路径参数**

| 参数 | 类型 | 说明 |
|------|------|------|
| username | string | 租户用户名 |
| id | string | 站点 ID |

**表单字段**

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| file | file | 是 | zip 或 tar.gz 压缩包 |

**行为说明**

- 清空站点根目录后解压上传内容
- 仅允许 zip、tar.gz、tgz
- 拒绝压缩包内的符号链接，防止路径遍历
- 解压路径限定在站点根目录内

**成功响应示例**

```json
{
  "success": true,
  "message": "站点已部署",
  "data": {
    "username": "tenant1",
    "id": "blog"
  }
}
```

**错误响应示例**

```json
{
  "success": false,
  "message": "解压 zip 失败: ..."
}
```

**状态码**

- `200 OK` - 部署成功
- `400 Bad Request` - 文件缺失、格式不支持或解压失败
- `404 Not Found` - 站点不存在
- `500 Internal Server Error` - 服务器内部错误

**使用示例**

```bash
curl -u admin:admin -X POST \
  -F "file=@./dist.zip" \
  http://localhost:1323/_api/sites/tenant1/blog/deploy
```

---

## 使用示例

### 创建新租户的站点

```bash
curl -u admin:admin -X POST http://localhost:1323/_api/sites \
  -H "Content-Type: application/json" \
  -d '{
    "id": "blog",
    "domain": "blog.mycompany.com",
    "username": "tenant1",
    "index": "index.html"
  }'
```

自动生成的路径：`data/sites/tenant1/blog`

### 创建默认租户的站点

```bash
curl -u admin:admin -X POST http://localhost:1323/_api/sites \
  -H "Content-Type: application/json" \
  -d '{
    "id": "docs",
    "domain": "docs.localhost"
  }'
```

自动生成的路径：`data/sites/default/docs`

### 获取所有站点

```bash
curl -u admin:admin http://localhost:1323/_api/sites
```

### 更新站点

```bash
curl -u admin:admin -X PUT http://localhost:1323/_api/sites/tenant1/blog \
  -H "Content-Type: application/json" \
  -d '{
    "enabled": false,
    "index": "home.html"
  }'
```

---

## 多租户支持

系统支持完整的多租户架构，每个租户可以拥有多个站点。

### 设计特点

- **命名空间隔离**：使用 `/:username/:id` 的路由形式，实现完整的租户隔离
- **自动路径生成**：站点文件存储在 `data/sites/{username}/{id}` 目录
- **租户独立管理**：每个租户可以独立创建、更新、删除自己的站点
- **默认租户**：不指定 `username` 时自动使用 "default" 租户

### 单租户示例

所有站点都属于默认租户 "default"：

```bash
# 创建站点
curl -u admin:admin -X POST http://localhost:1323/_api/sites \
  -H "Content-Type: application/json" \
  -d '{
    "id": "blog",
    "domain": "blog.localhost"
  }'

# 获取站点（使用默认租户）
curl -u admin:admin http://localhost:1323/_api/sites/default/blog

# 更新站点
curl -u admin:admin -X PUT http://localhost:1323/_api/sites/default/blog \
  -H "Content-Type: application/json" \
  -d '{"domain": "newblog.localhost"}'

# 删除站点
curl -u admin:admin -X DELETE http://localhost:1323/_api/sites/default/blog
```

### 多租户示例

不同租户拥有独立的站点：

```bash
# 创建租户 user1 的站点
curl -u admin:admin -X POST http://localhost:1323/_api/sites \
  -H "Content-Type: application/json" \
  -d '{
    "id": "blog",
    "domain": "blog.user1.com",
    "username": "user1"
  }'

# 创建租户 user2 的站点（相同的 ID 不会冲突）
curl -u admin:admin -X POST http://localhost:1323/_api/sites \
  -H "Content-Type: application/json" \
  -d '{
    "id": "blog",
    "domain": "blog.user2.com",
    "username": "user2"
  }'

# 分别获取不同租户的站点
curl -u admin:admin http://localhost:1323/_api/sites/user1/blog
curl -u admin:admin http://localhost:1323/_api/sites/user2/blog

# 站点存储位置
# user1: data/sites/user1/blog
# user2: data/sites/user2/blog
```

### 租户隔离效果

```json
{
  "user1": {
    "blog": { "id": "blog", "username": "user1", "domain": "blog.user1.com", "path": "data/sites/user1/blog" }
  },
  "user2": {
    "blog": { "id": "blog", "username": "user2", "domain": "blog.user2.com", "path": "data/sites/user2/blog" }
  },
  "default": {
    "blog": { "id": "blog", "username": "default", "domain": "blog.localhost", "path": "data/sites/default/blog" },
    "docs": { "id": "docs", "username": "default", "domain": "docs.localhost", "path": "data/sites/default/docs" }
  }
}
```

不同租户的站点存储在不同的目录中，实现了完整的数据隔离和租户隔离。

---

## 检查点管理

检查点功能提供站点版本管理和回滚能力。

### 检查点设计

- 每个站点使用**集中式** `metadata.json` 管理所有检查点
- 仅在**部署**时自动创建检查点
- **切换检查点**时不创建新检查点，仅更新 `current` 指针
- 不允许删除当前激活的检查点

### 检查点元数据结构

每个站点的 `metadata.json` 包含：

```json
{
  "site_id": "blog",
  "username": "default",
  "current": "20250106-160000-c3d4e5f6",
  "checkpoints": [
    {
      "id": "20250106-160000-c3d4e5f6",
      "created_at": "2025-01-06T16:00:00+08:00",
      "file_size": 1024000,
      "file_name": "site-v2.0.zip",
      "source": "deploy",
      "description": "部署: site-v2.0.zip"
    },
    {
      "id": "20250106-120000-b2c3d4e5",
      "created_at": "2025-01-06T12:00:00+08:00",
      "file_size": 950000,
      "file_name": "site-v1.0.zip",
      "source": "deploy",
      "description": "部署: site-v1.0.zip"
    }
  ],
  "updated_at": "2025-01-06T16:00:00+08:00"
}
```

### 检查点字段说明

| 字段 | 类型 | 说明 |
|------|------|------|
| id | string | 检查点唯一标识（时间戳-哈希） |
| created_at | string | 创建时间（ISO 8601） |
| file_size | int64 | 备份文件大小（字节） |
| file_name | string | 原始上传文件名 |
| source | string | 来源（"deploy" 或 "manual"） |
| description | string | 描述信息 |

### 9. 列出检查点

获取指定站点的所有检查点及当前激活的检查点。

**端点**

```
GET /_api/sites/:username/:id/checkpoints
```

**路径参数**

| 参数 | 类型 | 说明 |
|------|------|------|
| username | string | 租户用户名 |
| id | string | 站点 ID |

**响应示例**

```json
{
  "success": true,
  "data": {
    "current": "20250106-160000-c3d4e5f6",
    "checkpoints": [
      {
        "id": "20250106-160000-c3d4e5f6",
        "created_at": "2025-01-06T16:00:00+08:00",
        "file_size": 1024000,
        "file_name": "site-v2.0.zip",
        "source": "deploy",
        "description": "部署: site-v2.0.zip"
      },
      {
        "id": "20250106-120000-b2c3d4e5",
        "created_at": "2025-01-06T12:00:00+08:00",
        "file_size": 950000,
        "file_name": "site-v1.0.zip",
        "source": "deploy",
        "description": "部署: site-v1.0.zip"
      }
    ],
    "total": 2
  }
}
```

**示例**

```bash
curl -u admin:admin http://localhost:1323/_api/sites/default/blog/checkpoints
```

### 10. 获取检查点详情

获取指定检查点的详细信息。

**端点**

```
GET /_api/sites/:username/:id/checkpoints/:checkpoint_id
```

**路径参数**

| 参数 | 类型 | 说明 |
|------|------|------|
| username | string | 租户用户名 |
| id | string | 站点 ID |
| checkpoint_id | string | 检查点 ID |

**响应示例**

```json
{
  "success": true,
  "data": {
    "id": "20250106-153045-a1b2c3d4",
    "created_at": "2025-01-06T15:30:45+08:00",
    "file_size": 1048576,
    "file_name": "site-v1.0.zip",
    "source": "deploy",
    "description": "部署: site-v1.0.zip"
  }
}
```

**示例**

```bash
curl -u admin:admin http://localhost:1323/_api/sites/default/blog/checkpoints/20250106-153045-a1b2c3d4
```

### 11. 删除检查点

删除指定的检查点备份。**注意：不能删除当前激活的检查点。**

**端点**

```
DELETE /_api/sites/:username/:id/checkpoints/:checkpoint_id
```

**路径参数**

| 参数 | 类型 | 说明 |
|------|------|------|
| username | string | 租户用户名 |
| id | string | 站点 ID |
| checkpoint_id | string | 检查点 ID |

**响应示例（成功）**

```json
{
  "success": true,
  "message": "检查点已删除"
}
```

**响应示例（错误：尝试删除当前检查点）**

```json
{
  "success": false,
  "message": "不能删除当前激活的检查点"
}
```

**示例**

```bash
curl -u admin:admin -X DELETE http://localhost:1323/_api/sites/default/blog/checkpoints/20250106-120000-b2c3d4e5
```

### 12. 切换检查点

将站点切换到指定检查点版本。**仅切换 current 指针，不创建新检查点。**

**端点**

```
POST /_api/sites/:username/:id/checkpoints/:checkpoint_id/checkout
```

**路径参数**

| 参数 | 类型 | 说明 |
|------|------|------|
| username | string | 租户用户名 |
| id | string | 站点 ID |
| checkpoint_id | string | 检查点 ID |

**响应示例**

```json
{
  "success": true,
  "message": "站点已切换到检查点",
  "data": {
    "username": "default",
    "id": "blog",
    "checkpoint_id": "20250106-120000-b2c3d4e5"
  }
---

## 检查点工作流示例

### 自动检查点创建

每次部署时会自动创建检查点：

```bash
# 部署站点（自动创建检查点）
curl -u admin:admin -X POST http://localhost:1323/_api/sites/default/blog/deploy \
  -F "file=@site-v2.0.zip"

# 响应
{
  "success": true,
  "message": "站点已部署",
  "data": {
    "username": "default",
    "id": "blog",
    "checkpoint": {
      "id": "20250106-160000-c3d4e5f6",
      "created_at": "2025-01-06T16:00:00+08:00",
      "file_size": 1024000,
      "file_name": "site-v2.0.zip",
      "source": "deploy",
      "description": "部署: site-v2.0.zip"
    }
  }
}
```

### 版本回滚流程

```bash
# 1. 查看所有检查点和当前激活版本
curl -u admin:admin http://localhost:1323/_api/sites/default/blog/checkpoints
# 返回:
# {
#   "current": "20250106-160000-c3d4e5f6",  # 当前是 v2.0
#   "checkpoints": [...]
# }

# 2. 切换到旧版本（仅切换指针，不创建新检查点）
curl -u admin:admin -X POST \
  http://localhost:1323/_api/sites/default/blog/checkpoints/20250106-120000-b2c3d4e5/checkout

# 3. 验证恢复结果
curl http://blog.localhost

# 4. 查看当前激活版本
curl -u admin:admin http://localhost:1323/_api/sites/default/blog/checkpoints
# 返回:
# {
#   "current": "20250106-120000-b2c3d4e5",  # 现在是 v1.0
#   "checkpoints": [...]  # 检查点列表不变
# }

# 5. 如需再次切换回 v2.0
curl -u admin:admin -X POST \
  http://localhost:1323/_api/sites/default/blog/checkpoints/20250106-160000-c3d4e5f6/checkout
```

### 清理旧检查点

```bash
# 列出所有检查点
curl -u admin:admin http://localhost:1323/_api/sites/default/blog/checkpoints

# 删除不需要的旧版本（注意：不能删除 current 指向的检查点）
curl -u admin:admin -X DELETE \
  http://localhost:1323/_api/sites/default/blog/checkpoints/20250101-100000-old12345
```

---

## 检查点存储

- 检查点存储目录: `data/sites-checkpoints/{username}/{site_id}/`
- 每个站点包含：
  - `metadata.json` - 集中式元数据文件（包含所有检查点信息和 current 指针）
  - `checkpoints/` - 检查点备份文件目录
    - `{checkpoint_id}.tar.gz` - 站点备份文件
- 检查点 ID 格式: `{时间戳}-{内容哈希前8位}`
- 备份文件使用 tar.gz 格式压缩

**目录结构示例**

```
data/sites-checkpoints/
├── default/
│   ├── blog/
│   │   ├── metadata.json              # 集中式元数据
│   │   └── checkpoints/
│   │       ├── 20250106-153045-a1b2c3d4.tar.gz
│   │       └── 20250106-120000-b2c3d4e5.tar.gz
│   └── docs/
│       ├── metadata.json
│       └── checkpoints/
│           └── 20250106-140000-e5f6g7h8.tar.gz
└── user1/
    └── blog/
        ├── metadata.json
        └── checkpoints/
            └── 20250106-170000-f6g7h8i9.tar.gz
```

**metadata.json 示例**

```json
{
  "site_id": "blog",
  "username": "default",
  "current": "20250106-153045-a1b2c3d4",
  "checkpoints": [
    {
      "id": "20250106-153045-a1b2c3d4",
      "created_at": "2025-01-06T15:30:45+08:00",
      "file_size": 1048576,
      "file_name": "site-v2.0.zip",
      "source": "deploy",
      "description": "部署: site-v2.0.zip"
    },
    {
      "id": "20250106-120000-b2c3d4e5",
      "created_at": "2025-01-06T12:00:00+08:00",
      "file_size": 950000,
      "file_name": "site-v1.0.zip",
      "source": "deploy",
      "description": "部署: site-v1.0.zip"
    }
  ],
  "updated_at": "2025-01-06T15:30:45+08:00"
}
``` └── docs/
│       └── checkpoints/
│           ├── 20250106-140000-e5f6g7h8.tar.gz
│           └── 20250106-140000-e5f6g7h8.json
└── user1/
    └── blog/
        └── checkpoints/
            ├── 20250106-170000-f6g7h8i9.tar.gz
            └── 20250106-170000-f6g7h8i9.json
```

