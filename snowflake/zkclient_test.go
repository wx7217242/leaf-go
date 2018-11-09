package snowflake

import (
	"fmt"
	"github.com/samuel/go-zookeeper/zk"
	"testing"
	"time"
)


const zkServerHost = "192.168.0.179:2181"

func TestZKLeafClient_CreateLeafForeverNode(t *testing.T) {

	conn, _, err := zk.Connect([]string{zkServerHost}, time.Second*5)
	if err != nil {
		panic(err)
	}

	value := "hello world"

	workid, err := conn.Create(fmt.Sprintf("%s/", leafForeverPrefix),
		[]byte(value), zk.FlagSequence, zk.WorldACL(zk.PermAll))

	if err != nil {
		panic(err)
	}

	t.Logf("create leaf forever node succeed workid %v", workid)

}

func TestZKLeafClient_GetLeafForeverNodeData(t *testing.T) {


	conn, _, err := zk.Connect([]string{zkServerHost}, time.Second*5)
	if err != nil {
		panic(err)
	}

	bytes, _, err := conn.Get("/leaf-forever/0000000007")
	if err != nil {
		panic(err)
	}

	t.Logf("data: %s %v", string(bytes), bytes)
}
