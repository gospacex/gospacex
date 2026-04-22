package model

type Description struct {
	ProductId int64 `gorm:"column:product_id;comment:商品ID"`
	Description string `gorm:"column:description;comment:商品详情"`
	Type int64 `gorm:"column:type;comment:商品类型"`
}

func (Description) TableName() string { return "eb_store_product_description" }
