package rpcclient

import (
	"context"

	pb "myshop/common/kitexGen/product"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type ProductClient struct {
	conn   *grpc.ClientConn
	client pb.ProductServiceClient
}

func NewProductClient(addr string) (*ProductClient, error) {
	conn, err := grpc.Dial(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, err
	}
	return &ProductClient{conn: conn, client: pb.NewProductServiceClient(conn)}, nil
}

func (c *ProductClient) Close() error {
	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}

func (c *ProductClient) Create(ctx context.Context, req *pb.CreateProductReq) (*pb.CreateProductResp, error) {
	return c.client.Create(ctx, req)
}

func (c *ProductClient) Get(ctx context.Context, id int64) (*pb.GetProductResp, error) {
	return c.client.Get(ctx, &pb.GetProductReq{ Id: id })
}

func (c *ProductClient) List(ctx context.Context) (*pb.ListProductResp, error) {
	return c.client.List(ctx, &pb.ListProductReq{})
}

func (c *ProductClient) Update(ctx context.Context, req *pb.UpdateProductReq) (*pb.UpdateProductResp, error) {
	return c.client.Update(ctx, req)
}

func (c *ProductClient) Delete(ctx context.Context, id int64) (*pb.DeleteProductResp, error) {
	return c.client.Delete(ctx, &pb.DeleteProductReq{ Id: id })
}
