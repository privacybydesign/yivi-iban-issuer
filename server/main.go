package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
)

type Config struct {
	ServerConfig ServerConfig `json:"server_config"`

	JwtPrivateKeyPath string `json:"jwt_private_key_path"`
	IssuerId          string `json:"issuer_id"`
	FullCredential    string `json:"full_credential"`

	CmIbanConfig CmIbanConfig `json:"cm_iban_config,omitempty"`
	StorageType  string       `json:"storage_type"`
}

func main() {
	configPath := flag.String("config", "", "Path for the config.json to use")
	flag.Parse()

	if *configPath == "" {
		fmt.Println("please provide a config path using the --config flag")
	}

	fmt.Println("using config: %v", *configPath)

	config, err := readConfigFile(*configPath)
	if err != nil {
		fmt.Println("failed to read config file: %v", err)
	}

	fmt.Println("hosting on: %v:%v", config.ServerConfig.Host, config.ServerConfig.Port)

	jwtCreator, err := NewIrmaJwtCreator(
		config.JwtPrivateKeyPath,
		config.IssuerId,
		config.FullCredential,
	)
	if err != nil {
		fmt.Println("failed to instantiate jwt creator: %v", err)
	}

	ibanChecker, err := createIbanBackend(&config)
	if err != nil {
		fmt.Println("failed to instantiate sms backend: %v", err)
	}

	serverState := ServerState{
		ibanChecker:      ibanChecker,
		jwtCreator:       jwtCreator,
		transactionCache: make(map[string]string),
	}

	server, err := NewServer(&serverState, config.ServerConfig)
	if err != nil {
		fmt.Println("failed to create server: %v", err)
	}

	err = server.ListenAndServe()
	if err != nil {
		fmt.Println("failed to listen and serve: %v", err)
	}
}

func createIbanBackend(config *Config) (IbanChecker, error) {
	return NewCmIbanChecker(config.CmIbanConfig)
}

func readConfigFile(path string) (Config, error) {
	configBytes, err := os.ReadFile(path)

	if err != nil {
		return Config{}, err
	}

	var config Config
	err = json.Unmarshal(configBytes, &config)

	if err != nil {
		return Config{}, err
	}

	return config, nil
}
