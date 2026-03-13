# ITDB — IT 资产管理系统

一个面向企业 IT 基础设施的资产管理系统，支持硬件设备、软件许可、合同、单据、文件、机架、地点等资产的全生命周期管理。前后端分离架构，由 PHP 版 ITDB 重构为 Go + Vue3。

# 一、项目特性

- 前后端分离：`Go + SQLite` 后端，`Vue3 + Vite + TypeScript` 前端
- 纯 Go 实现：SQLite 驱动使用 `modernc.org/sqlite`，无 CGO 依赖，交叉编译友好
- 双认证模式：支持本地密码和 LDAP 两种登录方式
- 权限控制：完全访问 / 只读两级权限
- 操作审计：所有写操作记录到历史日志
- 自动备份：每日 0 点自动 VACUUM INTO 备份数据库，Schema 变更前也会自动备份
- 会话管理：JWT 认证，前端空闲 1 小时自动登出
- 标签打印：支持 QR 码生成、多种标签纸预设
- 机架可视化：独立机架视图页面
- 数据库导入：支持从旧系统 .db 文件直接导入替换
- 错误信息中文本地化

# 二、技术栈

## 2.1 后端

- Go（以 `backend/go.mod` 为准）
- SQLite（`modernc.org/sqlite`，纯 Go 实现）
- go-chi/chi v5（HTTP 路由）
- golang-jwt/jwt v5（JWT 认证）
- go-ldap/ldap v3（LDAP 认证）
- xuri/excelize v2（Excel 导出）
- mozillazg/go-pinyin（拼音转换）
- golang.org/x/crypto（密码加密）

## 2.2 前端

- Vue 3
- Vite
- TypeScript
- Pinia（状态管理）
- Vue Router
- Axios（HTTP 客户端）
- dayjs（日期处理）
- qrcode（二维码生成）

## 2.3 目录结构

```text
itdb
├─ backend
│  ├─ main.go                        # 入口
│  ├─ .env.example                   # 环境变量模板
│  ├─ go.mod / go.sum
│  ├─ data
│  │  ├─ itdb.db                     # SQLite 数据库（运行时生成）
│  │  ├─ files/                      # 上传文件存储
│  │  └─ backups/                    # 自动备份目录
│  ├─ scripts
│  │  ├─ backup_db.ps1               # PowerShell 备份脚本
│  │  └─ backup_db.sh                # Bash 备份脚本
│  └─ cmd
│     ├─ server/                     # 核心服务代码
│     └─ common/                     # 公共模块（本地化、工具函数）
├─ frontend
│  ├─ index.html
│  ├─ vite.config.ts
│  ├─ package.json
│  └─ src
│     ├─ api/                        # Axios 封装
│     ├─ assets/styles/              # 全局样式
│     ├─ components/                 # 公共组件
│     ├─ composables/                # 组合式函数
│     ├─ layouts/                    # 布局组件
│     ├─ pages/                      # 页面组件
│     ├─ router/                     # 路由定义
│     └─ stores/                     # Pinia 状态管理
├─ doc/                              # 项目文档
├─ .gitignore
└─ README.md
```

# 三、功能清单

## 3.1 资源管理

| 模块 | 说明 |
|------|------|
| 硬件资产 (Items) | 服务器、网络设备、PC 等硬件的全生命周期管理，支持 SN、IP、机架位置、关联发票/合同/文件 |
| 软件许可 (Software) | 软件许可证管理，支持许可数量、类型、版本、关联发票 |
| 合同 (Contracts) | 合同管理，支持合同类型/子类型、续签记录、关联硬件/软件/发票/文件 |
| 发票 (Invoices) | 发票管理，支持供应商/采购方、关联硬件/软件/合同/文件 |
| 文件 (Files) | 附件上传与管理，支持多种文件类型，可关联到硬件/软件/合同/发票 |
| 厂商/代理商 (Agents) | 供应商和代理商信息管理 |

## 3.2 基础设施

