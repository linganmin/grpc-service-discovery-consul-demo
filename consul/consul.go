package consul

import (
    "fmt"
    "time"

    "github.com/hashicorp/consul/api"
)

type ServiceEndpoint struct {
    IP   string
    Port int
    Tag  []string
    Name string
}

type con struct {
    client *api.Client
}

type ConService interface {
    RegisterService(sd *ServiceEndpoint) error
}

const (
    healthCheckInterval = time.Second * 3
    deregisterTime      = time.Second * 30
)

func NewConsul(addr string) (*con, error) {

    client, err := client(addr)

    if err != nil {
        return nil, err
    }

    return &con{client: client}, nil
}

func client(addr string) (*api.Client, error) {

    conf := api.DefaultConfig()
    conf.Address = addr

    client, err := api.NewClient(conf)

    if err != nil {
        return nil, err
    }

    return client, nil
}

func (c *con) RegisterService(sd *ServiceEndpoint) error {

    agent := c.client.Agent()

    reg := &api.AgentServiceRegistration{
        ID:      fmt.Sprintf("%v-%v-%v", sd.Name, sd.IP, sd.Port),
        Name:    sd.Name,
        Tags:    sd.Tag,
        Port:    sd.Port,
        Address: sd.IP,
        Check: &api.AgentServiceCheck{
            Interval:                       healthCheckInterval.String(),
            GRPC:                           fmt.Sprintf("%v:%v", sd.IP, sd.Port),
            DeregisterCriticalServiceAfter: deregisterTime.String(), // 健康检查失败后注销时间
        },
    }

    if err := agent.ServiceRegister(reg); err != nil {
        return err
    }

    return nil
}
