package common

import (
	"encoding/json"
	"github.com/astaxie/beego/logs"
)

func convertLogLevel(level string) int {

	switch level {
	case "debug":
		return logs.LevelDebug
	case "trace":
		return logs.LevelTrace
	case "info":
		return logs.LevelInfo
	case "warn":
		return logs.LevelWarn
	case "error":
		return logs.LevelError
	}

	return logs.LevelDebug
}

func InitLogger(logName string, logLevel string) error {

	if logName == "" {
		return nil
	}

	config := make(map[string]interface{})
	config["filename"] = logName
	config["level"] = convertLogLevel(logLevel)

	configStr, err := json.Marshal(config)
	if err != nil {
		return err
	}

	logs.SetLogger(logs.AdapterFile, string(configStr))
	return nil
}
