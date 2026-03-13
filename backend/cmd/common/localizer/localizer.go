package localizer

import (
	"regexp"
	"strings"
	"unicode"
)

var exactMessageCN = map[string]string{
	"invalid json":                                    "请求体 JSON 格式无效",
	"username is required":                            "用户名不能为空",
	"invalid username or password":                    "用户名或密码错误",
	"unauthenticated":                                 "未登录或登录已失效",
	"missing authorization header":                    "缺少 Authorization 请求头",
	"invalid authorization header":                    "Authorization 请求头格式无效",
	"invalid token":                                   "登录令牌无效或已过期",
	"read-only user":                                  "只读用户无写入权限",
	"invalid id":                                      "ID 参数无效",
	"item not found":                                  "资产不存在",
	"software not found":                              "软件不存在",
	"file not found":                                  "文件不存在",
	"location not found":                              "位置不存在",
	"rack not found":                                  "机柜不存在",
	"user not found":                                  "用户不存在",
	"agent not found":                                 "业务对象不存在",
	"invoice not found":                               "发票不存在",
	"contract not found":                              "合同不存在",
	"report not found":                                "报表不存在",
	"invalid item id":                                 "资产 ID 无效",
	"invalid software id":                             "软件 ID 无效",
	"invalid tag id":                                  "标签 ID 无效",
	"invalid contract id":                             "合同 ID 无效",
	"invalid event id":                                "事件 ID 无效",
	"invalid action id":                               "日志 ID 无效",
	"invalid location id":                             "位置 ID 无效",
	"invalid area id":                                 "区域 ID 无效",
	"invalid user node id":                            "用户节点 ID 无效",
	"invalid itemtype node id":                        "资产类型节点 ID 无效",
	"invalid hardware agent node id":                  "硬件厂商节点 ID 无效",
	"invalid software agent node id":                  "软件厂商节点 ID 无效",
	"invalid vendor agent node id":                    "供应商节点 ID 无效",
	"invalid buyer agent node id":                     "采购方节点 ID 无效",
	"invalid contractor node id":                      "承包方节点 ID 无效",
	"name is required":                                "名称不能为空",
	"title is required":                               "标题不能为空",
	"description is required":                         "描述不能为空",
	"action must be add or remove":                    "操作类型必须是 add 或 remove",
	"missing mandatory fields":                        "缺少必填字段",
	"invalid purchasedate":                            "采购日期格式无效",
	"invalid startdate":                               "开始日期格式无效",
	"invalid currentenddate":                          "结束日期格式无效",
	"invalid enddate":                                 "结束日期格式无效",
	"invalid actiondate":                              "日志日期格式无效",
	"invalid date":                                    "日期格式无效",
	"invalid multipart form":                          "上传表单格式无效",
	"invalid floorplan name":                          "平面图文件名无效",
	"delimiter must be one character":                 "分隔符必须是单个字符",
	"file is required":                                "必须上传文件",
	"missing file":                                    "缺少上传文件",
	"missing multipart form":                          "缺少上传表单",
	"title, typeid and date are required":             "标题、文件类型和日期不能为空",
	"title, version and manufacturerid are required":  "名称、版本和厂商不能为空",
	"vendorid, buyerid, number and date are required": "供应方、采购方、编号和日期不能为空",
	"name and floor are required":                     "位置名称和楼层不能为空",
	"usize and depth are required":                    "U 位和深度不能为空",
	"areaname is required":                            "区域名称不能为空",
	"cannot remove user id 1":                         "不能删除系统内置管理员用户（ID=1）",
	"username already exists":                         "用户名已存在",
	"password is required":                            "密码不能为空",
	"tag already exists":                              "标签已存在",
	"action not found":                                "日志不存在",
	"rack has associated items":                       "机柜下仍有关联资产，无法删除",
	"file still has associations":                     "文件仍有关联关系，无法删除",
	"floorplan not found":                             "平面图不存在",
	"unsupported dictionary":                          "不支持的字典类型",
	"cannot delete internal file type":                "不能删除内置文件类型",
	"cannot delete internal status type":              "不能删除内置状态类型",
	"cannot delete internal contract type":            "不能删除内置合同类型",
	"cannot remove default admin user":                "不能删除默认管理员账号",
	"itemtypeid is required":                          "资产类型不能为空",
	"manufacturerid is required":                      "厂商不能为空",
	"principal is required":                           "负责人不能为空",
	"model is required":                               "型号不能为空",
	"invalid user context":                            "用户上下文无效",
	"missing user":                                    "缺少用户上下文",
	"database bootstrap sql is empty":                 "数据库初始化 SQL 为空",
	"floorplan file must be an image":                 "建筑平面图仅支持图片文件扩展名",
}

