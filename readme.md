# 基于 Consul 作为 NameResolver 解析器实现 gRPC 服务发现的 Demo


## gRPC 的 Name Resolution

gRPC 中的默认 name-system 是 DNS，在客户端以插件形式提供了自定义 name-system 机制。

gRPC NameResolver 会根据 name-system 选择对应的解析器，用以解析用户提供的服务器名称，最终返回服务的地址列表（IP:Port）


[gRPC 名称解析文档](https://github.com/grpc/grpc/blob/master/doc/naming.md)


### resolver 源码

```go
// https://github.com/grpc/grpc-go/blob/master/resolver/resolver.go

// Package resolver defines APIs for name resolution in gRPC.
// All APIs in this package are experimental.
package resolver

import (
  "context"
  "net"
  "net/url"

  "google.golang.org/grpc/attributes"
  "google.golang.org/grpc/credentials"
  "google.golang.org/grpc/serviceconfig"
)

var (
  // m is a map from scheme to resolver builder.
  m = make(map[string]Builder)
  // defaultScheme is the default scheme to use.
  defaultScheme = "passthrough"
)

// Register 注册 resolver builder 到 m 中，在初始化的时候使用，线程不安全
func Register(b Builder) {
  m[b.Scheme()] = b
}

// Get returns the resolver builder registered with the given scheme.
//
// If no builder is register with the scheme, nil will be returned.
func Get(scheme string) Builder {
  if b, ok := m[scheme]; ok {
    return b
  }
  return nil
}

// SetDefaultScheme sets the default scheme that will be used. The default
// default scheme is "passthrough".
func SetDefaultScheme(scheme string) {
  defaultScheme = scheme
}

// GetDefaultScheme gets the default scheme that will be used.
func GetDefaultScheme() string {
  return defaultScheme
}

// Address 描述一个服务的地址信息
type Address struct {
  Addr string

  ServerName string

  // 包含了关于这个地址用于任意数据
  Attributes         *attributes.Attributes
  BalancerAttributes *attributes.Attributes
}

// BuildOptions 创建解析器的额外信息
type BuildOptions struct {
  // DisableServiceConfig indicates whether a resolver implementation should
  // fetch service config data.
  DisableServiceConfig bool
  DialCreds            credentials.TransportCredentials
  Dialer               func(context.Context, string) (net.Conn, error)
}

// State 与 ClientConn 相关的当前 Resolver 状态。
type State struct {
  // 最新的 target 解析出来的可用节点地址集
  Addresses []Address

  ServiceConfig *serviceconfig.ParseResult

  Attributes *attributes.Attributes
}

// ClientConn 用于通知服务信息更新的 callback 
type ClientConn interface {
  // UpdateState updates the state of the ClientConn appropriately.
  UpdateState(State) error
  // ReportError notifies the ClientConn that the Resolver encountered an
  // error.  The ClientConn will notify the load balancer and begin calling
  // ResolveNow on the Resolver with exponential backoff.
  ReportError(error)

  // ParseServiceConfig parses the provided service config and returns an
  // object that provides the parsed config.
  ParseServiceConfig(serviceConfigJSON string) *serviceconfig.ParseResult
}

// Target represents a target for gRPC, as specified in:
// https://github.com/grpc/grpc/blob/master/doc/naming.md.
// It is parsed from the target string that gets passed into Dial or DialContext
// by the user. And gRPC passes it to the resolver and the balancer.
//
// If the target follows the naming spec, and the parsed scheme is registered
// with gRPC, we will parse the target string according to the spec. If the
// target does not contain a scheme or if the parsed scheme is not registered
// (i.e. no corresponding resolver available to resolve the endpoint), we will
// apply the default scheme, and will attempt to reparse it.

// Target 请求目标地址解析出的对象
type Target struct {

  // URL contains the parsed dial target with an optional default scheme added
  // to it if the original dial target contained no scheme or contained an
  // unregistered scheme. Any query params specified in the original dial
  // target can be accessed from here.
  URL url.URL
}

// Builder 创建一个 resolver 并监听更新
type Builder interface {
  // Build creates a new resolver for the given target.
  //
  // gRPC dial calls Build synchronously, and fails if the returned error is
  // not nil.
  Build(target Target, cc ClientConn, opts BuildOptions) (Resolver, error)
  // Scheme returns the scheme supported by this resolver.
  // Scheme is defined at https://github.com/grpc/grpc/blob/master/doc/naming.md.
  Scheme() string
}

// ResolveNowOptions includes additional information for ResolveNow.
type ResolveNowOptions struct{}

// Resolver 解析器监视指定目标的更新，包括地址更新和服务配置更新。
type Resolver interface {
  // ResolveNow will be called by gRPC to try to resolve the target name
  // again. It's just a hint, resolver can ignore this if it's not necessary.
  //
  // It could be called multiple times concurrently.
  ResolveNow(ResolveNowOptions)
  // Close closes the resolver.
  Close()
}

// UnregisterForTesting removes the resolver builder with the given scheme from the
// resolver map.
// This function is for testing only.
func UnregisterForTesting(scheme string) {
  delete(m, scheme)
}

```

### resolver 做的事情

- 解析 target 获取 scheme
- 调用 resolver.Get 根据 scheme 拿到对应的 Builder
- 调用 Builder.Build 方法
  - 解析 target
  - 获取服务地址的信息
  - 调用 ClientConn.UpdateState  callback 把服务信息传递给上层的调用方
  - 返回 Resolver 接口实例给上层
- 上层可以通过 Resolver.ResolveNow 方法主动刷新服务信息

### 参考官方 dns_resolver 实现 consul_resolver

```go
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

// 实现了调用 consul 接口获取指定服务的可用节点
// WaitIndex 用于阻塞，直到有新的可用节点，避免重复刷新
// 将获取到的可用节点更新 c.cc.UpdateState
// 支持了 consul 的 tag 过滤，在 target 通过 query 参数传递
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

    // 解析 target 获取 consul 地址，服务名，服务tag
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

```

## grpc server

```go
// gRPC服务是使用Protobuf(PB)协议的，而PB提供了在运行时获取Proto定义信息的反射功能。
// grpc-go中的"google.golang.org/grpc/reflection"包就对这个反射功能提供了支持。
// 通过该反射我们就可以使用类似 grpcurl 的终端工具测试 rpc 接口了
reflection.Register(s)

// 健康检查
// 官方文档：https://github.com/grpc/grpc/blob/master/doc/health-checking.md
// gRPC-go 提供了健康检测库：https://pkg.go.dev/google.golang.org/grpc/health?tab=doc 把上面的文档接口化了。
grpc_health_v1.RegisterHealthServer(s, health.NewServer())

```

## grpc client

```go
  target := "consul://localhost:8500/hello.service" // schema:[//authority/]host[:port]/service[?query] 参考文档：https://github.com/grpc/grpc/blob/master/doc/naming.md
  ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
  defer cancel()

  conn, err := grpc.DialContext(ctx,
      target,
      grpc.WithTransportCredentials(insecure.NewCredentials()),
      grpc.WithDefaultServiceConfig(fmt.Sprintf(`{"loadBalancingConfig": [{"%s":{}}]}`, roundrobin.Name))) // 负载均衡策略，默认 pick_first ,文档：https://github.com/grpc/grpc/blob/master/doc/load-balancing.md

```


## Run

### 环境依赖

1. 和本地环境相通的 consul ，例如在本机使用 docker 启动一个 consul 节点
2. 将代码中的 consul 地址`localhost:8500`替换为可用地址
3. 执行`go mod tidy`处理依赖
4. `go run greeter_server/main.go` 启动服务，也可指定端口，例如：`go run greeter_server/main.go  -port 12124`, 可以去 consul dashboard 查看服务注册及健康检查状态，可以指定端口多启动几个节点
5. `go run greeter_client/main.go -name 小下` 发起客户端请求