| 模块 | 说明 |
|------|------|
| 位置 (Locations) | 机房/楼层管理，支持平面图上传和热区标注 |
| 机柜 (Racks) | 机柜管理，支持 U 位可视化、正反面视图 |
| 标签打印 (Labels) | QR 码标签生成，支持多种标签纸预设、批量打印 |

## 3.3 字典与分类

| 模块 | 说明 |
|------|------|
| 硬件类型 (Item Types) | 硬件资产分类字典 |
| 合同类型 (Contract Types) | 合同分类及子类型字典 |
| 部门 (Departments) | 部门字典 |
| 状态 (Status Types) | 资产状态字典，支持自定义颜色 |
| 文件类型 (File Types) | 文件分类字典 |
| 标签 (Tags) | 自由标签，可关联硬件和软件 |

## 3.4 系统功能

| 模块 | 说明 |
|------|------|
| 认证 | 本地密码 + LDAP 双模式登录，JWT 48 小时有效期 |
| 权限 | 管理员（完全访问）/ 普通用户（只读）两级权限 |
| 操作历史 | 所有写操作自动记录，支持导出 Excel |
| 浏览历史 | 最近查看记录 |
| 仪表盘 | 资产统计概览 |
| 报表 | 内置多种统计报表 |
| 数据库导入 | 支持从旧系统 .db 文件直接导入替换 |
| 自动备份 | 每日 0 点自动备份数据库，Schema 变更前也会自动备份 |
| 数据库/全量备份下载 | 支持在线下载数据库备份和全量备份（含上传文件） |
| 空闲登出 | 前端空闲 1 小时自动跳转登录页 |

# 四、环境要求

| 依赖 | 版本要求 |
|------|----------|
| Go | >= 1.21 |
| Node.js | >= 18 |
| npm | >= 9 |
| Git | >= 2.x |

> 后端使用纯 Go SQLite 驱动（`modernc.org/sqlite`），无需安装 GCC 或 CGO 环境。

# 五、本地开发快速启动

## 5.1 克隆项目

```bash
git clone https://github.com/<your-username>/itdb.git
cd itdb
```

## 5.2 启动后端

```bash
cd backend

# 复制环境变量模板并按需修改
cp .env.example .env

# 下载依赖
go mod download

# 运行
go run ./cmd/server
```

后端默认监听 `http://127.0.0.1:8080`。

环境变量说明（`.env`）：

| 变量 | 默认值 | 说明 |
|------|--------|------|
| `ITDB_SERVER_ADDR` | `:8080` | 监听地址 |
| `ITDB_DB_PATH` | `data/itdb.db` | SQLite 数据库路径 |
| `ITDB_UPLOAD_DIR` | `data/files` | 上传文件存储目录 |
| `ITDB_JWT_SECRET` | 随机生成 | JWT 签名密钥，生产环境务必设置 |
| `ITDB_HISTORY_LIMIT` | `2000` | 操作历史保留条数 |
| `ITDB_CORS_ORIGINS` | 空 | 允许的跨域来源，多个用逗号分隔 |

> 首次启动会自动创建数据库和默认管理员账户 `admin / admin123`。

## 5.3 启动前端

```bash
cd frontend

# 安装依赖
npm install

# 启动开发服务器
npm run dev
```

前端默认运行在 `http://127.0.0.1:3000`，自动将 `/api` 代理到后端 `http://127.0.0.1:8080`。

可通过 `VITE_API_BASE` 环境变量修改后端地址。

## 5.4 访问系统

浏览器打开 `http://127.0.0.1:3000`，使用默认账户登录：

- 用户名：`admin`
- 密码：`admin123`

# 六、生产环境部署

## 6.1 构建后端

```bash
cd backend

# Linux
GOOS=linux GOARCH=amd64 go build -o server ./cmd/server

# Windows
GOOS=windows GOARCH=amd64 go build -o server.exe ./cmd/server
```

将编译产物 `server`（或 `server.exe`）和 `data/` 目录部署到服务器，配置 `.env` 后直接运行即可。

## 6.2 构建前端

```bash
cd frontend
npm install
npm run build
```

构建产物在 `frontend/dist/` 目录，部署为静态文件。

