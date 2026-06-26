package server

// swaggerHealthRoot documents GET /health.
// @Summary 健康检查
// @Tags 健康检查
// @Produce json
// @Success 200 {object} swaggerHealthResponse
// @Router /health [get]
func swaggerHealthRoot() {}

// swaggerHealthAPI documents GET /api/health.
// @Summary API 健康检查
// @Tags 健康检查
// @Produce json
// @Success 200 {object} swaggerHealthResponse
// @Router /api/health [get]
func swaggerHealthAPI() {}

// swaggerLogin documents POST /api/auth/login.
// @Summary 用户登录
// @Tags 认证
// @Accept json
// @Produce json
// @Param body body authLoginRequest true "登录信息"
// @Success 200 {object} authLoginResponse
// @Failure 400 {object} swaggerErrorResponse
// @Failure 401 {object} swaggerErrorResponse
// @Router /api/auth/login [post]
func swaggerLogin() {}

// swaggerMe documents GET /api/auth/me.
// @Summary 获取当前用户
// @Tags 认证
// @Produce json
// @Security BearerAuth
// @Success 200 {object} swaggerGenericObject
// @Failure 401 {object} swaggerErrorResponse
// @Router /api/auth/me [get]
func swaggerMe() {}

// swaggerLogout documents POST /api/auth/logout.
// @Summary 登出当前会话
// @Tags 认证
// @Produce json
// @Security BearerAuth
// @Success 200 {object} swaggerOKResponse
// @Router /api/auth/logout [post]
func swaggerLogout() {}

// swaggerBootstrap documents GET /api/bootstrap.
// @Summary 获取前端启动字典数据
// @Tags 启动数据
// @Produce json
// @Security BearerAuth
// @Success 200 {object} swaggerGenericObject
// @Router /api/bootstrap [get]
func swaggerBootstrap() {}

// swaggerDashboardSummary documents GET /api/dashboard/summary.
// @Summary 获取仪表盘统计
// @Tags 仪表盘
// @Produce json
// @Security BearerAuth
// @Success 200 {object} swaggerDashboardSummaryResponse
// @Router /api/dashboard/summary [get]
func swaggerDashboardSummary() {}

// swaggerHistory documents GET /api/history.
// @Summary 获取操作历史
// @Tags 操作历史
// @Produce json
// @Security BearerAuth
// @Param search query string false "搜索关键词"
// @Param limit query int false "返回数量"
// @Param offset query int false "偏移量"
// @Success 200 {array} swaggerGenericObject
// @Router /api/history [get]
func swaggerHistory() {}

// swaggerHistoryExport documents GET /api/history/export.
// @Summary 导出操作历史
// @Tags 操作历史
// @Produce application/vnd.openxmlformats-officedocument.spreadsheetml.sheet
// @Security BearerAuth
// @Success 200 {file} file
// @Router /api/history/export [get]
func swaggerHistoryExport() {}

// swaggerViewHistoryList documents GET /api/view-history.
// @Summary 获取浏览历史
// @Tags 浏览历史
// @Produce json
// @Security BearerAuth
// @Success 200 {array} swaggerGenericObject
// @Router /api/view-history [get]
func swaggerViewHistoryList() {}

// swaggerViewHistoryCreate documents POST /api/view-history.
// @Summary 记录浏览历史
// @Tags 浏览历史
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param body body viewHistoryRequest true "浏览历史"
// @Success 200 {object} swaggerOKResponse
// @Failure 400 {object} swaggerErrorResponse
// @Router /api/view-history [post]
func swaggerViewHistoryCreate() {}

// swaggerBackupDatabase documents GET /api/backups/database.
// @Summary 下载数据库备份
// @Tags 备份
// @Produce application/octet-stream
// @Security BearerAuth
// @Success 200 {file} file
// @Router /api/backups/database [get]
func swaggerBackupDatabase() {}

// swaggerBackupFull documents GET /api/backups/full.
// @Summary 下载全量备份
// @Tags 备份
// @Produce application/gzip
// @Security BearerAuth
// @Success 200 {file} file
// @Router /api/backups/full [get]
func swaggerBackupFull() {}

// swaggerSettingsGet documents GET /api/settings.
// @Summary 获取系统设置
// @Tags 系统设置
// @Produce json
// @Security BearerAuth
// @Success 200 {object} swaggerGenericObject
// @Router /api/settings [get]
func swaggerSettingsGet() {}

