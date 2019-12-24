使用微服务发现组件：etcd

```
drwxr-xr-x 1 hello 197121    0 11月 28 11:32 cli/
-rw-r--r-- 1 hello 197121 2685 11月 28 10:20 registry.go
drwxr-xr-x 1 hello 197121    0 11月 28 11:31 ser/
-rw-r--r-- 1 hello 197121 2516 11月 28 10:20 watcher.go

```
server.go
```
package main

import (
	"atestetcd/chen"
	"flag"
	"fmt"
	"go.etcd.io/etcd/clientv3"
	"atestetcd/proto"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"log"
	"net"
	"sync"
	// "time"
)

var nodeID = flag.String("node", "node1", "node ID")
var port = flag.Int("port", 8080, "listening port")

type RpcServer struct {
	addr string
	s    *grpc.Server
}

func NewRpcServer(addr string) *RpcServer {
	s := grpc.NewServer()
	rs := &RpcServer{
		addr: addr,
		s:    s,
	}
	return rs
}

func (s *RpcServer) Run() {
	listener, err := net.Listen("tcp", s.addr)
	if err != nil {
		log.Printf("failed to listen: %v", err)
		return
	}
	log.Printf("rpc listening on:%s", s.addr)

	proto.RegisterEchoServiceServer(s.s, s)
	s.s.Serve(listener)
}

func (s *RpcServer) Stop() {
	s.s.GracefulStop()
}

func (s *RpcServer) Echo(ctx context.Context, req *proto.EchoReq) (*proto.EchoRsp, error) {
	text := "Hello " + req.EchoData + ", I am " + *nodeID
	log.Println(text)

	return &proto.EchoRsp{EchoData: text}, nil
}

func StartService() {
	etcdConfg := clientv3.Config{
		Endpoints: []string{"http://127.0.0.1:2379"},
	}

	registry, err := chen.NewRegistry(
		chen.Option{
			EtcdConfig:  etcdConfg,
			RegistryDir: "D:\tcd",
			ServiceName: "test",
			NodeID:      *nodeID,
			NData: chen.NodeData{
				Addr: fmt.Sprintf("127.0.0.1:%d", *port),
				//Metadata: map[string]string{"weight": "1"},
			},
			Ttl: 10, // * time.Second,
		})
	if err != nil {
		log.Panic(err)
		return
	}
	server := NewRpcServer(fmt.Sprintf("0.0.0.0:%d", *port))
	wg := sync.WaitGroup{}

	wg.Add(1)
	go func() {
		server.Run()
		wg.Done()
	}()

	wg.Add(1)
	go func() {
		registry.Register()
		wg.Done()
	}()

	//stop the server after one minute
	//go func() {
	//	time.Sleep(time.Minute)
	//	server.Stop()
	//	registry.Deregister()
	//}()

	wg.Wait()
}

//go run server.go -node node1 -port 28544
//go run server.go -node node2 -port 18562
//go run server.go -node node3 -port 27772
func main() {
	flag.Parse()
	StartService()
}
```