## 6.3 Nginx 反向代理配置

### HTTP 配置

```nginx
server {
    listen       80;
    server_name  itdb.example.com;

    # 前端静态文件
    location / {
        root   /opt/itdb/frontend/dist;
        index  index.html;
        try_files $uri $uri/ /index.html;
    }

    # 后端 API 反向代理
    location /api/ {
        proxy_pass http://127.0.0.1:8080;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;

        # 文件上传大小限制
        client_max_body_size 100m;

        # 数据库导入超时设置
        proxy_read_timeout 300s;
        proxy_send_timeout 300s;
    }

    # 健康检查
    location /health {
        proxy_pass http://127.0.0.1:8080;
    }
}
```

### HTTPS 配置

```nginx
server {
    listen       80;
    server_name  itdb.example.com;
    return 301   https://$host$request_uri;
}

server {
    listen       443 ssl http2;
    server_name  itdb.example.com;

    ssl_certificate     /etc/nginx/ssl/itdb.example.com.pem;
    ssl_certificate_key /etc/nginx/ssl/itdb.example.com.key;
    ssl_protocols       TLSv1.2 TLSv1.3;
    ssl_ciphers         HIGH:!aNULL:!MD5;

    # 前端静态文件
    location / {
        root   /opt/itdb/frontend/dist;
        index  index.html;
        try_files $uri $uri/ /index.html;
    }

    # 后端 API 反向代理
    location /api/ {
        proxy_pass http://127.0.0.1:8080;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;

        client_max_body_size 100m;
        proxy_read_timeout 300s;
        proxy_send_timeout 300s;
    }

    location /health {
        proxy_pass http://127.0.0.1:8080;
    }
}
```

# 七、API 接口总览

所有接口以 `/api` 为前缀，除登录接口外均需在请求头携带 `Authorization: Bearer <token>`。

写操作接口（POST / PUT / DELETE）需要管理员权限，只读用户仅可访问 GET 接口。

| 模块 | 前缀 | 主要操作 |
|------|------|----------|
| 认证 | `/api/auth` | 登录、登出、获取当前用户 |
| 硬件资产 | `/api/items` | CRUD、操作记录、标签关联 |
| 软件许可 | `/api/software` | CRUD、标签关联 |
| 合同 | `/api/contracts` | CRUD、合同事件 |
| 发票 | `/api/invoices` | CRUD |
| 文件 | `/api/files` | CRUD、上传下载 |
| 厂商/代理商 | `/api/agents` | CRUD |
| 用户 | `/api/users` | CRUD |
| 位置 | `/api/locations` | CRUD、区域管理、平面图 |
| 机柜 | `/api/racks` | CRUD |
| 字典 | `/api/dictionaries` | 增删改（硬件类型、合同类型、部门、状态等） |
| 标签 | `/api/tags` | CRUD、建议、关联查询 |
| 报表 | `/api/reports` | 列表、执行 |
| 标签打印 | `/api/labels` | 资产列表、预设管理、预览 |
| 浏览树 | `/api/browse/tree` | 树形结构浏览 |
| 仪表盘 | `/api/dashboard/summary` | 统计概览 |
| 操作历史 | `/api/history` | 列表、导出 Excel |
| 浏览历史 | `/api/view-history` | 列表、记录 |
| 系统设置 | `/api/settings` | 获取、更新、LDAP 测试 |
| 备份下载 | `/api/backups` | 数据库备份、全量备份下载 |
| 数据库导入 | `/api/import/database` | 上传 .db 文件替换当前数据库 |
| 健康检查 | `/health`、`/api/health` | 服务状态 |

# 八、数据库说明

使用 SQLite 单文件数据库，默认路径 `backend/data/itdb.db`，共 33 张表。

## 8.1 核心业务表

| 表名 | 说明 |
|------|------|
| `items` | 硬件资产（核心表，含 SN、IP、机架位置、CPU/RAM/HD 等字段） |
| `software` | 软件许可证 |
| `contracts` | 合同 |
| `invoices` | 发票 |
| `files` | 文件附件 |
| `agents` | 厂商/代理商 |
| `users` | 系统用户 |
| `locations` | 位置/机房 |
| `racks` | 机柜 |
| `tags` | 标签 |
| `actions` | 硬件操作记录 |
| `contractevents` | 合同事件 |