// swaggerSettingsUpdate documents PUT /api/settings.
// @Summary 更新系统设置
// @Tags 系统设置
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param body body settingsPayload true "系统设置"
// @Success 200 {object} swaggerGenericObject
// @Failure 403 {object} swaggerErrorResponse
// @Router /api/settings [put]
func swaggerSettingsUpdate() {}

// swaggerSettingsTestLDAP documents POST /api/settings/test-ldap.
// @Summary 测试 LDAP 连接
// @Tags 系统设置
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param body body settingsPayload true "LDAP 设置"
// @Success 200 {object} swaggerMessageResponse
// @Failure 400 {object} swaggerErrorResponse
// @Failure 403 {object} swaggerErrorResponse
// @Router /api/settings/test-ldap [post]
func swaggerSettingsTestLDAP() {}

// swaggerItemsList documents GET /api/items.
// @Summary 获取硬件资产列表
// @Tags 硬件资产
// @Produce json
// @Security BearerAuth
// @Param search query string false "搜索关键词"
// @Param limit query int false "返回数量，-1 表示全部"
// @Param offset query int false "偏移量"
// @Success 200 {array} swaggerGenericObject
// @Router /api/items [get]
func swaggerItemsList() {}

// swaggerItemsGet documents GET /api/items/{id}.
// @Summary 获取硬件资产详情
// @Tags 硬件资产
// @Produce json
// @Security BearerAuth
// @Param id path int true "硬件资产 ID"
// @Success 200 {object} swaggerGenericObject
// @Failure 404 {object} swaggerErrorResponse
// @Router /api/items/{id} [get]
func swaggerItemsGet() {}

// swaggerItemsCreate documents POST /api/items.
// @Summary 创建硬件资产
// @Tags 硬件资产
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param body body itemPayload true "硬件资产"
// @Success 201 {object} swaggerIDResponse
// @Failure 400 {object} swaggerErrorResponse
// @Failure 403 {object} swaggerErrorResponse
// @Router /api/items [post]
func swaggerItemsCreate() {}

// swaggerItemsUpdate documents PUT /api/items/{id}.
// @Summary 更新硬件资产
// @Tags 硬件资产
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "硬件资产 ID"
// @Param body body itemPayload true "硬件资产"
// @Success 200 {object} swaggerIDResponse
// @Failure 400 {object} swaggerErrorResponse
// @Failure 403 {object} swaggerErrorResponse
// @Router /api/items/{id} [put]
func swaggerItemsUpdate() {}

// swaggerItemsDelete documents DELETE /api/items/{id}.
// @Summary 删除硬件资产
// @Tags 硬件资产
// @Produce json
// @Security BearerAuth
// @Param id path int true "硬件资产 ID"
// @Success 200 {object} swaggerOKResponse
// @Failure 403 {object} swaggerErrorResponse
// @Router /api/items/{id} [delete]
func swaggerItemsDelete() {}

// swaggerItemTagsMutate documents POST /api/items/{id}/tags.
// @Summary 关联或移除硬件标签
// @Tags 硬件资产
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "硬件资产 ID"
// @Param body body tagMutationPayload true "标签操作"
// @Success 200 {object} swaggerOKResponse
// @Router /api/items/{id}/tags [post]
func swaggerItemTagsMutate() {}

// swaggerItemActionsList documents GET /api/items/{id}/actions.
// @Summary 获取硬件操作记录
// @Tags 硬件操作记录
// @Produce json
// @Security BearerAuth
// @Param id path int true "硬件资产 ID"
// @Success 200 {array} swaggerGenericObject
// @Router /api/items/{id}/actions [get]
func swaggerItemActionsList() {}

// swaggerItemActionsCreate documents POST /api/items/{id}/actions.
// @Summary 创建硬件操作记录
// @Tags 硬件操作记录
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "硬件资产 ID"
// @Param body body itemActionPayload true "操作记录"
// @Success 201 {object} swaggerIDResponse
// @Router /api/items/{id}/actions [post]
func swaggerItemActionsCreate() {}

// swaggerItemActionsUpdate documents PUT /api/items/{id}/actions/{actionId}.
// @Summary 更新硬件操作记录
// @Tags 硬件操作记录
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "硬件资产 ID"
// @Param actionId path int true "操作记录 ID"
// @Param body body itemActionPayload true "操作记录"
// @Success 200 {object} swaggerIDResponse
// @Router /api/items/{id}/actions/{actionId} [put]
func swaggerItemActionsUpdate() {}