registry.go
```
package chen

import (
	"encoding/json"
	"fmt"

	etcd3 "go.etcd.io/etcd/clientv3"
	"go.etcd.io/etcd/etcdserver/api/v3rpc/rpctypes"
	"github.com/golang/glog"
	"golang.org/x/net/context"
	"google.golang.org/grpc/grpclog"
	"time"
)

type EtcdReigistry struct {
	etcd3Client *etcd3.Client
	key         string
	value       string
	ttl         time.Duration
	ctx         context.Context
	cancel      context.CancelFunc
}

type Option struct {
	EtcdConfig  etcd3.Config
	RegistryDir string
	ServiceName string
	NodeID      string
	NData       NodeData
	Ttl         time.Duration
}

type NodeData struct {
	Addr     string
	Metadata map[string]string
}

func NewRegistry(option Option) (*EtcdReigistry, error) {
	client, err := etcd3.New(option.EtcdConfig)
	if err != nil {
		return nil, err
	}

	val, err := json.Marshal(option.NData)
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithCancel(context.Background())
	registry := &EtcdReigistry{
		etcd3Client: client,
		key:         option.RegistryDir + "/" + option.ServiceName + "/" + option.NodeID,
		value:       string(val),
		ttl:         option.Ttl,
		ctx:         ctx,
		cancel:      cancel,
	}

	fmt.Println(registry)
	return registry, nil
}

func (e *EtcdReigistry) Register() error {

	insertFunc := func() error {
		// fmt.Println("Grant: ", e)
		resp, err := e.etcd3Client.Grant(e.ctx, 10) // int64(e.ttl))
		if err != nil {
			fmt.Println("Grant error: ", err)
			return err
		}
		_, err = e.etcd3Client.Get(e.ctx, e.key)
		if err != nil {
			if err == rpctypes.ErrKeyNotFound {
				if _, err := e.etcd3Client.Put(e.ctx, e.key, e.value, etcd3.WithLease(resp.ID)); err != nil {
					grpclog.Printf("grpclb: set key '%s' with ttl to etcd3 failed: %s", e.key, err.Error())
				}
			} else {
				grpclog.Printf("grpclb: key '%s' connect to etcd3 failed: %s", e.key, err.Error())
			}
			return err
		} else {
			// refresh set to true for not notifying the watcher
			// fmt.Println("refresh: ", resp)
			if _, err := e.etcd3Client.Put(e.ctx, e.key, e.value, etcd3.WithLease(resp.ID)); err != nil {
				grpclog.Printf("grpclb: refresh key '%s' with ttl to etcd3 failed: %s", e.key, err.Error())
				return err
			}
		}
		return nil
	}

	err := insertFunc()
	if err != nil {
		return err
	}

	ticker := time.NewTicker(e.ttl / 5)
	for {
		select {
		case <-ticker.C:
			insertFunc()
		case <-e.ctx.Done():
			ticker.Stop()
			if _, err := e.etcd3Client.Delete(context.Background(), e.key); err != nil {
				grpclog.Printf("grpclb: deregister '%s' failed: %s", e.key, err.Error())
			}
			glog.Info("Delete...")
			return nil
		}
	}

	return nil
}

func (e *EtcdReigistry) Deregister() error {
	e.cancel()
	return nil
}

```

client.go

```
package main

import (
	"atestetcd/chen"
	"errors"
	"fmt"
	"go.etcd.io/etcd/clientv3"
	etcd3 "go.etcd.io/etcd/clientv3"

	//"github.com/nebulaim/telegramd/baselib/grpc_util/load_balancer"
	"google.golang.org/grpc/naming"

	//"github.com/nebulaim/telegramd/baselib/grpc_util/service_discovery/etcd3"
	"github.com/nebulaim/telegramd/baselib/grpc_util/service_discovery/examples/proto"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"log"
	"time"

	etcdnaming  "go.etcd.io/etcd/clientv3/naming"
)

type EtcdResolver struct {
	Config      etcd3.Config
	RegistryDir string
	ServiceName string
}

func NewResolver(registryDir, serviceName string, cfg etcd3.Config) naming.Resolver {
	return &EtcdResolver{RegistryDir: registryDir, ServiceName: serviceName, Config: cfg}
}

// Resolve to resolve the service from etcd
func (er *EtcdResolver) Resolve(target string) (naming.Watcher, error) {
	if er.ServiceName == "" {
		return nil, errors.New("no service name provided")
	}
	client, err := etcd3.New(er.Config)
	if err != nil {
		return nil, err
	}

	key := fmt.Sprintf("%s/%s", er.RegistryDir, er.ServiceName)
	return chen.NewEtcdWatcher(key, client), nil
}

func main() {
	/*etcdConfg := clientv3.Config{
		Endpoints: []string{"http://127.0.0.1:2379"},
	}*/
	cli, _ := clientv3.NewFromURL("http://localhost:2379")
	r := &etcdnaming.GRPCResolver{Client: cli}
	b := grpc.RoundRobin(r)

	c, err := grpc.Dial("", grpc.WithInsecure(), grpc.WithBalancer(b), grpc.WithTimeout(time.Second*5))
	if err != nil {
		log.Printf("grpc dial: %s", err)
		return
	}
	defer c.Close()

	client := proto.NewEchoServiceClient(c)

	for i := 0; i < 1000; i++ {
		resp, err := client.Echo(context.Background(), &proto.EchoReq{EchoData: "round robin"})
		if err != nil {
			log.Println("aa:", err)
			time.Sleep(time.Second)
			continue
		}
		log.Printf(resp.EchoData)
		time.Sleep(time.Second)
	}
}

```

