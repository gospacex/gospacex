package handler

import (
	"context"

	pb "myshop/common/kitexGen/product"
	"myshop/srvProduct/internal/model"
	"myshop/srvProduct/internal/repository"
	"myshop/srvProduct/internal/service"

	"gorm.io/gorm"
)

type ProductHandler struct {
	pb.UnimplementedProductServiceServer
	svc *service.ProductService
}

func NewProductHandler(db *gorm.DB) *ProductHandler {
	return &ProductHandler{
		svc: service.NewProductService(
			repository.NewProductRepo(db),
		),
	}
}

func (h *ProductHandler) Create(ctx context.Context, req *pb.CreateProductReq) (*pb.CreateProductResp, error) {
	m := &model.Product{
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
	if err := h.svc.Create(ctx, m); err != nil {
		return nil, err
	}
	return &pb.CreateProductResp{ Id: int64(m.Id) }, nil
}

func (h *ProductHandler) Get(ctx context.Context, req *pb.GetProductReq) (*pb.GetProductResp, error) {
	m, err := h.svc.Get(ctx, int64(req.Id))
	if err != nil {
		return nil, err
	}
	return &pb.GetProductResp{
		Id: int64(m.Id),
		MerId: int64(m.MerId),
		Image: m.Image,
		RecommendImage: m.RecommendImage,
		SliderImage: m.SliderImage,
		StoreName: m.StoreName,
		StoreInfo: m.StoreInfo,
		Keyword: m.Keyword,
		BarCode: m.BarCode,
		CateId: m.CateId,
		Price: float64(m.Price),
		VipPrice: float64(m.VipPrice),
		OtPrice: float64(m.OtPrice),
		Postage: float64(m.Postage),
		UnitName: m.UnitName,
		Sort: int64(m.Sort),
		Sales: int64(m.Sales),
		Stock: int64(m.Stock),
		IsShow: m.IsShow,
		IsHot: m.IsHot,
		IsBenefit: m.IsBenefit,
		IsBest: m.IsBest,
		IsNew: m.IsNew,
		IsVirtual: m.IsVirtual,
		VirtualType: int64(m.VirtualType),
		AddTime: int64(m.AddTime),
		IsPostage: m.IsPostage,
		IsDel: m.IsDel,
		MerUse: int64(m.MerUse),
		GiveIntegral: int64(m.GiveIntegral),
		Cost: float64(m.Cost),
		IsSeckill: m.IsSeckill,
		IsBargain: m.IsBargain,
		IsGood: m.IsGood,
		IsSub: m.IsSub,
		IsVip: m.IsVip,
		Ficti: int64(m.Ficti),
		Browse: int64(m.Browse),
		CodePath: m.CodePath,
		SoureLink: m.SoureLink,
		VideoLink: m.VideoLink,
		TempId: int64(m.TempId),
		SpecType: int64(m.SpecType),
		Activity: m.Activity,
		Spu: m.Spu,
		LabelId: m.LabelId,
		CommandWord: m.CommandWord,
		RecommendList: m.RecommendList,
		VipProduct: int64(m.VipProduct),
		Presale: int64(m.Presale),
		PresaleStartTime: int64(m.PresaleStartTime),
		PresaleEndTime: int64(m.PresaleEndTime),
		PresaleDay: int64(m.PresaleDay),
		Logistics: m.Logistics,
		Freight: int64(m.Freight),
		CustomForm: m.CustomForm,
		IsLimit: m.IsLimit,
		LimitType: int64(m.LimitType),
		LimitNum: int64(m.LimitNum),
		MinQty: int64(m.MinQty),
		DefaultSku: m.DefaultSku,
		ParamsList: m.ParamsList,
		LabelList: m.LabelList,
		ProtectionList: m.ProtectionList,
		IsGift: int64(m.IsGift),
		GiftPrice: float64(m.GiftPrice),
	}, nil
}

func (h *ProductHandler) List(ctx context.Context, req *pb.ListProductReq) (*pb.ListProductResp, error) {
	list, err := h.svc.List(ctx)
	if err != nil {
		return nil, err
	}
	var items []*pb.ProductItem
	for _, m := range list {
		items = append(items, &pb.ProductItem{
			Id: int64(m.Id),
			MerId: int64(m.MerId),
			Image: m.Image,
			RecommendImage: m.RecommendImage,
			SliderImage: m.SliderImage,
			StoreName: m.StoreName,
			StoreInfo: m.StoreInfo,
			Keyword: m.Keyword,
			BarCode: m.BarCode,
			CateId: m.CateId,
			Price: float64(m.Price),
			VipPrice: float64(m.VipPrice),
			OtPrice: float64(m.OtPrice),
			Postage: float64(m.Postage),
			UnitName: m.UnitName,
			Sort: int64(m.Sort),
			Sales: int64(m.Sales),
			Stock: int64(m.Stock),
			IsShow: m.IsShow,
			IsHot: m.IsHot,
			IsBenefit: m.IsBenefit,
			IsBest: m.IsBest,
			IsNew: m.IsNew,
			IsVirtual: m.IsVirtual,
			VirtualType: int64(m.VirtualType),
			AddTime: int64(m.AddTime),
			IsPostage: m.IsPostage,
			IsDel: m.IsDel,
			MerUse: int64(m.MerUse),
			GiveIntegral: int64(m.GiveIntegral),
			Cost: float64(m.Cost),
			IsSeckill: m.IsSeckill,
			IsBargain: m.IsBargain,
			IsGood: m.IsGood,
			IsSub: m.IsSub,
			IsVip: m.IsVip,
			Ficti: int64(m.Ficti),
			Browse: int64(m.Browse),
			CodePath: m.CodePath,
			SoureLink: m.SoureLink,
			VideoLink: m.VideoLink,
			TempId: int64(m.TempId),
			SpecType: int64(m.SpecType),
			Activity: m.Activity,
			Spu: m.Spu,
			LabelId: m.LabelId,
			CommandWord: m.CommandWord,
			RecommendList: m.RecommendList,
			VipProduct: int64(m.VipProduct),
			Presale: int64(m.Presale),
			PresaleStartTime: int64(m.PresaleStartTime),
			PresaleEndTime: int64(m.PresaleEndTime),
			PresaleDay: int64(m.PresaleDay),
			Logistics: m.Logistics,
			Freight: int64(m.Freight),
			CustomForm: m.CustomForm,
			IsLimit: m.IsLimit,
			LimitType: int64(m.LimitType),
			LimitNum: int64(m.LimitNum),
			MinQty: int64(m.MinQty),
			DefaultSku: m.DefaultSku,
			ParamsList: m.ParamsList,
			LabelList: m.LabelList,
			ProtectionList: m.ProtectionList,
			IsGift: int64(m.IsGift),
			GiftPrice: float64(m.GiftPrice),
		})
	}
	return &pb.ListProductResp{Items: items}, nil
}

func (h *ProductHandler) Update(ctx context.Context, req *pb.UpdateProductReq) (*pb.UpdateProductResp, error) {
	m, err := h.svc.Get(ctx, int64(req.Id))
	if err != nil {
		return nil, err
	}
	if req.MerId != 0 {
		m.MerId = req.MerId
	}
	if req.Image != "" {
		m.Image = req.Image
	}
	if req.RecommendImage != "" {
		m.RecommendImage = req.RecommendImage
	}
	if req.SliderImage != "" {
		m.SliderImage = req.SliderImage
	}
	if req.StoreName != "" {
		m.StoreName = req.StoreName
	}
	if req.StoreInfo != "" {
		m.StoreInfo = req.StoreInfo
	}
	if req.Keyword != "" {
		m.Keyword = req.Keyword
	}
	if req.BarCode != "" {
		m.BarCode = req.BarCode
	}
	if req.CateId != "" {
		m.CateId = req.CateId
	}
	if req.Price != 0.0 {
		m.Price = req.Price
	}
	if req.VipPrice != 0.0 {
		m.VipPrice = req.VipPrice
	}
	if req.OtPrice != 0.0 {
		m.OtPrice = req.OtPrice
	}
	if req.Postage != 0.0 {
		m.Postage = req.Postage
	}
	if req.UnitName != "" {
		m.UnitName = req.UnitName
	}
	if req.Sort != 0 {
		m.Sort = req.Sort
	}
	if req.Sales != 0 {
		m.Sales = req.Sales
	}
	if req.Stock != 0 {
		m.Stock = req.Stock
	}
	if req.IsShow != false {
		m.IsShow = req.IsShow
	}
	if req.IsHot != false {
		m.IsHot = req.IsHot
	}
	if req.IsBenefit != false {
		m.IsBenefit = req.IsBenefit
	}
	if req.IsBest != false {
		m.IsBest = req.IsBest
	}
	if req.IsNew != false {
		m.IsNew = req.IsNew
	}
	if req.IsVirtual != false {
		m.IsVirtual = req.IsVirtual
	}
	if req.VirtualType != 0 {
		m.VirtualType = req.VirtualType
	}
	if req.AddTime != 0 {
		m.AddTime = req.AddTime
	}
	if req.IsPostage != false {
		m.IsPostage = req.IsPostage
	}
	if req.MerUse != 0 {
		m.MerUse = req.MerUse
	}
	if req.GiveIntegral != 0 {
		m.GiveIntegral = req.GiveIntegral
	}
	if req.Cost != 0.0 {
		m.Cost = req.Cost
	}
	if req.IsSeckill != false {
		m.IsSeckill = req.IsSeckill
	}
	if req.IsBargain != false {
		m.IsBargain = req.IsBargain
	}
	if req.IsGood != false {
		m.IsGood = req.IsGood
	}
	if req.IsSub != false {
		m.IsSub = req.IsSub
	}
	if req.IsVip != false {
		m.IsVip = req.IsVip
	}
	if req.Ficti != 0 {
		m.Ficti = req.Ficti
	}
	if req.Browse != 0 {
		m.Browse = req.Browse
	}
	if req.CodePath != "" {
		m.CodePath = req.CodePath
	}
	if req.SoureLink != "" {
		m.SoureLink = req.SoureLink
	}
	if req.VideoLink != "" {
		m.VideoLink = req.VideoLink
	}
	if req.TempId != 0 {
		m.TempId = req.TempId
	}
	if req.SpecType != 0 {
		m.SpecType = req.SpecType
	}
	if req.Activity != "" {
		m.Activity = req.Activity
	}
	if req.Spu != "" {
		m.Spu = req.Spu
	}
	if req.LabelId != "" {
		m.LabelId = req.LabelId
	}
	if req.CommandWord != "" {
		m.CommandWord = req.CommandWord
	}
	if req.RecommendList != "" {
		m.RecommendList = req.RecommendList
	}
	if req.VipProduct != 0 {
		m.VipProduct = req.VipProduct
	}
	if req.Presale != 0 {
		m.Presale = req.Presale
	}
	if req.PresaleStartTime != 0 {
		m.PresaleStartTime = req.PresaleStartTime
	}
	if req.PresaleEndTime != 0 {
		m.PresaleEndTime = req.PresaleEndTime
	}
	if req.PresaleDay != 0 {
		m.PresaleDay = req.PresaleDay
	}
	if req.Logistics != "" {
		m.Logistics = req.Logistics
	}
	if req.Freight != 0 {
		m.Freight = req.Freight
	}
	if req.CustomForm != "" {
		m.CustomForm = req.CustomForm
	}
	if req.IsLimit != false {
		m.IsLimit = req.IsLimit
	}
	if req.LimitType != 0 {
		m.LimitType = req.LimitType
	}
	if req.LimitNum != 0 {
		m.LimitNum = req.LimitNum
	}
	if req.MinQty != 0 {
		m.MinQty = req.MinQty
	}
	if req.DefaultSku != "" {
		m.DefaultSku = req.DefaultSku
	}
	if req.ParamsList != "" {
		m.ParamsList = req.ParamsList
	}
	if req.LabelList != "" {
		m.LabelList = req.LabelList
	}
	if req.ProtectionList != "" {
		m.ProtectionList = req.ProtectionList
	}
	if req.IsGift != 0 {
		m.IsGift = req.IsGift
	}
	if req.GiftPrice != 0.0 {
		m.GiftPrice = req.GiftPrice
	}
	return &pb.UpdateProductResp{Success: true}, h.svc.Update(ctx, m)
}

func (h *ProductHandler) Delete(ctx context.Context, req *pb.DeleteProductReq) (*pb.DeleteProductResp, error) {
	return &pb.DeleteProductResp{Success: true}, h.svc.Delete(ctx, int64(req.Id))
}