// swaggerItemActionsDelete documents DELETE /api/items/{id}/actions/{actionId}.
// @Summary 删除硬件操作记录
// @Tags 硬件操作记录
// @Produce json
// @Security BearerAuth
// @Param id path int true "硬件资产 ID"
// @Param actionId path int true "操作记录 ID"
// @Success 200 {object} swaggerOKResponse
// @Router /api/items/{id}/actions/{actionId} [delete]
func swaggerItemActionsDelete() {}

// swaggerSoftwareList documents GET /api/software.
// @Summary 获取软件许可列表
// @Tags 软件许可
// @Produce json
// @Security BearerAuth
// @Param search query string false "搜索关键词"
// @Param limit query int false "返回数量，-1 表示全部"
// @Param offset query int false "偏移量"
// @Success 200 {array} swaggerGenericObject
// @Router /api/software [get]
func swaggerSoftwareList() {}

// swaggerSoftwareGet documents GET /api/software/{id}.
// @Summary 获取软件许可详情
// @Tags 软件许可
// @Produce json
// @Security BearerAuth
// @Param id path int true "软件许可 ID"
// @Success 200 {object} swaggerGenericObject
// @Router /api/software/{id} [get]
func swaggerSoftwareGet() {}

// swaggerSoftwareCreate documents POST /api/software.
// @Summary 创建软件许可
// @Tags 软件许可
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param body body softwarePayload true "软件许可"
// @Success 201 {object} swaggerIDResponse
// @Router /api/software [post]
func swaggerSoftwareCreate() {}

// swaggerSoftwareUpdate documents PUT /api/software/{id}.
// @Summary 更新软件许可
// @Tags 软件许可
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "软件许可 ID"
// @Param body body softwarePayload true "软件许可"
// @Success 200 {object} swaggerIDResponse
// @Router /api/software/{id} [put]
func swaggerSoftwareUpdate() {}

// swaggerSoftwareDelete documents DELETE /api/software/{id}.
// @Summary 删除软件许可
// @Tags 软件许可
// @Produce json
// @Security BearerAuth
// @Param id path int true "软件许可 ID"
// @Success 200 {object} swaggerOKResponse
// @Router /api/software/{id} [delete]
func swaggerSoftwareDelete() {}

// swaggerSoftwareTagsMutate documents POST /api/software/{id}/tags.
// @Summary 关联或移除软件标签
// @Tags 软件许可
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "软件许可 ID"
// @Param body body tagMutationPayload true "标签操作"
// @Success 200 {object} swaggerOKResponse
// @Router /api/software/{id}/tags [post]
func swaggerSoftwareTagsMutate() {}

// swaggerInvoicesList documents GET /api/invoices.
// @Summary 获取发票列表
// @Tags 发票
// @Produce json
// @Security BearerAuth
// @Param search query string false "搜索关键词"
// @Param limit query int false "返回数量，-1 表示全部"
// @Param offset query int false "偏移量"
// @Success 200 {array} swaggerGenericObject
// @Router /api/invoices [get]
func swaggerInvoicesList() {}

// swaggerInvoicesGet documents GET /api/invoices/{id}.
// @Summary 获取发票详情
// @Tags 发票
// @Produce json
// @Security BearerAuth
// @Param id path int true "发票 ID"
// @Success 200 {object} swaggerGenericObject
// @Router /api/invoices/{id} [get]
func swaggerInvoicesGet() {}

// swaggerInvoicesCreate documents POST /api/invoices.
// @Summary 创建发票
// @Tags 发票
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param body body invoicePayload true "发票"
// @Success 201 {object} swaggerIDResponse
// @Router /api/invoices [post]
func swaggerInvoicesCreate() {}

// swaggerInvoicesUpdate documents PUT /api/invoices/{id}.
// @Summary 更新发票
// @Tags 发票
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "发票 ID"
// @Param body body invoicePayload true "发票"
// @Success 200 {object} swaggerIDResponse
// @Router /api/invoices/{id} [put]
func swaggerInvoicesUpdate() {}

// swaggerInvoicesDelete documents DELETE /api/invoices/{id}.
// @Summary 删除发票
// @Tags 发票
// @Produce json
// @Security BearerAuth
// @Param id path int true "发票 ID"
// @Success 200 {object} swaggerOKResponse
// @Router /api/invoices/{id} [delete]
func swaggerInvoicesDelete() {}

