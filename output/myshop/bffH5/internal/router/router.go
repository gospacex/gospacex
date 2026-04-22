package router

import (
	"myshop/bffH5/internal/handler"
	"github.com/gin-gonic/gin"
)

func NewRouter() *gin.Engine {
	r := gin.Default()
	r.GET("/api/v1/products", handler.NewProductHandler().List)
	r.GET("/api/v1/products/:id", handler.NewProductHandler().Get)
	r.POST("/api/v1/products", handler.NewProductHandler().Create)
	r.PUT("/api/v1/products/:id", handler.NewProductHandler().Update)
	r.DELETE("/api/v1/products/:id", handler.NewProductHandler().Delete)
	return r
}