wacther.go
```
package chen

import (
	"encoding/json"
	etcd3 "go.etcd.io/etcd/clientv3"
	"go.etcd.io/etcd/mvcc/mvccpb"
	"golang.org/x/net/context"
	"google.golang.org/grpc/grpclog"
	"google.golang.org/grpc/naming"
)

// EtcdWatcher is the implementation of grpc.naming.Watcher
type EtcdWatcher struct {
	key     string
	client  *etcd3.Client
	updates []*naming.Update
	ctx     context.Context
	cancel  context.CancelFunc
}

func (w *EtcdWatcher) Close() {
	w.cancel()
}

func NewEtcdWatcher(key string, cli *etcd3.Client) naming.Watcher {
	ctx, cancel := context.WithCancel(context.Background())
	w := &EtcdWatcher{
		key:     key,
		client:  cli,
		ctx:     ctx,
		updates: make([]*naming.Update, 0),
		cancel:  cancel,
	}
	return w
}

func (w *EtcdWatcher) Next() ([]*naming.Update, error) {
	updates := make([]*naming.Update, 0)

	if len(w.updates) == 0 {
		// query addresses from etcd
		resp, err := w.client.Get(w.ctx, w.key, etcd3.WithPrefix())
		if err == nil {
			addrs := extractAddrs(resp)
			if len(addrs) > 0 {
				for _, v := range addrs {
					updates = append(updates, &naming.Update{Op: naming.Add, Addr: v.Addr, Metadata: &v.Metadata})
				}
				w.updates = updates
				return updates, nil
			}
		} else {
			grpclog.Println("Etcd Watcher Get key error:", err)
		}
	}

	// generate etcd Watcher
	rch := w.client.Watch(w.ctx, w.key, etcd3.WithPrefix())
	for wresp := range rch {
		for _, ev := range wresp.Events {
			switch ev.Type {
			case mvccpb.PUT:
				nodeData := NodeData{}
				err := json.Unmarshal([]byte(ev.Kv.Value), &nodeData)
				if err != nil {
					grpclog.Println("Parse node data error:", err)
					continue
				}
				updates = append(updates, &naming.Update{Op: naming.Add, Addr: nodeData.Addr, Metadata: &nodeData.Metadata})
			case mvccpb.DELETE:
				nodeData := NodeData{}
				err := json.Unmarshal([]byte(ev.Kv.Value), &nodeData)
				if err != nil {
					grpclog.Println("Parse node data error:", err)
					continue
				}
				updates = append(updates, &naming.Update{Op: naming.Delete, Addr: nodeData.Addr, Metadata: &nodeData.Metadata})
			}
		}
	}
	return updates, nil
}

func extractAddrs(resp *etcd3.GetResponse) []NodeData {
	addrs := []NodeData{}

	if resp == nil || resp.Kvs == nil {
		return addrs
	}

	for i := range resp.Kvs {
		if v := resp.Kvs[i].Value; v != nil {
			nodeData := NodeData{}
			err := json.Unmarshal(v, &nodeData)
			if err != nil {
				grpclog.Println("Parse node data error:", err)
				continue
			}
			addrs = append(addrs, nodeData)
		}
	}

	return addrs
}

```



