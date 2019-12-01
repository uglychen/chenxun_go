
package main

import (
    "context"
    "fmt"
    resolver "learn_grpc_consul/registerConsul"
    proto "learn_grpc_consul/helloWorld"
    "google.golang.org/grpc"
    "google.golang.org/grpc/balancer/roundrobin"
    "log"
    "time"

)

func main() {

    schema, err := resolver.GenerateAndRegisterConsulResolver("192.168.1.8:8500", "HelloService")
    if err != nil {
        log.Fatal("init consul resovler err", err.Error())
    }

    // Set up a connection to the server.
    conn, err := grpc.Dial(fmt.Sprintf("%s:///HelloWorldService", schema), grpc.WithInsecure(), grpc.WithBalancerName(roundrobin.Name))
    if err != nil {
        log.Fatalf("did not connect: %v", err)
    }
    defer conn.Close()
    c := proto.NewHelloServerClient(conn)

    // Contact the server and print out its response.
    name := "nosixtools"

    for {
        ctx, cancel := context.WithTimeout(context.Background(), time.Second)
        defer cancel()

        r, err := c.SayHello(ctx, &proto.HelloRequest{Name: name})
        if err != nil {
            log.Println("could not greet: %v", err)

        } else {
            log.Printf("Hello: %s", r.Message)
        }
        time.Sleep(time.Second)
    }

}