package config

import (
	"time"

	"github.com/kelseyhightower/envconfig"
)

type EnvVars struct {
	Blockchain
}

type Blockchain struct {
	Difficulty         int           `envconfig:"CHAIN_MINING_DIFFICULTY" required:"true"`
	MiningSender       string        `envconfig:"CHAIN_MINING_SENDER" default:"DENIZ"`
	DefaultRewardToken string        `envconfig:"CHAIN_DEFAULT_REWARD_TOKEN" default:"DNZ"`
	MiningReward       float64       `envconfig:"CHAIN_MINING_REWARD" required:"true"`
	MiningTimerSeconds time.Duration `envconfig:"CHAIN_MINING_TIMER_SECONDS" required:"true"`
	PortRangeStart     uint16        `envconfig:"CHAIN_BLOCKCHAIN_PORT_RANGE_START" required:"true"`
	PortRangeEnd       uint16        `envconfig:"CHAIN_BLOCKCHAIN_PORT_RANGE_END" required:"true"`
	IpRangeStart       uint8         `envconfig:"CHAIN_NODE_IP_RANGE_START" required:"true"`
	IpRangeEnd         uint8         `envconfig:"CHAIN_NODE_IP_RANGE_END" required:"true"`
	NodeSyncTimeSec    time.Duration `envconfig:"CHAIN_BLOCKCHAIN_NODE_SYNC_TIME_SEC" required:"true"`
	BlockChainPort     uint16        `envconfig:"CHAIN_PORT" required:"true"`
	DbSavePath         string        `envconfig:"CHAIN_DB_SAVE_PATH" required:"true"`
}

func GetConfig() (*EnvVars, error) {
	conf := EnvVars{}
	err := envconfig.Process("CHAIN_", &conf)
	if err != nil {
		panic(err)
	}
	return &conf, nil
}
