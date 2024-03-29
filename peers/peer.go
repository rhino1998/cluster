package peers

import (
	"fmt"
	"github.com/rhino1998/cluster/common"
	"github.com/rhino1998/cluster/info"
	"github.com/rhino1998/cluster/tasks"
	"github.com/rhino1998/cluster/util"
	"log"
	"net"
	"net/rpc"
	"sync"
	"sync/atomic"
	"time"
)

//A peer node
type Peer struct {
	client *rpc.Client
	Addr   string `json:"addr"`
	conn   net.Conn
	info.Info
	id   uint64
	dead uint32
	wg   *sync.WaitGroup
}

func ThisPeer(intaddr, addr string, description info.Info) *Peer {
	description.IntAddr = intaddr
	return &Peer{Addr: addr, Info: description, id: util.IpValue(addr), dead: 0}
}

//initializes a new peer
func NewPeer(intaddr, locaddr, remaddr string) (*Peer, error) {

	locip, _, _ := net.SplitHostPort(locaddr)
	remip, remport, _ := net.SplitHostPort(remaddr)
	var conn net.Conn
	var err error
	conn, err = net.Dial("tcp", remaddr)
	if err != nil {
		return nil, err
	}
	client := rpc.NewClient(conn)

	var description info.Info
	err = client.Call("Node.Greet", &locaddr, &description)
	if err != nil {
		return nil, err
	}
	if locip == remip {
		rintip, _, _ := net.SplitHostPort(description.IntAddr)
		conn2, err := net.Dial("tcp", fmt.Sprintf("%v:%v", rintip, remport))
		if err == nil {
			client.Close()
			conn.Close()
			client = rpc.NewClient(conn2)
			conn = conn2
		}
	}
	peer := &Peer{Addr: remaddr, Info: description, dead: 0, id: util.IpValue(remaddr), client: client, conn: conn, wg: &sync.WaitGroup{}}
	go func() {
		for !peer.isDead() {
			if !peer.livecheck() {
				peer.kill()
			}
			select {
			case <-time.After(30 * time.Second):
			}
		}
	}()
	return peer, err
}

//Returns a peer id and satisfies Entry interface
func (self *Peer) Key() uint64 {
	return self.id
}

//Returns bool of iff the peer is dead
func (self *Peer) isDead() bool {
	return atomic.LoadUint32(&self.dead) == 1
}

//Kills the peer
func (self *Peer) kill() {
	log.Println(self.Addr, "killed")
	self.client.Close()
	self.conn.Close()
	atomic.StoreUint32(&self.dead, 1)
}

func (self *Peer) evict() {
	log.Println("Evicted")
	atomic.StoreUint32(&self.dead, 1)

	ch := make(chan struct{})
	go func() {
		self.wg.Wait()
		ch <- struct{}{}
	}()
	select {
	case <-ch:
	case <-time.After(20 * time.Second):
	}
	log.Println("killed")
	self.kill()
}

func (self *Peer) ping() (err error) {
	self.wg.Add(1)
	err = self.client.Call("Node.Ping", &struct{}{}, &struct{}{})
	self.wg.Done()
	return err
}

//gets the peers of the peer
func (self *Peer) getpeers(x int) (peers []string, err error) {
	self.wg.Add(1)
	err = self.client.Call("Node.GetPeers", &x, &peers)
	self.wg.Done()
	return peers, err
}

//Puts a value on the dht
func (self *Peer) put(item *common.Item) (success bool, err error) {
	self.wg.Add(1)
	err = self.client.Call("Node.Put", item, &success)
	self.wg.Done()
	return success, err
}

//Gets a value from the dht
func (self *Peer) get(key string) (data []byte, err error) {
	self.wg.Add(1)
	err = self.client.Call("Node.Get", &key, &data)
	self.wg.Done()
	return data, err
}

//Puts a task on the queue of the peer
func (self *Peer) AllocateTask(task *tasks.Task) (result []byte, err error) {
	self.wg.Add(1)
	err = self.client.Call("Node.AllocateTask", task, &result)
	self.wg.Done()
	return result, err
}

func (self *Peer) livecheck() bool {
	ch := make(chan struct{})
	go func() {
		self.ping()
		ch <- struct{}{}
	}()
	select {
	case <-ch:
		return true
	case <-time.After(20 * time.Second):
		return false

	}
}
