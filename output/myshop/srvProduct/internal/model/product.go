package model

type Product struct {
	Id int64 `gorm:"primaryKey;column:id;comment:商品id"`
	MerId int64 `gorm:"column:mer_id;comment:商户Id(0为总后台管理员创建,不为0的时候是商户后台创建)"`
	Image string `gorm:"column:image;comment:商品图片"`
	RecommendImage string `gorm:"column:recommend_image;comment:推荐图"`
	SliderImage string `gorm:"column:slider_image;comment:轮播图"`
	StoreName string `gorm:"column:store_name;comment:商品名称"`
	StoreInfo string `gorm:"column:store_info;comment:商品简介"`
	Keyword string `gorm:"column:keyword;comment:关键字"`
	BarCode string `gorm:"column:bar_code;comment:商品条码（一维码）"`
	CateId string `gorm:"column:cate_id;comment:分类id"`
	Price float64 `gorm:"column:price;comment:商品价格"`
	VipPrice float64 `gorm:"column:vip_price;comment:会员价格"`
	OtPrice float64 `gorm:"column:ot_price;comment:市场价"`
	Postage float64 `gorm:"column:postage;comment:邮费"`
	UnitName string `gorm:"column:unit_name;comment:单位名"`
	Sort int64 `gorm:"column:sort;comment:排序"`
	Sales int64 `gorm:"column:sales;comment:销量"`
	Stock int64 `gorm:"column:stock;comment:库存"`
	IsShow bool `gorm:"column:is_show;comment:状态（0：未上架，1：上架）"`
	IsHot bool `gorm:"column:is_hot;comment:是否热卖"`
	IsBenefit bool `gorm:"column:is_benefit;comment:是否优惠"`
	IsBest bool `gorm:"column:is_best;comment:是否精品"`
	IsNew bool `gorm:"column:is_new;comment:是否新品"`
	IsVirtual bool `gorm:"column:is_virtual;comment:商品是否是虚拟商品"`
	VirtualType int64 `gorm:"column:virtual_type;comment:虚拟商品类型"`
	AddTime int64 `gorm:"column:add_time;comment:添加时间"`
	IsPostage bool `gorm:"column:is_postage;comment:是否包邮"`
	IsDel bool `gorm:"column:is_del;comment:是否删除"`
	MerUse int64 `gorm:"column:mer_use;comment:商户是否代理 0不可代理1可代理"`
	GiveIntegral int64 `gorm:"column:give_integral;comment:获得积分"`
	Cost float64 `gorm:"column:cost;comment:成本价"`
	IsSeckill bool `gorm:"column:is_seckill;comment:秒杀状态 0 未开启 1已开启"`
	IsBargain bool `gorm:"column:is_bargain;comment:砍价状态 0未开启 1开启"`
	IsGood bool `gorm:"column:is_good;comment:是否优品推荐"`
	IsSub bool `gorm:"column:is_sub;comment:是否单独分佣"`
	IsVip bool `gorm:"column:is_vip;comment:是否开启会员价格"`
	Ficti int64 `gorm:"column:ficti;comment:虚拟销量"`
	Browse int64 `gorm:"column:browse;comment:浏览量"`
	CodePath string `gorm:"column:code_path;comment:商品二维码地址(用户小程序海报)"`
	SoureLink string `gorm:"column:soure_link;comment:淘宝京东1688类型"`
	VideoLink string `gorm:"column:video_link;comment:主图视频链接"`
	TempId int64 `gorm:"column:temp_id;comment:运费模板ID"`
	SpecType int64 `gorm:"column:spec_type;comment:规格 0单 1多"`
	Activity string `gorm:"column:activity;comment:活动显示排序1=秒杀，2=砍价，3=拼团"`
	Spu string `gorm:"column:spu;comment:商品SPU"`
	LabelId string `gorm:"column:label_id;comment:标签ID"`
	CommandWord string `gorm:"column:command_word;comment:复制口令"`
	RecommendList string `gorm:"column:recommend_list;comment:推荐商品id"`
	VipProduct int64 `gorm:"column:vip_product;comment:是否会员专属商品"`
	Presale int64 `gorm:"column:presale;comment:是否预售商品"`
	PresaleStartTime int64 `gorm:"column:presale_start_time;comment:预售开始时间"`
	PresaleEndTime int64 `gorm:"column:presale_end_time;comment:预售结束时间"`
	PresaleDay int64 `gorm:"column:presale_day;comment:预售结束后几天内发货"`
	Logistics string `gorm:"column:logistics;comment:物流方式"`
	Freight int64 `gorm:"column:freight;comment:运费设置"`
	CustomForm string `gorm:"column:custom_form;comment:自定义表单"`
	IsLimit bool `gorm:"column:is_limit;comment:是否开启限购"`
	LimitType int64 `gorm:"column:limit_type;comment:限购类型1单次限购2永久限购"`
	LimitNum int64 `gorm:"column:limit_num;comment:限购数量"`
	MinQty int64 `gorm:"column:min_qty;comment:起购数量"`
	DefaultSku string `gorm:"column:default_sku;comment:默认规格"`
	ParamsList string `gorm:"column:params_list;comment:商品参数"`
	LabelList string `gorm:"column:label_list;comment:商品标签"`
	ProtectionList string `gorm:"column:protection_list;comment:商品保障"`
	IsGift int64 `gorm:"column:is_gift;comment:是否是礼品"`
	GiftPrice float64 `gorm:"column:gift_price;comment:礼品附加费"`
}

func (Product) TableName() string { return "eb_store_product" }