// swaggerContractsList documents GET /api/contracts.
// @Summary 获取合同列表
// @Tags 合同
// @Produce json
// @Security BearerAuth
// @Param search query string false "搜索关键词"
// @Param limit query int false "返回数量，-1 表示全部"
// @Param offset query int false "偏移量"
// @Success 200 {array} swaggerGenericObject
// @Router /api/contracts [get]
func swaggerContractsList() {}

// swaggerContractsGet documents GET /api/contracts/{id}.
// @Summary 获取合同详情
// @Tags 合同
// @Produce json
// @Security BearerAuth
// @Param id path int true "合同 ID"
// @Success 200 {object} swaggerGenericObject
// @Router /api/contracts/{id} [get]
func swaggerContractsGet() {}

// swaggerContractsCreate documents POST /api/contracts.
// @Summary 创建合同
// @Tags 合同
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param body body contractPayload true "合同"
// @Success 201 {object} swaggerIDResponse
// @Router /api/contracts [post]
func swaggerContractsCreate() {}

// swaggerContractsUpdate documents PUT /api/contracts/{id}.
// @Summary 更新合同
// @Tags 合同
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "合同 ID"
// @Param body body contractPayload true "合同"
// @Success 200 {object} swaggerIDResponse
// @Router /api/contracts/{id} [put]
func swaggerContractsUpdate() {}

// swaggerContractsDelete documents DELETE /api/contracts/{id}.
// @Summary 删除合同
// @Tags 合同
// @Produce json
// @Security BearerAuth
// @Param id path int true "合同 ID"
// @Success 200 {object} swaggerOKResponse
// @Router /api/contracts/{id} [delete]
func swaggerContractsDelete() {}

// swaggerContractEventsList documents GET /api/contracts/{id}/events.
// @Summary 获取合同事件
// @Tags 合同事件
// @Produce json
// @Security BearerAuth
// @Param id path int true "合同 ID"
// @Success 200 {array} swaggerGenericObject
// @Router /api/contracts/{id}/events [get]
func swaggerContractEventsList() {}

// swaggerContractEventsCreate documents POST /api/contracts/{id}/events.
// @Summary 创建合同事件
// @Tags 合同事件
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "合同 ID"
// @Param body body contractEventPayload true "合同事件"
// @Success 201 {object} swaggerIDResponse
// @Router /api/contracts/{id}/events [post]
func swaggerContractEventsCreate() {}

// swaggerContractEventsUpdate documents PUT /api/contracts/{id}/events/{eventId}.
// @Summary 更新合同事件
// @Tags 合同事件
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "合同 ID"
// @Param eventId path int true "事件 ID"
// @Param body body contractEventPayload true "合同事件"
// @Success 200 {object} swaggerIDResponse
// @Router /api/contracts/{id}/events/{eventId} [put]
func swaggerContractEventsUpdate() {}

// swaggerContractEventsDelete documents DELETE /api/contracts/{id}/events/{eventId}.
// @Summary 删除合同事件
// @Tags 合同事件
// @Produce json
// @Security BearerAuth
// @Param id path int true "合同 ID"
// @Param eventId path int true "事件 ID"
// @Success 200 {object} swaggerOKResponse
// @Router /api/contracts/{id}/events/{eventId} [delete]
func swaggerContractEventsDelete() {}

// swaggerFilesList documents GET /api/files.
// @Summary 获取文件列表
// @Tags 文件
// @Produce json
// @Security BearerAuth
// @Param search query string false "搜索关键词"
// @Success 200 {array} swaggerGenericObject
// @Router /api/files [get]
func swaggerFilesList() {}

// swaggerFilesGet documents GET /api/files/{id}.
// @Summary 获取文件详情
// @Tags 文件
// @Produce json
// @Security BearerAuth
// @Param id path int true "文件 ID"
// @Success 200 {object} swaggerGenericObject
// @Router /api/files/{id} [get]
func swaggerFilesGet() {}

// swaggerFilesDownload documents GET /api/files/{id}/download.
// @Summary 下载文件
// @Tags 文件
// @Produce application/octet-stream
// @Security BearerAuth
// @Param id path int true "文件 ID"
// @Success 200 {file} file
// @Router /api/files/{id}/download [get]
func swaggerFilesDownload() {}

