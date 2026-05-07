package handlers

import (
	"context"
	"fmt"
	"go-alipay/config"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-pay/gopay"
	"github.com/go-pay/gopay/alipay"
	"github.com/go-pay/gopay/pkg/xlog"
)

func AliPayNotify(c *gin.Context) {
	tradeStatus := c.PostForm("trade_status")
	fmt.Println("-------------->", tradeStatus)
	if tradeStatus == "TRADE_CLOSED" {
		c.JSON(http.StatusOK, gin.H{
			"msg": "交易已关闭",
		})
	}
	if tradeStatus == "TRADE_SUCCESS" {
		//验签
		//todo 做自己的业务
		c.JSON(http.StatusOK, gin.H{
			"msg": "成功！",
		})
	}
}

func AliPayReturn(c *gin.Context) {
	fmt.Println("-------->")
	notifyReq, err := alipay.ParseNotifyToBodyMap(c.Request)
	if err != nil {
		xlog.Error(err)
		c.JSON(http.StatusBadRequest, gin.H{
			"msg": "参数错误",
		})
		return
	}
	ok, err := alipay.VerifySign(config.AliPublicKey, notifyReq)
	if err != nil {
		xlog.Error(err)
		c.JSON(http.StatusBadRequest, gin.H{
			"msg": "参数错误",
		})
		return
	}
	msg := ""
	if ok {
		msg = "验签成功"
	} else {
		msg = "验签失败"
	}
	//TODO,做自己的业务
	c.JSON(http.StatusOK, gin.H{
		"msg": msg,
	})
}

func AddOrder(c *gin.Context) {
	//1.获取url进行支付
	client, err := alipay.NewClient(config.AppId, config.PrivateKey, config.IsProduction)
	if err != nil {
		xlog.Error(err)
		return
	}
	client.SetCharset("utf-8").
		SetSignType(alipay.RSA2).
		SetNotifyUrl(config.NotifyURL).
		SetReturnUrl(config.ReturnURL)

	ts := time.Now().UnixMilli()
	fmt.Println("OutTradeNo", ts)
	outTradeNo := fmt.Sprintf("%d", ts)
	bm := make(gopay.BodyMap)
	bm.Set("subject", "2101A专题专高六")
	funcName(bm, outTradeNo)
	bm.Set("total_amount", "10.00")
	bm.Set("product_code", config.ProductCode)

	payUrl, err := client.TradePagePay(context.Background(), bm)
	if err != nil {
		xlog.Error(err)
		return
	}
	//todo 执行入库操作
	//if {if { if {}else{}}else}else{}
	xlog.Debugf("==============>payUrl", payUrl)
	c.JSON(http.StatusOK, gin.H{
		"payUrl": payUrl,
	})
}

func funcName(bm gopay.BodyMap, outTradeNo string) gopay.BodyMap {
	return bm.Set("out_trade_no", outTradeNo)
}

func Test(c *gin.Context) {
	fmt.Println("--------------->")
}
