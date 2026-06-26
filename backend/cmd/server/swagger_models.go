package server

type swaggerErrorResponse struct {
	Error string `json:"error" example:"请求参数错误"`
}

type swaggerIDResponse struct {
	ID int64 `json:"id" example:"1"`
}

type swaggerOKResponse struct {
	OK bool `json:"ok" example:"true"`
}

type swaggerHealthResponse struct {
	Status string `json:"status" example:"ok"`
}

type swaggerMessageResponse struct {
	Message string `json:"message" example:"操作成功"`
}

type swaggerImportDatabaseResponse struct {
	OK      bool   `json:"ok" example:"true"`
	Message string `json:"message" example:"数据库导入成功"`
}

type swaggerGenericObject map[string]interface{}

type swaggerGenericList []map[string]interface{}

type swaggerDashboardSummaryResponse struct {
	Counts map[string]int64 `json:"counts"`
}

type swaggerReportRunResponse struct {
	Meta  map[string]interface{}   `json:"meta"`
	Rows  []map[string]interface{} `json:"rows"`
	Chart []map[string]interface{} `json:"chart"`
}

type swaggerLabelPreviewRequest struct {
	ItemIDs []int64 `json:"itemIds" example:"1,2,3"`
	Preset  string  `json:"preset" example:"A4-3x8"`
}

type swaggerDictionaryPayload map[string]interface{}

type swaggerTagPayload struct {
	Name string `json:"name" example:"生产"`
}