// swaggerFilesCreate documents POST /api/files.
// @Summary 上传文件
// @Tags 文件
// @Accept multipart/form-data
// @Produce json
// @Security BearerAuth
// @Param title formData string true "标题"
// @Param typeId formData int true "文件类型 ID"
// @Param date formData string true "签署日期"
// @Param file formData file true "文件"
// @Param itemLinks formData string false "关联硬件 ID，逗号分隔"
// @Param softwareLinks formData string false "关联软件 ID，逗号分隔"
// @Param invoiceLinks formData string false "关联发票 ID，逗号分隔"
// @Param contractLinks formData string false "关联合同 ID，逗号分隔"
// @Success 201 {object} swaggerIDResponse
// @Router /api/files [post]
func swaggerFilesCreate() {}

// swaggerFilesUpdate documents PUT /api/files/{id}.
// @Summary 更新文件
// @Tags 文件
// @Accept multipart/form-data
// @Produce json
// @Security BearerAuth
// @Param id path int true "文件 ID"
// @Param title formData string true "标题"
// @Param typeId formData int true "文件类型 ID"
// @Param date formData string true "签署日期"
// @Param file formData file false "替换文件"
// @Param itemLinks formData string false "关联硬件 ID，逗号分隔"
// @Param softwareLinks formData string false "关联软件 ID，逗号分隔"
// @Param invoiceLinks formData string false "关联发票 ID，逗号分隔"
// @Param contractLinks formData string false "关联合同 ID，逗号分隔"
// @Success 200 {object} swaggerIDResponse
// @Router /api/files/{id} [put]
func swaggerFilesUpdate() {}

// swaggerFilesDelete documents DELETE /api/files/{id}.
// @Summary 删除文件
// @Tags 文件
// @Produce json
// @Security BearerAuth
// @Param id path int true "文件 ID"
// @Success 200 {object} swaggerOKResponse
// @Router /api/files/{id} [delete]
func swaggerFilesDelete() {}

// swaggerAgentsList documents GET /api/agents.
// @Summary 获取厂商/代理商列表
// @Tags 厂商代理
// @Produce json
// @Security BearerAuth
// @Param search query string false "搜索关键词"
// @Param limit query int false "返回数量，-1 表示全部"
// @Param offset query int false "偏移量"
// @Success 200 {array} swaggerGenericObject
// @Router /api/agents [get]
func swaggerAgentsList() {}

// swaggerAgentsGet documents GET /api/agents/{id}.
// @Summary 获取厂商/代理商详情
// @Tags 厂商代理
// @Produce json
// @Security BearerAuth
// @Param id path int true "厂商/代理商 ID"
// @Success 200 {object} swaggerGenericObject
// @Router /api/agents/{id} [get]
func swaggerAgentsGet() {}

// swaggerAgentsCreate documents POST /api/agents.
// @Summary 创建厂商/代理商
// @Tags 厂商代理
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param body body agentPayload true "厂商/代理商"
// @Success 201 {object} swaggerIDResponse
// @Router /api/agents [post]
func swaggerAgentsCreate() {}

// swaggerAgentsUpdate documents PUT /api/agents/{id}.
// @Summary 更新厂商/代理商
// @Tags 厂商代理
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "厂商/代理商 ID"
// @Param body body agentPayload true "厂商/代理商"
// @Success 200 {object} swaggerIDResponse
// @Router /api/agents/{id} [put]
func swaggerAgentsUpdate() {}

// swaggerAgentsDelete documents DELETE /api/agents/{id}.
// @Summary 删除厂商/代理商
// @Tags 厂商代理
// @Produce json
// @Security BearerAuth
// @Param id path int true "厂商/代理商 ID"
// @Success 200 {object} swaggerOKResponse
// @Router /api/agents/{id} [delete]
func swaggerAgentsDelete() {}

// swaggerUsersList documents GET /api/users.
// @Summary 获取用户列表
// @Tags 用户
// @Produce json
// @Security BearerAuth
// @Param search query string false "搜索关键词"
// @Param limit query int false "返回数量"
// @Param offset query int false "偏移量"
// @Success 200 {array} swaggerGenericObject
// @Router /api/users [get]
func swaggerUsersList() {}

// swaggerUsersGet documents GET /api/users/{id}.
// @Summary 获取用户详情
// @Tags 用户
// @Produce json
// @Security BearerAuth
// @Param id path int true "用户 ID"
// @Success 200 {object} swaggerGenericObject
// @Router /api/users/{id} [get]
func swaggerUsersGet() {}

// swaggerUsersCreate documents POST /api/users.
// @Summary 创建用户
// @Tags 用户
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param body body userPayload true "用户"
// @Success 201 {object} swaggerIDResponse
// @Router /api/users [post]
func swaggerUsersCreate() {}

