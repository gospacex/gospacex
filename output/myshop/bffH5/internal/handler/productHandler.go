package handler

import (
	"net/http"
	"strconv"
	"myshop/bffH5/internal/rpcClient"
	pb "myshop/common/kitexGen/product"

	"github.com/gin-gonic/gin"
)

type ProductHandler struct {
	client *rpcclient.ProductClient
}

func NewProductHandler() *ProductHandler {
	cli, err := rpcclient.NewProductClient("127.0.0.1:8001")
	if err != nil {
		panic("failed to create product client: " + err.Error())
	}
	return &ProductHandler{client: cli}
}

func (h *ProductHandler) Create(c *gin.Context) {
	var req struct {
		MerId int64 `json:"mer_id"`
		Image string `json:"image"`
		RecommendImage string `json:"recommend_image"`
		SliderImage string `json:"slider_image"`
		StoreName string `json:"store_name"`
		StoreInfo string `json:"store_info"`
		Keyword string `json:"keyword"`
		BarCode string `json:"bar_code"`
		CateId string `json:"cate_id"`
		Price float64 `json:"price"`
		VipPrice float64 `json:"vip_price"`
		OtPrice float64 `json:"ot_price"`
		Postage float64 `json:"postage"`
		UnitName string `json:"unit_name"`
		Sort int64 `json:"sort"`
		Sales int64 `json:"sales"`
		Stock int64 `json:"stock"`
		IsShow bool `json:"is_show"`
		IsHot bool `json:"is_hot"`
		IsBenefit bool `json:"is_benefit"`
		IsBest bool `json:"is_best"`
		IsNew bool `json:"is_new"`
		IsVirtual bool `json:"is_virtual"`
		VirtualType int64 `json:"virtual_type"`
		AddTime int64 `json:"add_time"`
		IsPostage bool `json:"is_postage"`
		MerUse int64 `json:"mer_use"`
		GiveIntegral int64 `json:"give_integral"`
		Cost float64 `json:"cost"`
		IsSeckill bool `json:"is_seckill"`
		IsBargain bool `json:"is_bargain"`
		IsGood bool `json:"is_good"`
		IsSub bool `json:"is_sub"`
		IsVip bool `json:"is_vip"`
		Ficti int64 `json:"ficti"`
		Browse int64 `json:"browse"`
		CodePath string `json:"code_path"`
		SoureLink string `json:"soure_link"`
		VideoLink string `json:"video_link"`
		TempId int64 `json:"temp_id"`
		SpecType int64 `json:"spec_type"`
		Activity string `json:"activity"`
		Spu string `json:"spu"`
		LabelId string `json:"label_id"`
		CommandWord string `json:"command_word"`
		RecommendList string `json:"recommend_list"`
		VipProduct int64 `json:"vip_product"`
		Presale int64 `json:"presale"`
		PresaleStartTime int64 `json:"presale_start_time"`
		PresaleEndTime int64 `json:"presale_end_time"`
		PresaleDay int64 `json:"presale_day"`
		Logistics string `json:"logistics"`
		Freight int64 `json:"freight"`
		CustomForm string `json:"custom_form"`
		IsLimit bool `json:"is_limit"`
		LimitType int64 `json:"limit_type"`
		LimitNum int64 `json:"limit_num"`
		MinQty int64 `json:"min_qty"`
		DefaultSku string `json:"default_sku"`
		ParamsList string `json:"params_list"`
		LabelList string `json:"label_list"`
		ProtectionList string `json:"protection_list"`
		IsGift int64 `json:"is_gift"`
		GiftPrice float64 `json:"gift_price"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	rpcReq := &pb.CreateProductReq{
		MerId: req.MerId,
		Image: req.Image,
		RecommendImage: req.RecommendImage,
		SliderImage: req.SliderImage,
		StoreName: req.StoreName,
		StoreInfo: req.StoreInfo,
		Keyword: req.Keyword,
		BarCode: req.BarCode,
		CateId: req.CateId,
		Price: req.Price,
		VipPrice: req.VipPrice,
		OtPrice: req.OtPrice,
		Postage: req.Postage,
		UnitName: req.UnitName,
		Sort: req.Sort,
		Sales: req.Sales,
		Stock: req.Stock,
		IsShow: req.IsShow,
		IsHot: req.IsHot,
		IsBenefit: req.IsBenefit,
		IsBest: req.IsBest,
		IsNew: req.IsNew,
		IsVirtual: req.IsVirtual,
		VirtualType: req.VirtualType,
		AddTime: req.AddTime,
		IsPostage: req.IsPostage,
		MerUse: req.MerUse,
		GiveIntegral: req.GiveIntegral,
		Cost: req.Cost,
		IsSeckill: req.IsSeckill,
		IsBargain: req.IsBargain,
		IsGood: req.IsGood,
		IsSub: req.IsSub,
		IsVip: req.IsVip,
		Ficti: req.Ficti,
		Browse: req.Browse,
		CodePath: req.CodePath,
		SoureLink: req.SoureLink,
		VideoLink: req.VideoLink,
		TempId: req.TempId,
		SpecType: req.SpecType,
		Activity: req.Activity,
		Spu: req.Spu,
		LabelId: req.LabelId,
		CommandWord: req.CommandWord,
		RecommendList: req.RecommendList,
		VipProduct: req.VipProduct,
		Presale: req.Presale,
		PresaleStartTime: req.PresaleStartTime,
		PresaleEndTime: req.PresaleEndTime,
		PresaleDay: req.PresaleDay,
		Logistics: req.Logistics,
		Freight: req.Freight,
		CustomForm: req.CustomForm,
		IsLimit: req.IsLimit,
		LimitType: req.LimitType,
		LimitNum: req.LimitNum,
		MinQty: req.MinQty,
		DefaultSku: req.DefaultSku,
		ParamsList: req.ParamsList,
		LabelList: req.LabelList,
		ProtectionList: req.ProtectionList,
		IsGift: req.IsGift,
		GiftPrice: req.GiftPrice,
	}

	resp, err := h.client.Create(c.Request.Context(), rpcReq)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"id": resp.Id})
}

func (h *ProductHandler) Get(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	resp, err := h.client.Get(c.Request.Context(), int64(id))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"id": resp.Id,
		"mer_id": resp.MerId,
		"image": resp.Image,
		"recommend_image": resp.RecommendImage,
		"slider_image": resp.SliderImage,
		"store_name": resp.StoreName,
		"store_info": resp.StoreInfo,
		"keyword": resp.Keyword,
		"bar_code": resp.BarCode,
		"cate_id": resp.CateId,
		"price": resp.Price,
		"vip_price": resp.VipPrice,
		"ot_price": resp.OtPrice,
		"postage": resp.Postage,
		"unit_name": resp.UnitName,
		"sort": resp.Sort,
		"sales": resp.Sales,
		"stock": resp.Stock,
		"is_show": resp.IsShow,
		"is_hot": resp.IsHot,
		"is_benefit": resp.IsBenefit,
		"is_best": resp.IsBest,
		"is_new": resp.IsNew,
		"is_virtual": resp.IsVirtual,
		"virtual_type": resp.VirtualType,
		"add_time": resp.AddTime,
		"is_postage": resp.IsPostage,
		"is_del": resp.IsDel,
		"mer_use": resp.MerUse,
		"give_integral": resp.GiveIntegral,
		"cost": resp.Cost,
		"is_seckill": resp.IsSeckill,
		"is_bargain": resp.IsBargain,
		"is_good": resp.IsGood,
		"is_sub": resp.IsSub,
		"is_vip": resp.IsVip,
		"ficti": resp.Ficti,
		"browse": resp.Browse,
		"code_path": resp.CodePath,
		"soure_link": resp.SoureLink,
		"video_link": resp.VideoLink,
		"temp_id": resp.TempId,
		"spec_type": resp.SpecType,
		"activity": resp.Activity,
		"spu": resp.Spu,
		"label_id": resp.LabelId,
		"command_word": resp.CommandWord,
		"recommend_list": resp.RecommendList,
		"vip_product": resp.VipProduct,
		"presale": resp.Presale,
		"presale_start_time": resp.PresaleStartTime,
		"presale_end_time": resp.PresaleEndTime,
		"presale_day": resp.PresaleDay,
		"logistics": resp.Logistics,
		"freight": resp.Freight,
		"custom_form": resp.CustomForm,
		"is_limit": resp.IsLimit,
		"limit_type": resp.LimitType,
		"limit_num": resp.LimitNum,
		"min_qty": resp.MinQty,
		"default_sku": resp.DefaultSku,
		"params_list": resp.ParamsList,
		"label_list": resp.LabelList,
		"protection_list": resp.ProtectionList,
		"is_gift": resp.IsGift,
		"gift_price": resp.GiftPrice,
	})
}

func (h *ProductHandler) List(c *gin.Context) {
	resp, err := h.client.List(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	var items []map[string]interface{}
	for _, item := range resp.Items {
		items = append(items, map[string]interface{}{
			"id": item.Id,
			"mer_id": item.MerId,
			"image": item.Image,
			"recommend_image": item.RecommendImage,
			"slider_image": item.SliderImage,
			"store_name": item.StoreName,
			"store_info": item.StoreInfo,
			"keyword": item.Keyword,
			"bar_code": item.BarCode,
			"cate_id": item.CateId,
			"price": item.Price,
			"vip_price": item.VipPrice,
			"ot_price": item.OtPrice,
			"postage": item.Postage,
			"unit_name": item.UnitName,
			"sort": item.Sort,
			"sales": item.Sales,
			"stock": item.Stock,
			"is_show": item.IsShow,
			"is_hot": item.IsHot,
			"is_benefit": item.IsBenefit,
			"is_best": item.IsBest,
			"is_new": item.IsNew,
			"is_virtual": item.IsVirtual,
			"virtual_type": item.VirtualType,
			"add_time": item.AddTime,
			"is_postage": item.IsPostage,
			"is_del": item.IsDel,
			"mer_use": item.MerUse,
			"give_integral": item.GiveIntegral,
			"cost": item.Cost,
			"is_seckill": item.IsSeckill,
			"is_bargain": item.IsBargain,
			"is_good": item.IsGood,
			"is_sub": item.IsSub,
			"is_vip": item.IsVip,
			"ficti": item.Ficti,
			"browse": item.Browse,
			"code_path": item.CodePath,
			"soure_link": item.SoureLink,
			"video_link": item.VideoLink,
			"temp_id": item.TempId,
			"spec_type": item.SpecType,
			"activity": item.Activity,
			"spu": item.Spu,
			"label_id": item.LabelId,
			"command_word": item.CommandWord,
			"recommend_list": item.RecommendList,
			"vip_product": item.VipProduct,
			"presale": item.Presale,
			"presale_start_time": item.PresaleStartTime,
			"presale_end_time": item.PresaleEndTime,
			"presale_day": item.PresaleDay,
			"logistics": item.Logistics,
			"freight": item.Freight,
			"custom_form": item.CustomForm,
			"is_limit": item.IsLimit,
			"limit_type": item.LimitType,
			"limit_num": item.LimitNum,
			"min_qty": item.MinQty,
			"default_sku": item.DefaultSku,
			"params_list": item.ParamsList,
			"label_list": item.LabelList,
			"protection_list": item.ProtectionList,
			"is_gift": item.IsGift,
			"gift_price": item.GiftPrice,
		})
	}
	c.JSON(http.StatusOK, gin.H{"items": items})
}

func (h *ProductHandler) Update(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	var req struct {
		MerId int64 `json:"mer_id"`
		Image string `json:"image"`
		RecommendImage string `json:"recommend_image"`
		SliderImage string `json:"slider_image"`
		StoreName string `json:"store_name"`
		StoreInfo string `json:"store_info"`
		Keyword string `json:"keyword"`
		BarCode string `json:"bar_code"`
		CateId string `json:"cate_id"`
		Price float64 `json:"price"`
		VipPrice float64 `json:"vip_price"`
		OtPrice float64 `json:"ot_price"`
		Postage float64 `json:"postage"`
		UnitName string `json:"unit_name"`
		Sort int64 `json:"sort"`
		Sales int64 `json:"sales"`
		Stock int64 `json:"stock"`
		IsShow bool `json:"is_show"`
		IsHot bool `json:"is_hot"`
		IsBenefit bool `json:"is_benefit"`
		IsBest bool `json:"is_best"`
		IsNew bool `json:"is_new"`
		IsVirtual bool `json:"is_virtual"`
		VirtualType int64 `json:"virtual_type"`
		AddTime int64 `json:"add_time"`
		IsPostage bool `json:"is_postage"`
		MerUse int64 `json:"mer_use"`
		GiveIntegral int64 `json:"give_integral"`
		Cost float64 `json:"cost"`
		IsSeckill bool `json:"is_seckill"`
		IsBargain bool `json:"is_bargain"`
		IsGood bool `json:"is_good"`
		IsSub bool `json:"is_sub"`
		IsVip bool `json:"is_vip"`
		Ficti int64 `json:"ficti"`
		Browse int64 `json:"browse"`
		CodePath string `json:"code_path"`
		SoureLink string `json:"soure_link"`
		VideoLink string `json:"video_link"`
		TempId int64 `json:"temp_id"`
		SpecType int64 `json:"spec_type"`
		Activity string `json:"activity"`
		Spu string `json:"spu"`
		LabelId string `json:"label_id"`
		CommandWord string `json:"command_word"`
		RecommendList string `json:"recommend_list"`
		VipProduct int64 `json:"vip_product"`
		Presale int64 `json:"presale"`
		PresaleStartTime int64 `json:"presale_start_time"`
		PresaleEndTime int64 `json:"presale_end_time"`
		PresaleDay int64 `json:"presale_day"`
		Logistics string `json:"logistics"`
		Freight int64 `json:"freight"`
		CustomForm string `json:"custom_form"`
		IsLimit bool `json:"is_limit"`
		LimitType int64 `json:"limit_type"`
		LimitNum int64 `json:"limit_num"`
		MinQty int64 `json:"min_qty"`
		DefaultSku string `json:"default_sku"`
		ParamsList string `json:"params_list"`
		LabelList string `json:"label_list"`
		ProtectionList string `json:"protection_list"`
		IsGift int64 `json:"is_gift"`
		GiftPrice float64 `json:"gift_price"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	rpcReq := &pb.UpdateProductReq{
		Id: int64(id),
		MerId: req.MerId,
		Image: req.Image,
		RecommendImage: req.RecommendImage,
		SliderImage: req.SliderImage,
		StoreName: req.StoreName,
		StoreInfo: req.StoreInfo,
		Keyword: req.Keyword,
		BarCode: req.BarCode,
		CateId: req.CateId,
		Price: req.Price,
		VipPrice: req.VipPrice,
		OtPrice: req.OtPrice,
		Postage: req.Postage,
		UnitName: req.UnitName,
		Sort: req.Sort,
		Sales: req.Sales,
		Stock: req.Stock,
		IsShow: req.IsShow,
		IsHot: req.IsHot,
		IsBenefit: req.IsBenefit,
		IsBest: req.IsBest,
		IsNew: req.IsNew,
		IsVirtual: req.IsVirtual,
		VirtualType: req.VirtualType,
		AddTime: req.AddTime,
		IsPostage: req.IsPostage,
		MerUse: req.MerUse,
		GiveIntegral: req.GiveIntegral,
		Cost: req.Cost,
		IsSeckill: req.IsSeckill,
		IsBargain: req.IsBargain,
		IsGood: req.IsGood,
		IsSub: req.IsSub,
		IsVip: req.IsVip,
		Ficti: req.Ficti,
		Browse: req.Browse,
		CodePath: req.CodePath,
		SoureLink: req.SoureLink,
		VideoLink: req.VideoLink,
		TempId: req.TempId,
		SpecType: req.SpecType,
		Activity: req.Activity,
		Spu: req.Spu,
		LabelId: req.LabelId,
		CommandWord: req.CommandWord,
		RecommendList: req.RecommendList,
		VipProduct: req.VipProduct,
		Presale: req.Presale,
		PresaleStartTime: req.PresaleStartTime,
		PresaleEndTime: req.PresaleEndTime,
		PresaleDay: req.PresaleDay,
		Logistics: req.Logistics,
		Freight: req.Freight,
		CustomForm: req.CustomForm,
		IsLimit: req.IsLimit,
		LimitType: req.LimitType,
		LimitNum: req.LimitNum,
		MinQty: req.MinQty,
		DefaultSku: req.DefaultSku,
		ParamsList: req.ParamsList,
		LabelList: req.LabelList,
		ProtectionList: req.ProtectionList,
		IsGift: req.IsGift,
		GiftPrice: req.GiftPrice,
	}

	resp, err := h.client.Update(c.Request.Context(), rpcReq)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": resp.Success})
}

func (h *ProductHandler) Delete(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	resp, err := h.client.Delete(c.Request.Context(), int64(id))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": resp.Success})
}
