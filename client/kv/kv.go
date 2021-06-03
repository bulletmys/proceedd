package kv

import (
	"encoding/json"
	"fmt"
	"github.com/golobby/config/v2"
	"gopkg.in/yaml.v2"
	"io"
	"log"
	"net/http"
	"os"
	"time"
)

type KVUpdater struct {
	KVStorageAddr  string
	GetDiffConfUrl string
	GetFullConfUrl string
	RefreshTimeout time.Duration
	File           *os.File
	Config         map[string]interface{}
	LastModified   int64
}

type ConfigDiff struct {
	Upd map[string]interface{}
	Del []string
}

type ConfigFull struct {
	Config       map[string]interface{}
	LastModified int64
}

func initKVUpdater(c *config.Config) (*KVUpdater, error) {
	addr, err := c.GetString("kv.addr")
	if err != nil {
		return nil, err
	}

	diffConfUrl, err := c.GetString("kv.conf_diff_url")
	if err != nil {
		return nil, err
	}

	fullConfUrl, err := c.GetString("kv.conf_full_url")
	if err != nil {
		return nil, err
	}

	timeout, err := c.GetString("kv.refresh_timeout")
	if err != nil {
		return nil, err
	}
	refreshTimeout, err := time.ParseDuration(timeout)
	if err != nil {
		return nil, fmt.Errorf("failed to parse refresh timeout: %w", err)
	}

	kvConfigPath, err := c.GetString("kv.config_path")
	if err != nil {
		return nil, err
	}
	// If the file doesn't exist, create it
	f, err := os.OpenFile(kvConfigPath, os.O_CREATE|os.O_RDWR, 0644)
	if err != nil {
		log.Fatal(err)
	}

	return &KVUpdater{
		KVStorageAddr:  addr,
		RefreshTimeout: refreshTimeout,
		File:           f,
		GetDiffConfUrl: diffConfUrl,
		GetFullConfUrl: fullConfUrl,
	}, nil
}

func (c *KVUpdater) writeConfFile() {
	// Clear file content before write
	c.File.Truncate(0)
	c.File.Seek(0, 0)

	encoder := yaml.NewEncoder(c.File)
	if err := encoder.Encode(c.Config); err != nil {
		log.Printf("failed to write config file: %v", err)
	}
}

func (c *KVUpdater) updFullConfig(data io.ReadCloser) error {
	configFull := ConfigFull{Config: make(map[string]interface{})}

	decoder := json.NewDecoder(data)
	if err := decoder.Decode(&configFull); err != nil {
		return err
	}
	fmt.Println(configFull.Config)
	c.Config = configFull.Config
	c.LastModified = configFull.LastModified
	c.writeConfFile()

	return nil
}

func (c *KVUpdater) updDiffConfig(data io.ReadCloser) error {
	var configDiff ConfigDiff

	decoder := json.NewDecoder(data)
	if err := decoder.Decode(&configDiff); err != nil {
		return err
	}

	for _, key := range configDiff.Del {
		delete(c.Config, key)
	}
	for k, v := range configDiff.Upd {
		c.Config[k] = v
	}
	if len(configDiff.Upd) > 0 || len(configDiff.Del) > 0 {
		c.writeConfFile()
	}

	return nil
}

func (c *KVUpdater) getConfig(client *http.Client, apiUrl string, configParser func(data io.ReadCloser) error) error {
	resp, err := client.Get(apiUrl)
	if err != nil {
		return fmt.Errorf("failed to get kv config: %v", err)
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusNotModified:
		log.Println("Not Modified")
	case http.StatusOK:
		log.Printf("Got new config, resp size: %v\n", resp.ContentLength)
		err = configParser(resp.Body)
	default:
		err = fmt.Errorf("unknown status from kv storage: %v", resp.StatusCode)
	}

	return err
}

func (c *KVUpdater) updater(client *http.Client, cfgDiffFeature bool) {
	apiUrl := fmt.Sprintf("%v%v?ts=%v", c.KVStorageAddr, c.GetFullConfUrl, c.LastModified)
	configParser := c.updFullConfig
	if cfgDiffFeature {
		apiUrl = fmt.Sprintf("%v%v", c.KVStorageAddr, c.GetDiffConfUrl)
		configParser = c.updDiffConfig
	}

	for {
		t1 := time.Now()
		if err := c.getConfig(client, apiUrl, configParser); err != nil {
			log.Printf("failed to update config: %v", err)
		}
		t2 := time.Now()
		log.Println(t2.Sub(t1))

		apiUrl = fmt.Sprintf("%v%v?ts=%v", c.KVStorageAddr, c.GetFullConfUrl, c.LastModified)
		if cfgDiffFeature {
			apiUrl = fmt.Sprintf("%v%v", c.KVStorageAddr, c.GetDiffConfUrl)
		}

		time.Sleep(c.RefreshTimeout)
	}
}

func Start(c *config.Config) error {
	conf, err := initKVUpdater(c)
	if err != nil {
		return fmt.Errorf("failed to read config: %w", err)
	}
	defer conf.File.Close()

	tr := &http.Transport{
		MaxIdleConns:    10,
		IdleConnTimeout: 30 * time.Second,
	}
	client := &http.Client{Transport: tr}

	//if err := conf.getConfig(client); err != nil {
	//	return fmt.Errorf("failed to load full config: %w", err)
	//}

	conf.updater(client, false)

	return nil
}
