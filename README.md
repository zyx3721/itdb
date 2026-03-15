# ITDB — IT 资产管理系统

基于 [sivann/itdb](https://github.com/sivann/itdb)（PHP + SQLite）重构的 IT 资产管理系统，使用 Go + Vue3 前后端分离架构重新实现。支持硬件设备、软件许可、合同、单据、文件、机架、地点等资产的全生命周期管理。

登录页面左侧动画效果借鉴了 [Animated Characters Login Page](https://21st.dev/community/components/aghasisahakyan1/animated-characters-login-page)。

# 一、项目截图

## 1.1 登录页

![image-20260313174118765](https://raw.githubusercontent.com/zyx3721/Picbed/main/blog-images/2026/03/13/74587552586a79ffd81ec863b2a03dc0-image-20260313174118765-844435.png)

## 1.2 首页

![image-20260313174152051](https://raw.githubusercontent.com/zyx3721/Picbed/main/blog-images/2026/03/13/549633c97062e2df1facc6585ee98b2f-image-20260313174152051-bbb467.png)



# 二、项目特性

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

# 三、技术栈

## 3.1 后端

- Go（以 `backend/go.mod` 为准）
- SQLite（`modernc.org/sqlite`，纯 Go 实现）
- go-chi/chi v5（HTTP 路由）
- golang-jwt/jwt v5（JWT 认证）
- go-ldap/ldap v3（LDAP 认证）
- xuri/excelize v2（Excel 导出）
- mozillazg/go-pinyin（拼音转换）
- golang.org/x/crypto（密码加密）

## 3.2 前端

- Vue 3
- Vite
- TypeScript
- Pinia（状态管理）
- Vue Router
- Axios（HTTP 客户端）
- dayjs（日期处理）
- qrcode（二维码生成）

## 3.3 目录结构

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

# 四、功能清单

## 4.1 资源管理

| 模块 | 说明 |
|------|------|
| 硬件资产 (Items) | 服务器、网络设备、PC 等硬件的全生命周期管理，支持 SN、IP、机架位置、关联发票/合同/文件 |
| 软件许可 (Software) | 软件许可证管理，支持许可数量、类型、版本、关联发票 |
| 合同 (Contracts) | 合同管理，支持合同类型/子类型、续签记录、关联硬件/软件/发票/文件 |
| 发票 (Invoices) | 发票管理，支持供应商/采购方、关联硬件/软件/合同/文件 |
| 文件 (Files) | 附件上传与管理，支持多种文件类型，可关联到硬件/软件/合同/发票 |
| 厂商/代理商 (Agents) | 供应商和代理商信息管理 |

## 4.2 基础设施

| 模块 | 说明 |
|------|------|
| 位置 (Locations) | 机房/楼层管理，支持平面图上传和热区标注 |
| 机柜 (Racks) | 机柜管理，支持 U 位可视化、正反面视图 |
| 标签打印 (Labels) | QR 码标签生成，支持多种标签纸预设、批量打印 |

## 4.3 字典与分类

| 模块 | 说明 |
|------|------|
| 硬件类型 (Item Types) | 硬件资产分类字典 |
| 合同类型 (Contract Types) | 合同分类及子类型字典 |
| 部门 (Departments) | 部门字典 |
| 状态 (Status Types) | 资产状态字典，支持自定义颜色 |
| 文件类型 (File Types) | 文件分类字典 |
| 标签 (Tags) | 自由标签，可关联硬件和软件 |

## 4.4 系统功能

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

# 五、环境要求

| 依赖 | 版本要求 |
|------|----------|
| Go | >= 1.24 |
| Node.js | >= 20 |
| npm | >= 9 |
| Git | >= 2.x |

> 后端使用纯 Go SQLite 驱动（`modernc.org/sqlite`），无需安装 GCC 或 CGO 环境。

# 六、本地开发快速启动

## 6.1 克隆项目

```bash
git clone https://github.com/zyx3721/itdb.git
cd itdb
```

## 6.2 后端配置与启动

1. 进入后端目录下载相关依赖：

```bash
cd backend
go mod tidy
```

2. 配置环境变量：

```bash
# 步骤1：复制模板文件
cp .env.example .env

# 步骤2：编辑 .env，按实际环境修改监听地址、密钥等信息
# 后端监听地址
ITDB_SERVER_ADDR=127.0.0.1:8080

# 数据库与上传目录
ITDB_DB_PATH=./data/itdb.db
ITDB_UPLOAD_DIR=./data/files

# 鉴权与接口行为
ITDB_JWT_SECRET=itdb-change-me
ITDB_HISTORY_LIMIT=1000
ITDB_CORS_ORIGINS=*
```

环境变量说明：

| 变量                 | 默认值           | 说明                           |
| -------------------- | ---------------- | ------------------------------ |
| `ITDB_SERVER_ADDR`   | `127.0.0.1:8080` | 监听地址                       |
| `ITDB_DB_PATH`       | `data/itdb.db`   | SQLite 数据库路径              |
| `ITDB_UPLOAD_DIR`    | `data/files`     | 上传文件存储目录               |
| `ITDB_JWT_SECRET`    | `itdb-change-me` | JWT 签名密钥，生产环境务必设置 |
| `ITDB_HISTORY_LIMIT` | `1000`           | 操作历史保留条数               |
| `ITDB_CORS_ORIGINS`  | `*`              | 允许的跨域来源，多个用逗号分隔 |

3. 运行后端服务：

```bash
# 方式1：前台运行（终端关闭则服务停止）
go run main.go

# 方式2：后台运行（日志输出到 app.log）
nohup go run main.go > app.log 2>&1 &
```

后端服务默认运行在 `http://localhost:8080` ，如需指定端口，请修改环境变量文件内的 `ITDB_SERVER_ADDR` 参数。首次启动会自动创建数据库和默认管理员账户 `admin / admin123` 。

## 6.3 前端配置与启动

1. 进入前端目录下载相关依赖：

```bash
cd frontend
npm install
```

2. 配置 API 地址（可选）：

```bash
# 配置说明：
# - 后端端口 = 8080：无需创建 .env 文件（默认值为 http://127.0.0.1:8080）
# - 后端端口 ≠ 8080：需要创建 .env 文件（指定正确端口，例如后端端口改为 8090）
#   创建 .env 文件，例如：
echo "VITE_API_BASE=http://localhost:8090" > .env
```

3. 启动前端服务：

```bash
# 方式1：前台运行（终端关闭则服务停止）
npm run dev

# 方式2：后台运行（日志输出到 frontend.log）
nohup npm run dev > frontend.log 2>&1 &
```

前端服务默认运行在 `http://localhost:3000`  ，提供了非本机也能访问，将 `localhost` 改为实际 IP 地址即可。

## 6.4 访问系统

- **首页**：`http://localhost:3000`
- **首次访问**：
  - 用户名：`admin`
  - 密码：`admin123`

# 七、生产环境部署

## 7.1 克隆项目

```bash
git clone https://github.com/zyx3721/itdb.git
cd itdb
```

## 7.2 后端构建与配置

1. 进入后端目录下载相关依赖：

```bash
cd backend
go mod tidy
```

2. 配置环境变量：

```bash
# 步骤1：复制模板文件
cp .env.example .env

# 步骤2：编辑 .env，按实际环境修改监听地址、密钥等信息
# 后端监听地址
ITDB_SERVER_ADDR=127.0.0.1:8080

# 数据库与上传目录
ITDB_DB_PATH=./data/itdb.db
ITDB_UPLOAD_DIR=./data/files

# 鉴权与接口行为
ITDB_JWT_SECRET=itdb-change-me
ITDB_HISTORY_LIMIT=1000
ITDB_CORS_ORIGINS=*
```

3. 构建后端可执行文件：

```bash
go build -o itdb-backend main.go
```

4. 运行后端服务： 

```bash
# 方式1：前台运行（终端关闭则服务停止）
./itdb-backend

# 方式2：后台运行（日志输出到 app.log）
nohup ./itdb-backend > app.log 2>&1 &

# 方法3：加入 systemd 管理启动运行
# 服务配置参考如下，请自行修改相应目录路径
cat > /etc/systemd/system/itdb-backend.service <<EOF
[Unit]
Description=ITDB Backend Service
After=network.target

[Service]
Type=simple
WorkingDirectory=/data/itdb/backend
ExecStart=/data/itdb/backend/itdb-backend

StandardOutput=append:/data/itdb/backend/app.log
StandardError=inherit

Restart=on-failure
RestartSec=5

[Install]
WantedBy=multi-user.target
EOF

# 重载服务配置并启动
systemctl daemon-reload
systemctl start itdb-backend

# 设置开机自启
systemctl enable --now itdb-backend
```

后端服务默认运行在 `http://localhost:8080` ，如需指定端口，请修改环境变量文件内的 `ITDB_SERVER_ADDR` 参数。

## 7.3 前端构建与配置

1. 进入前端目录下载相关依赖：

```bash
cd frontend
npm install
```

2. 构建前端项目：

```bash
npm run build
```

构建产物在 `dist` 目录，可部署到任何静态服务器（Nginx、Vercel、Netlify 等）。生产环境前端无需配置 API 地址，统一通过 Nginx `/api/` 反向代理到后端。

## 7.4 配置Nginx反向代理

在服务器上准备前端目录（例如 `/data/itdb/frontend/dist`），**将本地 `dist` 目录中的所有文件和子目录整体上传到该目录**，保持结构不变，例如：

```bash
/data/itdb/frontend/dist/
├── assets/
├── images/
├── index.html
```

Nginx 中的 `root` 应指向 **包含 `index.html` 的目录本身**（如 `/data/itdb/frontend/dist` ，可按实际路径调整），而不是上级目录。

### 7.4.1 HTTP 示例

> 配置 Nginx （按需替换域名/路径/证书），`HTTP 示例` ：

```nginx
server {
    listen 80;
    server_name your-domain.com;   # 修改为你的域名/主机名，例如：itdb.cn
    
    # 前端静态资源目录（dist 构建产物）
    root /data/itdb/frontend/dist;  # 按实际部署路径修改
    index index.html;
    
    # 限制上传文件大小（可选）
    client_max_body_size 50m;
    
    # 前端路由回退到 index.html（适配前端 history 模式）
    location / {
        try_files $uri $uri/ /index.html;
    }
    
    # 后端 API 反向代理
    location /api/ {
        proxy_pass http://127.0.0.1:8080;  # 与后端 API 相同地址
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection "upgrade";
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
        proxy_connect_timeout 60s;
        proxy_send_timeout 300s;
        proxy_read_timeout 300s;
    }
    
    # 健康检查
    location /health {
        proxy_pass http://127.0.0.1:8080;
    }
}
```

### 7.4.2 HTTPS 示例

> HTTPS 示例（含 80→443 跳转，请替换证书路径）：

```nginx
# 80 强制跳转到 443
server {
    listen 80;
    server_name your-domain.com;   # 修改为你的域名/主机名，例如：itdb.cn
    return 301 https://$host$request_uri;
}

server {
    # listen 443 ssl http2;  # Nginx 1.25 以下版本写法
    listen 443 ssl;
    http2 on;
    server_name your-domain.com;   # 修改为你的域名/主机名，例如：itdb.cn

    # 证书路径（替换为实际证书文件）
    ssl_certificate     /usr/local/nginx/ssl/your-domain.com.pem;  # 例如：/usr/local/nginx/ssl/yunwei.cn.pem
    ssl_certificate_key /usr/local/nginx/ssl/your-domain.com.key;  # 例如：/usr/local/nginx/ssl/yunwei.cn.key
    
    # SSL安全优化
    ssl_protocols              TLSv1.2 TLSv1.3;
    ssl_prefer_server_ciphers  on;
    ssl_ciphers                ECDHE-RSA-AES256-GCM-SHA512:DHE-RSA-AES256-GCM-SHA512:ECDHE-RSA-AES256-GCM-SHA384:DHE-RSA-AES256-GCM-SHA384;
    ssl_session_timeout        10m;
    ssl_session_cache          shared:SSL:10m;

    # 前端静态资源目录（dist 构建产物）
    root /data/itdb/frontend/dist;  # 按实际部署路径修改
    index index.html;
    
    # 限制上传文件大小（可选）
    client_max_body_size 50m;
    
    # 前端路由回退到 index.html（适配前端 history 模式）
    location / {
        try_files $uri $uri/ /index.html;
    }
    
    # 后端 API 反向代理
    location /api/ {
        proxy_pass http://127.0.0.1:8080;  # 与后端 API 相同地址
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection "upgrade";
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
        proxy_connect_timeout 60s;
        proxy_send_timeout 300s;
        proxy_read_timeout 300s;
    }
    
    # 健康检查
    location /health {
        proxy_pass http://127.0.0.1:8080;
    }
}
```

重载 Nginx：

```bash
# 检查语法
nginx -t

# 重载配置
## 方法1
nginx -s reload
## 方法2
systemctl reload nginx
```

## 7.5 访问系统

- **首页**：`http://your-domain.com`
- **首次访问**：
  - 用户名：`admin`
  - 密码：`admin123`
- **后端健康检查**：`http://your-domain.com/health` 

# 八、API 文档

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

# 九、数据库说明

使用 SQLite 单文件数据库，默认路径 `backend/data/itdb.db`，共 36 张表。

## 9.1 核心业务表

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

## 9.2 关联表

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

## 9.3 字典表

| 表名 | 说明 |
|------|------|
| `itemtypes` | 硬件类型 |
| `contracttypes` | 合同类型 |
| `contractsubtypes` | 合同子类型 |
| `dpttypes` | 部门 |
| `statustypes` | 资产状态（含颜色） |
| `filetypes` | 文件类型 |

## 9.4 系统表

| 表名 | 说明 |
|------|------|
| `settings` | 系统设置（LDAP 配置等，单行表） |
| `history` | 操作审计日志 |
| `viewhist` | 浏览历史 |
| `labelpapers` | 标签纸预设 |
| `locareas` | 位置区域（平面图热区） |

# 十、常见问题

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

# 十一、安全建议

1. **修改默认密码**：首次部署后立即修改 `admin` 账户的默认密码
2. **设置 JWT 密钥**：生产环境务必在 `.env` 中设置 `ITDB_JWT_SECRET`，避免使用随机生成的临时密钥
3. **启用 HTTPS**：生产环境建议通过 Nginx 配置 SSL 证书，启用 HTTPS 访问
4. **限制访问来源**：通过 Nginx 或防火墙限制系统的访问 IP 范围
5. **定期备份**：虽然系统已有每日自动备份，建议额外配置异地备份策略
6. **文件目录权限**：确保 `backend/data/` 目录权限合理，避免非授权访问数据库和上传文件
7. **环境变量安全**：`.env` 文件包含敏感信息，确保不被提交到版本控制（已在 `.gitignore` 中排除）
8. **CORS 配置**：生产环境按需配置 `ITDB_CORS_ORIGINS`，避免设置为 `*`
