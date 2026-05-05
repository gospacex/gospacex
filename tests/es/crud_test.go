package es

import (
	"fmt"
	"testing"
	"time"

	"github.com/elastic/go-elasticsearch/v7"
)

func getTestClient(t *testing.T) *elasticsearch.Client {
	cfg := elasticsearch.Config{
		Addresses: []string{"http://localhost:9200"},
	}
	client, err := elasticsearch.NewClient(cfg)
	if err != nil {
		t.Skipf("Skipping test: failed to create Elasticsearch client: %v", err)
	}

	res, err := client.Info()
	if err != nil {
		t.Skipf("Skipping test: Elasticsearch not available: %v", err)
	}
	defer res.Body.Close()

	return client
}

func TestCRUD(t *testing.T) {
	client := getTestClient(t)

	// 使用时间戳生成唯一ID，避免覆盖
	uniqueID := int(time.Now().UnixNano() % 1000000)
	fmt.Println("------->", uniqueID)
	testProduct := &Product{
		ID:        uniqueID,
		MerID:     1005,
		StoreName: "Test Product",
		StoreInfo: "Test product description",
		Price:     99.99,
		Stock:     100,
		IsShow:    true,
		IsNew:     true,
	}
	fmt.Printf("Using test product ID: %d\n", uniqueID)

	//	//DeleteProduct(client, testProduct.ID) // 不在 create 前清理数据

	t.Run("CreateProduct", func(t *testing.T) {
		err := CreateProduct(client, testProduct)
		if err != nil {
			t.Fatalf("CreateProduct failed: %v", err)
		}
	})

	t.Run("GetProduct", func(t *testing.T) {
		product, err := GetProduct(client, testProduct.ID)
		if err != nil {
			t.Fatalf("GetProduct failed: %v", err)
		}
		if product.ID != testProduct.ID {
			t.Errorf("Expected ID %d, got %d", testProduct.ID, product.ID)
		}
		if product.StoreName != testProduct.StoreName {
			t.Errorf("Expected StoreName %s, got %s", testProduct.StoreName, product.StoreName)
		}
		if product.Price != testProduct.Price {
			t.Errorf("Expected Price %.2f, got %.2f", testProduct.Price, product.Price)
		}
		if product.Stock != testProduct.Stock {
			t.Errorf("Expected Stock %d, got %d", testProduct.Stock, product.Stock)
		}
	})

	t.Run("UpdateProduct", func(t *testing.T) {
		updatedProduct := &Product{
			ID:        testProduct.ID,
			StoreName: "Updated Product",
			Price:     199.99,
			Stock:     200,
		}
		err := UpdateProduct(client, updatedProduct)
		if err != nil {
			t.Fatalf("UpdateProduct failed: %v", err)
		}

		product, err := GetProduct(client, updatedProduct.ID)
		if err != nil {
			t.Fatalf("GetProduct after update failed: %v", err)
		}
		if product.StoreName != updatedProduct.StoreName {
			t.Errorf("Expected updated StoreName %s, got %s", updatedProduct.StoreName, product.StoreName)
		}
		if product.Price != updatedProduct.Price {
			t.Errorf("Expected updated Price %.2f, got %.2f", updatedProduct.Price, product.Price)
		}
		if product.Stock != updatedProduct.Stock {
			t.Errorf("Expected updated Stock %d, got %d", updatedProduct.Stock, product.Stock)
		}
	})

	t.Run("DeleteProduct", func(t *testing.T) {
		err := DeleteProduct(client, testProduct.ID)
		if err != nil {
			t.Fatalf("DeleteProduct failed: %v", err)
		}

		_, err = GetProduct(client, testProduct.ID)
		if err == nil {
			t.Error("Expected error when getting deleted product, got nil")
		}
	})
}
