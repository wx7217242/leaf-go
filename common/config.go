package common

import (
	"encoding/xml"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"strings"
)

const (
	leafForeverWorkIdFileName = "leaf_forever_workid"
)

type Config struct {
	ZookeeperHost   string `xml:"zookeeper_host"`
	LeafHost        string `xml:"leaf_host"`
	HttpPort        int    `xml:"http_port"`
	RpcServicePort  int    `xml:"rpc_service_port"`
	TimeThresholdMS int    `xml:"time_threshold_ms"` // 单位是毫秒
	LogName         string `xml:"log_name"`
	LogLevel        string `xml:"log_level"`

	LeafForeverWorkId string
	filename          string
}

func LoadLeafConfig(filename string) (Config, error) {

	var config Config
	config.filename = filename

	bytes, err := ioutil.ReadFile(filename)
	if err != nil {
		return config, err
	}

	err = xml.Unmarshal(bytes, &config)
	if err == nil {
		if strings.TrimSpace(config.LeafHost) == "127.0.0.1" ||
			strings.TrimSpace(config.LeafHost) == "0.0.0.0" {
			return config, fmt.Errorf("LeafHost cant't be 127.0.0.1 or 0.0.0.0")
		}
	}

	config.LeafForeverWorkId = ""

	// 读取leaf forever workid
	leafForeverWorkIdFullFileName, _ := config.getLeafForeverWorkIdFullFileName()
	bytes, err = ioutil.ReadFile(leafForeverWorkIdFullFileName)
	if err != nil {
		if !os.IsNotExist(err) {
			return config, err
		}
	} else {
		config.LeafForeverWorkId = string(bytes)
	}

	return config, nil
}

func (c Config) getLeafForeverWorkIdFullFileName() (string, error) {
	if c.filename == "" {
		return "", errors.New("config filename can't be empty")
	}

	// 读取leaf forever workid
	fullFileName := ""
	lastIndex := strings.LastIndex(c.filename, string(os.PathSeparator))
	if lastIndex == -1 {
		fullFileName = leafForeverWorkIdFileName
	} else {
		fullFileName = c.filename[lastIndex+1:] + leafForeverWorkIdFileName
	}

	return fullFileName, nil
}

func (c Config) SaveLeafForeverWorkId() error {

	fullFileName, err := c.getLeafForeverWorkIdFullFileName()
	if err != nil {
		return err
	}

	return ioutil.WriteFile(fullFileName, []byte(c.LeafForeverWorkId), 0644)
}