## 8.2 关联表

| 表名 | 说明 |
|------|------|
| `item2inv` | 硬件 ↔ 发票 |
| `item2soft` | 硬件 ↔ 软件（含安装日期） |
| `item2file` | 硬件 ↔ 文件 |
| `itemlink` | 硬件 ↔ 硬件互联 |
| `contract2item` | 合同 ↔ 硬件 |
| `contract2soft` | 合同 ↔ 软件 |
| `contract2inv` | 合同 ↔ 发票 |
| `contract2file` | 合同 ↔ 文件 |
| `invoice2file` | 发票 ↔ 文件 |
| `soft2inv` | 软件 ↔ 发票 |
| `software2file` | 软件 ↔ 文件 |
| `tag2item` | 标签 ↔ 硬件 |
| `tag2software` | 标签 ↔ 软件 |

## 8.3 字典表

| 表名 | 说明 |
|------|------|
| `itemtypes` | 硬件类型 |
| `contracttypes` | 合同类型 |
| `contractsubtypes` | 合同子类型 |
| `dpttypes` | 部门 |
| `statustypes` | 资产状态（含颜色） |
| `filetypes` | 文件类型 |

## 8.4 系统表

| 表名 | 说明 |
|------|------|
| `settings` | 系统设置（LDAP 配置等，单行表） |
| `history` | 操作审计日志 |
| `viewhist` | 浏览历史 |
| `labelpapers` | 标签纸预设 |
| `locareas` | 位置区域（平面图热区） |

# 九、常见问题

**Q: 忘记管理员密码怎么办？**

删除数据库文件 `backend/data/itdb.db`，重启后端服务会自动创建新数据库和默认 `admin / admin123` 账户。或者导入一个新的 .db 文件，如果导入的数据库无用户，系统会自动创建 admin 账户。

**Q: 如何修改 JWT 有效期？**

当前 JWT 有效期为 48 小时，硬编码在后端代码中。如需修改，编辑 `backend/cmd/server/handlers_auth.go` 中的 `tokenExpiry` 常量。

**Q: 数据库文件可以直接复制迁移吗？**

可以。SQLite 是单文件数据库，停止后端服务后直接复制 `itdb.db` 文件即可完成迁移。也可以通过系统内置的数据库导入功能在线替换。

**Q: 如何从旧版 PHP ITDB 迁移？**

旧版 PHP ITDB 同样使用 SQLite 数据库，可通过系统的「数据库导入」功能直接上传旧版 .db 文件进行替换。系统会自动执行 Schema 迁移。

**Q: 自动备份存储在哪里？**

自动备份存储在 `backend/data/backups/` 目录，命名格式为 `itdb-YYYYMMDD.db`，每天 0 点自动执行。

**Q: 上传文件大小有限制吗？**

后端默认无大小限制，但如果使用 Nginx 反向代理，需要配置 `client_max_body_size`（参考上方 Nginx 配置示例）。

# 十、安全建议

1. **修改默认密码**：首次部署后立即修改 `admin` 账户的默认密码
2. **设置 JWT 密钥**：生产环境务必在 `.env` 中设置 `ITDB_JWT_SECRET`，避免使用随机生成的临时密钥
3. **启用 HTTPS**：生产环境建议通过 Nginx 配置 SSL 证书，启用 HTTPS 访问
4. **限制访问来源**：通过 Nginx 或防火墙限制系统的访问 IP 范围
5. **定期备份**：虽然系统已有每日自动备份，建议额外配置异地备份策略
6. **文件目录权限**：确保 `backend/data/` 目录权限合理，避免非授权访问数据库和上传文件
7. **环境变量安全**：`.env` 文件包含敏感信息，确保不被提交到版本控制（已在 `.gitignore` 中排除）
8. **CORS 配置**：生产环境按需配置 `ITDB_CORS_ORIGINS`，避免设置为 `*`
