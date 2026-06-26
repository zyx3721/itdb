package main

import "itdb-backend/cmd/server"

// @title ITDB API
// @version 1.0.0
// @description ITDB IT 资产管理系统后端 API 文档。除登录和健康检查外，接口默认需要 Bearer Token。
// @BasePath /
// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
func main() {
	server.Run()
}