// swaggerUsersUpdate documents PUT /api/users/{id}.
// @Summary 更新用户
// @Tags 用户
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "用户 ID"
// @Param body body userPayload true "用户"
// @Success 200 {object} swaggerIDResponse
// @Router /api/users/{id} [put]
func swaggerUsersUpdate() {}

// swaggerUsersDelete documents DELETE /api/users/{id}.
// @Summary 删除用户
// @Tags 用户
// @Produce json
// @Security BearerAuth
// @Param id path int true "用户 ID"
// @Success 200 {object} swaggerOKResponse
// @Router /api/users/{id} [delete]
func swaggerUsersDelete() {}

// swaggerLocationsList documents GET /api/locations.
// @Summary 获取位置列表
// @Tags 位置
// @Produce json
// @Security BearerAuth
// @Param search query string false "搜索关键词"
// @Success 200 {array} swaggerGenericObject
// @Router /api/locations [get]
func swaggerLocationsList() {}

// swaggerLocationsGet documents GET /api/locations/{id}.
// @Summary 获取位置详情
// @Tags 位置
// @Produce json
// @Security BearerAuth
// @Param id path int true "位置 ID"
// @Success 200 {object} swaggerGenericObject
// @Router /api/locations/{id} [get]
func swaggerLocationsGet() {}

// swaggerLocationsFloorplan documents GET /api/locations/{id}/floorplan.
// @Summary 查看位置平面图
// @Tags 位置
// @Produce application/octet-stream
// @Security BearerAuth
// @Param id path int true "位置 ID"
// @Success 200 {file} file
// @Router /api/locations/{id}/floorplan [get]
func swaggerLocationsFloorplan() {}

// swaggerLocationsCreate documents POST /api/locations.
// @Summary 创建位置
// @Tags 位置
// @Accept multipart/form-data
// @Produce json
// @Security BearerAuth
// @Param name formData string true "位置名称"
// @Param floor formData string true "楼层"
// @Param file formData file false "平面图"
// @Success 201 {object} swaggerIDResponse
// @Router /api/locations [post]
func swaggerLocationsCreate() {}

// swaggerLocationsUpdate documents PUT /api/locations/{id}.
// @Summary 更新位置
// @Tags 位置
// @Accept multipart/form-data
// @Produce json
// @Security BearerAuth
// @Param id path int true "位置 ID"
// @Param name formData string true "位置名称"
// @Param floor formData string true "楼层"
// @Param file formData file false "平面图"
// @Success 200 {object} swaggerIDResponse
// @Router /api/locations/{id} [put]
func swaggerLocationsUpdate() {}

// swaggerLocationsDelete documents DELETE /api/locations/{id}.
// @Summary 删除位置
// @Tags 位置
// @Produce json
// @Security BearerAuth
// @Param id path int true "位置 ID"
// @Success 200 {object} swaggerOKResponse
// @Router /api/locations/{id} [delete]
func swaggerLocationsDelete() {}

// swaggerLocationAreasList documents GET /api/locations/{id}/areas.
// @Summary 获取位置区域
// @Tags 位置区域
// @Produce json
// @Security BearerAuth
// @Param id path int true "位置 ID"
// @Success 200 {array} swaggerGenericObject
// @Router /api/locations/{id}/areas [get]
func swaggerLocationAreasList() {}

// swaggerLocationAreasCreate documents POST /api/locations/{id}/areas.
// @Summary 创建位置区域
// @Tags 位置区域
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "位置 ID"
// @Param body body locAreaPayload true "区域"
// @Success 201 {object} swaggerIDResponse
// @Router /api/locations/{id}/areas [post]
func swaggerLocationAreasCreate() {}

// swaggerLocationAreasUpdate documents PUT /api/locations/{id}/areas/{areaId}.
// @Summary 更新位置区域
// @Tags 位置区域
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "位置 ID"
// @Param areaId path int true "区域 ID"
// @Param body body locAreaPayload true "区域"
// @Success 200 {object} swaggerIDResponse
// @Router /api/locations/{id}/areas/{areaId} [put]
func swaggerLocationAreasUpdate() {}

// swaggerLocationAreasDelete documents DELETE /api/locations/{id}/areas/{areaId}.
// @Summary 删除位置区域
// @Tags 位置区域
// @Produce json
// @Security BearerAuth
// @Param id path int true "位置 ID"
// @Param areaId path int true "区域 ID"
// @Success 200 {object} swaggerOKResponse
// @Router /api/locations/{id}/areas/{areaId} [delete]
func swaggerLocationAreasDelete() {}

