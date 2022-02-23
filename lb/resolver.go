package lb

import (
    "fmt"
    "log"
    "net/url"
    "strings"
    "sync"

    "github.com/hashicorp/consul/api"
    "google.golang.org/grpc/resolver"
)

// ----------- resolver 做的事情
// 解析 target 获取 scheme
// 调用 resolver.Get 根据 scheme 拿到对应的 Builder
// 调用 Builder.Build 方法
// - 解析 target
// - 获取服务地址的信息
// - 调用 ClientConn.NewAddress 和 NewServiceConfig 这俩 callback 把服务信息传递给上层的调用方
// - 返回 Resolver 接口实例给上层
// 上层可以通过Resolver.ResolveNow 方法主动刷新服务信息

type consulResolver struct {
    address              string
    tag                  string
    wg                   sync.WaitGroup
    cc                   resolver.ClientConn
    name                 string
    disableServiceConfig bool
    lastIndex            uint64
}

// ResolveNow 和 Consul 保持了订阅发布关系，不需要定时刷新
func (c *consulResolver) ResolveNow(o resolver.ResolveNowOptions) {

}

// Close 暂时不处理
func (c *consulResolver) Close() {

}

func (c *consulResolver) watcher() {

    conf := api.DefaultConfig()

    conf.Address = c.address

    client, err := api.NewClient(conf)

    if err != nil {
        log.Fatalf("create consul client err:%+v", err)
    }

    for {

        services, meta, err := client.Health().Service(c.name, c.tag, true, &api.QueryOptions{WaitIndex: c.lastIndex})

        if len(services) == 0 {
            panic(fmt.Sprintf("no available endpoints for server:%s,tag:%s", c.name, c.tag))
        }
        if err != nil {
            fmt.Printf("retrieving instances from consul err: %+v", err)
            continue
        }
        c.lastIndex = meta.LastIndex

        var endpoints []resolver.Address

        for _, service := range services {
            endpoints = append(endpoints, resolver.Address{
                Addr: fmt.Sprintf("%v:%v", service.Service.Address, service.Service.Port),
            })
        }

        _ = c.cc.UpdateState(resolver.State{
            Addresses: endpoints,
        })
    }
}

// ------------

const (
    schemeName = "consul"
)

type consulBuilder struct {
}

func Init() {
    resolver.Register(NewBuilder())
}

func NewBuilder() resolver.Builder {
    return &consulBuilder{}
}

func (b *consulBuilder) Build(target resolver.Target, cc resolver.ClientConn, opts resolver.BuildOptions) (resolver.Resolver, error) {

    host, _, name, tag, err := parseTarget(target.URL)

    if err != nil {
        return nil, err
    }

    c := &consulResolver{
        address:              host,
        tag:                  tag,
        cc:                   cc,
        name:                 name,
        disableServiceConfig: opts.DisableServiceConfig,
        lastIndex:            0,
    }

    c.wg.Add(1)
    go c.watcher()

    return c, nil
}

func (b consulBuilder) Scheme() string {
    return schemeName
}

func parseTarget(target url.URL) (host, port, name string, tag string, err error) {

    tag = target.Query().Get("tag")

    return target.Host, target.Port(), strings.Replace(target.Path, "/", "", -1), tag, err
}
