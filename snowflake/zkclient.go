package snowflake

import (
	"fmt"
	"github.com/samuel/go-zookeeper/zk"
	"time"
)

type ZKLeafClient struct {
	Conn    *zk.Conn
	Event   <-chan zk.Event
	servers []string
}

func (c *ZKLeafClient) Connect(servers []string) error {

	c.servers = servers

	conn, events, err := zk.Connect(servers, time.Second*10)
	if err != nil {
		return err
	}

	c.Conn = conn
	c.Event = events

	return nil
}

// 创建持久节点
func (c *ZKLeafClient) CreateLeafForeverNode() (string, error) {

	err := c.createNodeIfNotExists(leafForeverPrefix)
	if err != nil {
		return "", err
	}

	path := fmt.Sprintf("%s/", leafForeverPrefix)
	s, err := c.Conn.Create(path, []byte{}, zk.FlagSequence, zk.WorldACL(zk.PermAll))

	if err != nil {
		return "", err
	}

	return s, nil
}

// 获取持久节点
func (c *ZKLeafClient) GetLeafForeverNodeData(workid string) (string, error) {

	bytes, _, err := c.Conn.Get(workid)
	if err != nil {
		return "", err
	}

	return string(bytes), nil
}

// 设置持久节点的信息
func (c *ZKLeafClient) SetLeafForeverNodeData(workid string, data string) error {
	_, err := c.Conn.Set(workid, []byte(data), -1)
	return err
}

func (c *ZKLeafClient) IsLeafTempNodeExist(rpcHost string) (bool, error) {

	exists, _, err := c.Conn.Exists(fmt.Sprintf("%s/%s", leafTempPrefix, rpcHost))
	return exists, err
}

func (c *ZKLeafClient) CreateLeafTempNode(rpcHost string) (string, error) {

	err := c.createNodeIfNotExists(leafTempPrefix)
	if err != nil {
		return "", err
	}

	path := fmt.Sprintf("%s/%s", leafTempPrefix, rpcHost)
	s, err := c.Conn.Create(path, []byte{}, zk.FlagEphemeral, zk.WorldACL(zk.PermAll))

	if err != nil {
		return "", err
	}

	return s, nil
}

func (c *ZKLeafClient) GetLeafTempChildNodeData(workid string) (string, error) {

	bytes, _, err := c.Conn.Get(workid)
	if err != nil {
		return "", err
	}

	return string(bytes), nil
}

func (c *ZKLeafClient) SetLeafTempChildNodeData(workid string, data string) error {
	_, err := c.Conn.Set(workid, []byte(data), -1)
	return err
}

func (c *ZKLeafClient) WatchLeafTempChildNode(workid string) {
	// TODO: strings, stat, events, err := c.Conn.ChildrenW(workid)

}

func (c *ZKLeafClient) GetLeafTempChildren() ([]string, error) {
	err := c.createNodeIfNotExists(leafTempPrefix)
	if err != nil {
		return []string{}, err
	}

	strings, _, err := c.Conn.Children(leafTempPrefix)
	return strings, err
}

func (c *ZKLeafClient) DeleteNode(workid string, version int) {
	c.Conn.Delete(workid, int32(version))
}

func (c *ZKLeafClient) DeleteLeafTempNode(rpcHost string) {
	c.Conn.Delete(fmt.Sprintf("%s/%s", leafTempPrefix, rpcHost), -1)
}

func (c *ZKLeafClient) createNodeIfNotExists(path string) error {

	exists, _, err := c.Conn.Exists(path)
	if err != nil {
		return err
	}

	if !exists {
		_, err := c.Conn.Create(path, []byte{}, 0, zk.WorldACL(zk.PermAll))
		if err != nil {
			return err
		}
	}
	return nil
}