// swaggerRacksList documents GET /api/racks.
// @Summary 获取机柜列表
// @Tags 机柜
// @Produce json
// @Security BearerAuth
// @Param search query string false "搜索关键词"
// @Success 200 {array} swaggerGenericObject
// @Router /api/racks [get]
func swaggerRacksList() {}

// swaggerRacksGet documents GET /api/racks/{id}.
// @Summary 获取机柜详情
// @Tags 机柜
// @Produce json
// @Security BearerAuth
// @Param id path int true "机柜 ID"
// @Success 200 {object} swaggerGenericObject
// @Router /api/racks/{id} [get]
func swaggerRacksGet() {}

// swaggerRacksCreate documents POST /api/racks.
// @Summary 创建机柜
// @Tags 机柜
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param body body rackPayload true "机柜"
// @Success 201 {object} swaggerIDResponse
// @Router /api/racks [post]
func swaggerRacksCreate() {}

// swaggerRacksUpdate documents PUT /api/racks/{id}.
// @Summary 更新机柜
// @Tags 机柜
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "机柜 ID"
// @Param body body rackPayload true "机柜"
// @Success 200 {object} swaggerIDResponse
// @Router /api/racks/{id} [put]
func swaggerRacksUpdate() {}

// swaggerRacksDelete documents DELETE /api/racks/{id}.
// @Summary 删除机柜
// @Tags 机柜
// @Produce json
// @Security BearerAuth
// @Param id path int true "机柜 ID"
// @Success 200 {object} swaggerOKResponse
// @Router /api/racks/{id} [delete]
func swaggerRacksDelete() {}

// swaggerDictionariesList documents GET /api/dictionaries.
// @Summary 获取所有字典数据
// @Tags 字典
// @Produce json
// @Security BearerAuth
// @Success 200 {object} swaggerGenericObject
// @Router /api/dictionaries [get]
func swaggerDictionariesList() {}

// swaggerDictionariesCreate documents POST /api/dictionaries/{name}.
// @Summary 创建字典行
// @Tags 字典
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param name path string true "字典名称"
// @Param body body swaggerDictionaryPayload true "字典行"
// @Success 201 {object} swaggerIDResponse
// @Router /api/dictionaries/{name} [post]
func swaggerDictionariesCreate() {}

// swaggerDictionariesUpdate documents PUT /api/dictionaries/{name}/{id}.
// @Summary 更新字典行
// @Tags 字典
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param name path string true "字典名称"
// @Param id path int true "字典行 ID"
// @Param body body swaggerDictionaryPayload true "字典行"
// @Success 200 {object} swaggerIDResponse
// @Router /api/dictionaries/{name}/{id} [put]
func swaggerDictionariesUpdate() {}

// swaggerDictionariesDelete documents DELETE /api/dictionaries/{name}/{id}.
// @Summary 删除字典行
// @Tags 字典
// @Produce json
// @Security BearerAuth
// @Param name path string true "字典名称"
// @Param id path int true "字典行 ID"
// @Success 200 {object} swaggerOKResponse
// @Router /api/dictionaries/{name}/{id} [delete]
func swaggerDictionariesDelete() {}

// swaggerTagsList documents GET /api/tags.
// @Summary 获取标签列表
// @Tags 标签
// @Produce json
// @Security BearerAuth
// @Param search query string false "搜索关键词"
// @Success 200 {array} swaggerGenericObject
// @Router /api/tags [get]
func swaggerTagsList() {}

// swaggerTagsSuggest documents GET /api/tags/suggest.
// @Summary 标签建议
// @Tags 标签
// @Produce json
// @Security BearerAuth
// @Param term query string false "输入关键词"
// @Success 200 {array} string
// @Router /api/tags/suggest [get]
func swaggerTagsSuggest() {}

// swaggerTagsCreate documents POST /api/tags.
// @Summary 创建标签
// @Tags 标签
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param body body swaggerTagPayload true "标签"
// @Success 201 {object} swaggerIDResponse
// @Router /api/tags [post]
func swaggerTagsCreate() {}

// swaggerTagsUpdate documents PUT /api/tags/{id}.
// @Summary 更新标签
// @Tags 标签
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "标签 ID"
// @Param body body swaggerTagPayload true "标签"
// @Success 200 {object} swaggerIDResponse
// @Router /api/tags/{id} [put]
func swaggerTagsUpdate() {}

