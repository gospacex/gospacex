package main

import (
	"go-alipay/handlers"

	"github.com/gin-gonic/gin"
)

func main() {

	//2.支付成功后，支付宝回调我们
	engine := gin.Default()
	//支付宝回调
	engine.POST("/notify", handlers.AliPayNotify)
	//支付成功跳转
	engine.GET("/return", handlers.AliPayReturn)
	//支付订单
	engine.GET("/order", handlers.AddOrder)

	engine.GET("/test", handlers.Test)

	engine.Run(":8888")
}
