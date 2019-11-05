package main

import (
    "context"
    "fmt"

    pb "learn_grpc_consul/helloWorld"
    "google.golang.org/grpc"
    "log"
    "time"
    resolver "rpc_chen/examples/discovery/resolver"
    "flag"
)

var (
    gprc_serviceName = flag.String("serviceName", "HelloWorldService", "the name used register consul")
    gprc_host = flag.String("host", "192.168.1.8", "ip address")
    gprc_port = flag.Int("port", 9000, "the service listen port")
    gprc_consul_port = flag.Int("consul_port", 8500, "connect consual http port")
)

func main() {

    schema, err := resolver.GenerateAndRegisterConsulResolver("192.168.1.8:8500", *gprc_serviceName)
    if err != nil {
        log.Fatal("init consul resovler err", err.Error())
    }

    // Set up a connection to the server.
    conn, err := grpc.Dial(fmt.Sprintf("%s:///"+ *gprc_serviceName, schema), grpc.WithInsecure())
    if err != nil {
        log.Fatalf("did not connect: %v", err)
    }
    defer conn.Close()
    c := pb.NewHelloServerClient(conn)

    // Contact the server and print out its response.
    name := "chenxun"

    for {
        ctx, cancel := context.WithTimeout(context.Background(), time.Second)
        defer cancel()

        r, err := c.SayHello(ctx, &pb.HelloRequest{Name: name})
        if err != nil {
            log.Println("could not greet: %v", err)

        } else {
            log.Printf("Hello: %s", r.Message)
        }
        time.Sleep(2*time.Second)
    }
}
