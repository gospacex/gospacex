package emqx

import (
	mqtt "github.com/eclipse/paho.mqtt.golang"
)

var (
	MQClient *mqtt.Client
)

func Init(enableChain bool, host string) (db mqtt.Client, err error) {
	opts := mqtt.NewClientOptions().AddBroker("tcp://localhost:1883")
	MQClient := mqtt.NewClient(opts)
	if token := MQClient.Connect(); token.Wait() && token.Error() != nil {
		panic(token.Error())
	}
	return MQClient, err
}
