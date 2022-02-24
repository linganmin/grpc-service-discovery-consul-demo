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

type consulResolver struct {
    address              string
    tag                  string
    wg                   sync.WaitGroup
    cc                   resolver.ClientConn
    name                 string
    disableServiceConfig bool
    lastIndex            uint64
}

// ResolveNow 更新逻辑在 watcher 里处理掉了
func (c *consulResolver) ResolveNow(o resolver.ResolveNowOptions) {

}

// Close 暂时不处理
func (c *consulResolver) Close() {

}

func (c *consulResolver) watcher() {

    defer c.wg.Done()

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

func (b *consulBuilder) Scheme() string {
    return schemeName
}

func parseTarget(target url.URL) (host, port, name string, tag string, err error) {

    tag = target.Query().Get("tag")

    return target.Host, target.Port(), strings.Replace(target.Path, "/", "", -1), tag, err
}
