package etcd

import (
	"fmt"
	"time"

	"go.etcd.io/etcd/client/v3"
)

var (
	EtcdClient *clientv3.Client
)

func Init(endpoints []string) {
	EtcdClient, err := clientv3.New(clientv3.Config{
		Endpoints:   endpoints,
		DialTimeout: 5 * time.Second,
	})
	if err != nil {
		fmt.Printf("connect to etcd failed, err:%v\n", err)
		return
	}
	fmt.Println("connect to etcd success")
	defer EtcdClient.Close()
}