// swaggerTagsDelete documents DELETE /api/tags/{id}.
// @Summary 删除标签
// @Tags 标签
// @Produce json
// @Security BearerAuth
// @Param id path int true "标签 ID"
// @Success 200 {object} swaggerOKResponse
// @Router /api/tags/{id} [delete]
func swaggerTagsDelete() {}

// swaggerTagsItems documents GET /api/tags/{id}/items.
// @Summary 获取标签关联硬件
// @Tags 标签
// @Produce json
// @Security BearerAuth
// @Param id path int true "标签 ID"
// @Success 200 {array} swaggerGenericObject
// @Router /api/tags/{id}/items [get]
func swaggerTagsItems() {}

// swaggerTagsSoftware documents GET /api/tags/{id}/software.
// @Summary 获取标签关联软件
// @Tags 标签
// @Produce json
// @Security BearerAuth
// @Param id path int true "标签 ID"
// @Success 200 {array} swaggerGenericObject
// @Router /api/tags/{id}/software [get]
func swaggerTagsSoftware() {}

// swaggerReportsList documents GET /api/reports.
// @Summary 获取报表列表
// @Tags 报表
// @Produce json
// @Security BearerAuth
// @Success 200 {array} reportDefinition
// @Router /api/reports [get]
func swaggerReportsList() {}

// swaggerReportsRun documents GET /api/reports/{name}.
// @Summary 执行报表
// @Tags 报表
// @Produce json
// @Security BearerAuth
// @Param name path string true "报表名称"
// @Param limit query int false "返回数量"
// @Success 200 {object} swaggerReportRunResponse
// @Router /api/reports/{name} [get]
func swaggerReportsRun() {}

// swaggerImportDatabase documents POST /api/import/database.
// @Summary 导入 SQLite 数据库
// @Tags 数据库导入
// @Accept multipart/form-data
// @Produce json
// @Security BearerAuth
// @Param file formData file true "SQLite .db 文件"
// @Success 200 {object} swaggerImportDatabaseResponse
// @Failure 400 {object} swaggerErrorResponse
// @Router /api/import/database [post]
func swaggerImportDatabase() {}

// swaggerBrowseTree documents GET /api/browse/tree.
// @Summary 获取浏览树
// @Tags 浏览树
// @Produce json
// @Security BearerAuth
// @Param id query string false "节点 ID"
// @Success 200 {array} browseNode
// @Router /api/browse/tree [get]
func swaggerBrowseTree() {}

// swaggerLabelsItems documents GET /api/labels/items.
// @Summary 获取标签打印资产
// @Tags 标签打印
// @Produce json
// @Security BearerAuth
// @Param search query string false "搜索关键词"
// @Param orderBy query string false "排序字段"
// @Param limit query int false "返回数量"
// @Param offset query int false "偏移量"
// @Success 200 {array} swaggerGenericObject
// @Router /api/labels/items [get]
func swaggerLabelsItems() {}

// swaggerLabelsPresets documents GET /api/labels/presets.
// @Summary 获取标签纸预设
// @Tags 标签打印
// @Produce json
// @Security BearerAuth
// @Success 200 {array} swaggerGenericObject
// @Router /api/labels/presets [get]
func swaggerLabelsPresets() {}

// swaggerLabelsPreview documents POST /api/labels/preview.
// @Summary 预览标签打印
// @Tags 标签打印
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param body body swaggerLabelPreviewRequest true "标签预览参数"
// @Success 200 {object} swaggerGenericObject
// @Router /api/labels/preview [post]
func swaggerLabelsPreview() {}

// swaggerLabelsPresetsCreate documents POST /api/labels/presets.
// @Summary 创建标签纸预设
// @Tags 标签打印
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param body body swaggerDictionaryPayload true "标签纸预设"
// @Success 201 {object} swaggerIDResponse
// @Router /api/labels/presets [post]
func swaggerLabelsPresetsCreate() {}

// swaggerLabelsPresetsDelete documents DELETE /api/labels/presets/{id}.
// @Summary 删除标签纸预设
// @Tags 标签打印
// @Produce json
// @Security BearerAuth
// @Param id path int true "标签纸预设 ID"
// @Success 200 {object} swaggerOKResponse
// @Router /api/labels/presets/{id} [delete]
func swaggerLabelsPresetsDelete() {}
