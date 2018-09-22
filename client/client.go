package client

import (
	"context"
	"encoding/json"
	"fmt"
	"goetcd/server"
	"log"
	"sync"
	"time"

	"github.com/coreos/etcd/clientv3"
)

//Master master node message
type Master struct {
	Path   string
	Nodes  map[string]*Node
	Client *clientv3.Client
	sync.Mutex
}

//Node node message
type Node struct {
	State bool
	Key   string
	Info  *server.ServiceInfo
}

//NewMaster init the struct
func NewMaster(endpoints []string, watchPath string) (*Master, error) {
	cli, err := clientv3.New(clientv3.Config{
		Endpoints:   endpoints,
		DialTimeout: time.Second,
	})
	fmt.Println(">>>>>>>>>>>>>>>>>", cli)
	if err != nil {
		log.Fatal(err)
		return nil, err
	}

	master := &Master{
		Path:   watchPath,
		Nodes:  make(map[string]*Node),
		Client: cli,
	}

	//go master.WatchNodes()
	return master, err
}

//AddNode add node message into the map
func (m *Master) AddNode(key string, info *server.ServiceInfo) {
	node := &Node{
		State: true,
		Key:   key,
		Info:  info,
	}
	m.Lock()
	m.Nodes[node.Key] = node
	m.Unlock()
}

//WatchNodes watch node message
func (m *Master) WatchNodes() {
	rch := m.Client.Watch(context.Background(), m.Path, clientv3.WithPrefix())
	for wresp := range rch {
		for _, ev := range wresp.Events {
			switch ev.Type {
			case clientv3.EventTypePut:
				fmt.Printf("[%s] %q:%q\n", ev.Type, ev.Kv.Key, ev.Kv.Value)
				info := GetServiceInfo(ev)
				m.AddNode(string(ev.Kv.Key), info)
			case clientv3.EventTypeDelete:
				fmt.Printf("[%s] %q : %q\n", ev.Type, ev.Kv.Key, ev.Kv.Value)
				delete(m.Nodes, string(ev.Kv.Key))
			}
		}
	}
}

//GetServiceInfo get service info message
func GetServiceInfo(ev *clientv3.Event) *server.ServiceInfo {
	info := &server.ServiceInfo{}
	err := json.Unmarshal([]byte(ev.Kv.Value), info)
	if err != nil {
		log.Fatal(err)
	}
	return info
}
