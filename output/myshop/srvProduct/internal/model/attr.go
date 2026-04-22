package model

type Attr struct {
	Id int64 `gorm:"primaryKey;column:id;comment:自增ID"`
	ProductId int64 `gorm:"column:product_id;comment:商品ID"`
	AttrName string `gorm:"column:attr_name;comment:属性名"`
	AttrValues string `gorm:"column:attr_values;comment:属性值"`
	Type int64 `gorm:"column:type;comment:活动类型 0=商品，1=秒杀，2=砍价，3=拼团"`
}

func (Attr) TableName() string { return "eb_store_product_attr" }
