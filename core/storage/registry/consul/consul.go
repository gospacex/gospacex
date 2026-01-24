package consul

import (
	"fmt"
	"log"
	"net"
	"strconv"
	"micro/srv/internal/conf"

	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"

	"github.com/hashicorp/consul/api"
)

var (
	RGClient *api.Client
)

// consul 定义一个consul结构体，其内部有一个`*api.Client`字段。
type consulService struct {
	client *api.Client
}

// NewConsul 连接至consul服务返回一个consul对象
func NewConsul(addr string) (*consulService, error) {
	cfg := api.DefaultConfig()
	cfg.Address = addr
	c, err := api.NewClient(cfg)
	if err != nil {
		return nil, err
	}
	return &consulService{c}, nil
}

// RegisterService 将gRPC服务注册到consul
func (c *consulService) RGService(serviceName string, ip string, port int) error {
	// 健康检查
	check := &api.AgentServiceCheck{
		GRPC:     fmt.Sprintf("%s:%d", ip, port), // 这里一定是外部可以访问的地址
		Timeout:  "10s",                          // 超时时间
		Interval: "10s",                          // 运行检查的频率
		// 指定时间后自动注销不健康的服务节点
		// 最小超时时间为1分钟，收获不健康服务的进程每30秒运行一次，因此触发注销的时间可能略长于配置的超时时间。
		DeregisterCriticalServiceAfter: "1m",
	}
	srv := &api.AgentServiceRegistration{
		ID:      fmt.Sprintf("%s-%s-%d", serviceName, ip, port), // 服务唯一ID
		Name:    serviceName,                                    // 服务名称
		Tags:    []string{"srv"},                                // 为服务打标签
		Address: ip,
		Port:    port,
		Check:   check,
	}
	return c.client.Agent().ServiceRegister(srv)
}

func Init(cfg *conf.ConsulConfig) (err error) {
	lis, err := net.Listen("tcp", net.JoinHostPort(cfg.Host, strconv.Itoa(cfg.Port)))
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	s := grpc.NewServer()
	healthSrv := health.NewServer()
	healthpb.RegisterHealthServer(s, healthSrv)
	healthSrv.SetServingStatus("helloworld.Greeter", healthpb.HealthCheckResponse_SERVING)
	consulC := &consulService{}
	fmt.Println("-------->consul init:", fmt.Sprintf("%s:%s", cfg.Host, strconv.Itoa(cfg.Port)))
	consulC, err = NewConsul(fmt.Sprintf("%s:%s", cfg.Host, strconv.Itoa(cfg.Port)))
	if err != nil {
		return
	}
	RGClient = consulC.client
	//判断，如果cfg.Services不为空，那么就注册服务
	if len(cfg.Services) != 0 {
		for _, v := range cfg.Services {
			fmt.Println(v)
			consulC.RGService(v.Name, v.Host, v.Port)
		}
	}
	if err := s.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
	return
}
