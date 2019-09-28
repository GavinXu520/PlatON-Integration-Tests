package testcases

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/PlatONnetwork/PlatON-Go/common"
)

var (
	config = Config{}
)

const (
	DefaultConfigFilePath = "/config.json"
)

type Config struct {
	Account                common.Address `json:"account"` //address in genesis
	GeinisPrikey           string         `json:"prikey"`
	Url                    string         `json:"url"`
	Dir                    string         `json:"dir"`
	RestrictingConfigFile  string         `json:"restricting_config_file"`
	StakingConfigFile      string         `json:"staking_config_file"`
	PrivateKeyFile         string         `json:"private_key_file"`
	DefaultAccountAddrFile string         `json:"default_account_addr_file"`
}

func parseConfigJson(configPath string) {
	if configPath == "" {
		dir, _ := os.Getwd()
		configPath = dir + DefaultConfigFilePath
	}

	if !filepath.IsAbs(configPath) {
		configPath, _ = filepath.Abs(configPath)
	}

	bytes, err := ioutil.ReadFile(configPath)
	if err != nil {
		panic(fmt.Errorf("parse config file error,%s", err.Error()))
	}

	if err := json.Unmarshal(bytes, &config); err != nil {
		panic(fmt.Errorf("parse config to json error,%s", err.Error()))
	}
}

func loadData(configPath string) {
	parseConfigJson(configPath)
	parsePkFile()
	for _, value := range allAccounts {
		AccountPool.Put(value)
	}
}
