# 一、什么是go-micro  
  
- [ ] Go Micro是一个插件化的基础框架，基于此可以构建微服务。Micro的设计哲学是『可插拔』的插件化架构。在架构之外，它默认实现了consul作为服务发现，通过http进行通信，通过protobuf和json进行编解码。我们一步步深入下去。



Go Micro是：  
-  一个用Golang编写的包  
-  一系列插件化的接口定义  
-  基于RPc  
-  Go Micro为下面的模块定义了接口：
1. 服务发现
1. 编解码
1. 服务端、客户端
1. 订阅、发布消息

# 二、使用go-micro编写微服务

## 安装protoc   
1.github上下载一个cpp包：https://github.com/google/protobuf/releases
make  make install安装即可  
2.protoc-gen-go  
go get -u github.com/golang/protobuf/protoc-gen-go  
3.安装protoc-gen-micro  
go get github.com/micro/protoc-gen-micro

## 安装Consul
micro默认使用consul作为微服务发现  

```
Consul is used as the default service discovery system.

Discovery is pluggable. Find plugins for etcd, kubernetes, zookeeper and more in the micro/go-plugins repo.
```
[https://www.consul.io/intro/getting-started/install.html](https://note.youdao.com/)

启动cansul方式参考如下：**注意修改自己-data-dir目录路劲**

```
consul agent -server  -node chenxun-server -bind=192.168.199.62  -data-dir D:\工作文件备份\consul_1.0.0_windows_amd64\tmp1  -ui

#  consul agent -server -bootstrap-expect 1 -node chenxun-server -bind=192.168.199.62 -data-dir c:/tmp

# ./consul agent -server -bootstrap-expect 1  -data-dir /tmp/consul -node=chenxun-server -bind=192.168.145.130 -ui

```

准备proto文件：  文件保存为chenxun.proto，名称随便写，在实际项目中根据项目写就好了  

chenxun.proto
```
syntax = "proto3";

service Greeter {
	rpc Hello(HelloRequest) returns (HelloResponse) {}
}

message HelloRequest {
	string name = 1;
}

message HelloResponse {
	string greeting = 2;
}
```

Generate the proto：


```
protoc --proto_path=$GOPATH/src:. --micro_out=. --go_out=. chenxun.proto
```

执行命令后能看到下面文件：
```
-rw-r--r--.  1 root  root      2441 Jul  7 10:38 chenxun.micro.go
-rw-r--r--.  1 root  root      2914 Jul  7 10:38 chenxun.pb.go
-rw-r--r--.  1 root  root       185 Jul  6 11:36 chenxun.proto

```

比如我把这三个文件放在gopath路劲下面的src目录下面的mygoproject/gomirco

那么在import的时候写：  import  "mygoproject/gomirco"

Service端代码：

```
package main

import (
	"context"
	"fmt"

	micro "github.com/micro/go-micro"
	proto "mygoproject/gomirco" //这里写你的proto文件放置路劲
)

type Greeter struct{}

func (g *Greeter) Hello(ctx context.Context, req *proto.HelloRequest, rsp *proto.HelloResponse) error {
	rsp.Greeting = "Hello " + req.Name
	return nil
}

func main() {
	// Create a new service. Optionally include some options here.
	service := micro.NewService(
		micro.Name("greeter"),
	)

	// Init will parse the command line flags.
	service.Init()

	// Register handler
	proto.RegisterGreeterHandler(service.Server(), new(Greeter))

	// Run the server
	if err := service.Run(); err != nil {
		fmt.Println(err)
	}
}
```

Client端代码：

```
package main

import (
	"context"
	"fmt"

	micro "github.com/micro/go-micro"
	proto "mygoproject/gomirco" //这里写你的proto文件放置路劲
)


func main() {
	// Create a new service. Optionally include some options here.
	service := micro.NewService(micro.Name("greeter.client"))
	service.Init()

	// Create new greeter client
	greeter := proto.NewGreeterService("greeter", service.Client())

	// Call the greeter
	rsp, err := greeter.Hello(context.TODO(), &proto.HelloRequest{Name: "John"})
	if err != nil {
		fmt.Println(err)
	}

	// Print response
	fmt.Println(rsp.Greeting)
}
```

运行service：

```
go run examples/service/main.go
```

运行client:

```
go run examples/client/main.go
```
