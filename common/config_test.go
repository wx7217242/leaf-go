package common

import (
	"testing"
	"log"
)

func TestLoadConfig(t *testing.T) {

	filename := "../config.xml"
	config, err := LoadLeafConfig(filename)
	if err != nil {
		t.Errorf("load %s failed, err %v", filename, err)
		log.Fatal("")
	}

	t.Logf("load config %s succeed, config %v", filename, config)
}