var (
	reAreaAssoc        = regexp.MustCompile(`^area has (\d+) associations$`)
	reTagAssoc         = regexp.MustCompile(`^tag has associations \(items=(\d+)\s+software=(\d+)\)$`)
	reDeleteItemType   = regexp.MustCompile(`^cannot delete item type in use by (\d+) items$`)
	reDeleteFileType   = regexp.MustCompile(`^cannot delete file type in use by (\d+) files$`)
	reDeleteStatusType = regexp.MustCompile(`^cannot delete status type in use by (\d+) items$`)
	reDeleteDptType    = regexp.MustCompile(`^cannot delete department type in use by (\d+) items$`)
	reDeleteContType   = regexp.MustCompile(`^cannot delete contract type in use by (\d+) contracts$`)
	reDeleteTagAssoc   = regexp.MustCompile(`^cannot delete tag with associations \(items=(\d+)\s+software=(\d+)\)$`)
	reLineFields       = regexp.MustCompile(`^line (\d+) has (\d+) fields, expected (\d+)$`)
	reLineCause        = regexp.MustCompile(`^line (\d+):\s*(.+)$`)
)

func localizeMessage(message string) string {
	msg := strings.TrimSpace(message)
	if msg == "" {
		return "未知错误"
	}

	lower := strings.ToLower(msg)
	if translated, ok := exactMessageCN[lower]; ok {
		return translated
	}

	if strings.HasPrefix(lower, "unsupported date format:") {
		return "不支持的日期格式：" + strings.TrimSpace(msg[len("unsupported date format:"):])
	}
	if strings.HasPrefix(lower, "hash default admin password failed:") {
		return "初始化默认管理员密码失败：" + strings.TrimSpace(msg[len("hash default admin password failed:"):])
	}
	if strings.HasPrefix(lower, "insert default admin failed:") {
		return "写入默认管理员账号失败：" + strings.TrimSpace(msg[len("insert default admin failed:"):])
	}
	if strings.HasPrefix(lower, "execute bootstrap sql #") {
		return "执行数据库初始化语句失败：" + msg
	}
	if strings.Contains(lower, "only one usage of each socket address") || strings.Contains(lower, "address already in use") {
		return "端口已被占用，请更换端口或关闭占用进程"
	}
	if strings.Contains(lower, "no such file or directory") {
		return "文件或目录不存在"
	}
	if strings.Contains(lower, "permission denied") {
		return "权限不足"
	}
	if strings.Contains(lower, "database is locked") {
		return "数据库被锁定，请稍后重试"
	}

	if strings.HasPrefix(lower, "line ") {
		if m := reLineFields.FindStringSubmatch(lower); len(m) == 4 {
			return "导入文件第 " + m[1] + " 行字段数量为 " + m[2] + "，期望 " + m[3]
		}
		if m := reLineCause.FindStringSubmatch(msg); len(m) == 3 {
			return "导入第 " + m[1] + " 行失败：" + localizeMessage(m[2])
		}
		return "导入处理失败：" + msg
	}

	if m := reAreaAssoc.FindStringSubmatch(lower); len(m) == 2 {
		return "区域仍有关联数据（" + m[1] + " 条），无法删除"
	}
	if m := reTagAssoc.FindStringSubmatch(lower); len(m) == 3 {
		return "标签仍有关联关系（资产=" + m[1] + "，软件=" + m[2] + "），无法删除"
	}
	if m := reDeleteItemType.FindStringSubmatch(lower); len(m) == 2 {
		return "该资产类型已被 " + m[1] + " 个资产使用，无法删除"
	}
	if m := reDeleteFileType.FindStringSubmatch(lower); len(m) == 2 {
		return "该文件类型已被 " + m[1] + " 个文件使用，无法删除"
	}
	if m := reDeleteStatusType.FindStringSubmatch(lower); len(m) == 2 {
		return "该状态类型已被 " + m[1] + " 个资产使用，无法删除"
	}
	if m := reDeleteDptType.FindStringSubmatch(lower); len(m) == 2 {
		return "该部门类型已被 " + m[1] + " 个资产使用，无法删除"
	}
	if m := reDeleteContType.FindStringSubmatch(lower); len(m) == 2 {
		return "该合同类型已被 " + m[1] + " 份合同使用，无法删除"
	}
	if m := reDeleteTagAssoc.FindStringSubmatch(lower); len(m) == 3 {
		return "标签仍有关联关系（资产=" + m[1] + "，软件=" + m[2] + "），无法删除"
	}

	if strings.HasPrefix(lower, "invalid ") {
		return "参数无效：" + strings.TrimSpace(msg[len("invalid "):])
	}
	if strings.HasPrefix(lower, "missing ") {
		return "缺少必要参数：" + strings.TrimSpace(msg[len("missing "):])
	}
	if strings.HasSuffix(lower, " is required") {
		return strings.TrimSpace(msg[:len(msg)-len(" is required")]) + " 不能为空"
	}
	if strings.HasPrefix(lower, "cannot ") {
		return "操作不允许：" + msg
	}
	if strings.HasPrefix(lower, "unsupported ") {
		return "不支持的操作：" + strings.TrimSpace(msg[len("unsupported "):])
	}

	replaced := msg
	replaced = strings.ReplaceAll(replaced, "UNIQUE constraint failed", "唯一约束冲突")
	replaced = strings.ReplaceAll(replaced, "FOREIGN KEY constraint failed", "外键约束冲突")
	replaced = strings.ReplaceAll(replaced, "NOT NULL constraint failed", "非空约束冲突")
	replaced = strings.ReplaceAll(replaced, "CHECK constraint failed", "检查约束冲突")
	replaced = strings.ReplaceAll(replaced, "SQLITE_BUSY", "数据库忙")
	replaced = strings.ReplaceAll(replaced, "SQLITE_CONSTRAINT", "数据库约束冲突")
	replaced = strings.ReplaceAll(replaced, "SQL logic error", "SQL 逻辑错误")
	replaced = strings.ReplaceAll(replaced, "syntax error", "语法错误")
	replaced = strings.ReplaceAll(replaced, "no such table", "不存在的数据表")
	replaced = strings.ReplaceAll(replaced, "no such column", "不存在的字段")
	replaced = strings.ReplaceAll(replaced, "constraint failed", "约束冲突")
	replaced = strings.ReplaceAll(replaced, "database schema has changed", "数据库结构已变更")
	replaced = strings.ReplaceAll(replaced, "cannot start a transaction within a transaction", "事务开启失败：当前已存在事务")
	replaced = strings.ReplaceAll(replaced, "foreign key mismatch", "外键不匹配")
	replaced = strings.ReplaceAll(replaced, "datatype mismatch", "数据类型不匹配")
	replaced = strings.ReplaceAll(replaced, "busy timeout", "数据库忙超时")

	if containsChinese(replaced) {
		return replaced
	}

	if containsLatin(replaced) {
		return "系统错误，请稍后重试"
	}

	return replaced
}

func containsChinese(s string) bool {
	for _, r := range s {
		if unicode.Is(unicode.Han, r) {
			return true
		}
	}
	return false
}

func containsLatin(s string) bool {
	for _, r := range s {
		if unicode.IsLetter(r) && r <= unicode.MaxASCII {
			return true
		}
	}
	return false
}

func LocalizeMessage(message string) string {
	return localizeMessage(message)
}
