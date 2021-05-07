package repository

import (
	"errors"
	"fmt"
	"github.com/bulletmys/proceedd/server/kv/internal/models"
	"gopkg.in/yaml.v2"
	"log"
	"os"
	"sync"
	"sync/atomic"
	"time"
)

type KVMapRepository struct {
	Config       map[string]interface{}
	lastModified int64
	cfgMutex     *sync.RWMutex

	configFile *os.File

	// Feature
	Versions    []Version
	MinVersion  uint64
	LastVersion uint64
	verMutex    *sync.RWMutex
}

// fileUpdater overwrites the config file when data is updated
func (r *KVMapRepository) fileUpdater() {
	lastUpdateTS := int64(-1)

	for {
		if lastUpdateTS != atomic.LoadInt64(&r.lastModified) {
			// Clear file content before write
			log.Println(r.configFile.Truncate(0))
			log.Println(r.configFile.Seek(0, 0))

			encoder := yaml.NewEncoder(r.configFile)

			r.cfgMutex.RLock()
			if err := encoder.Encode(r.Config); err != nil {
				log.Printf("failed to upd config file: %v", err)
			}
			lastUpdateTS = r.lastModified
			r.cfgMutex.RUnlock()
		}

		time.Sleep(30 * time.Second) //TODO move to app conf and add graceful
	}
}

func NewKVMapRepository() (*KVMapRepository, error) {
	config := make(map[string]interface{})

	f, err := os.OpenFile("test.yaml", os.O_CREATE|os.O_RDWR, 0644)
	if err != nil {
		return nil, err
	}

	encoder := yaml.NewDecoder(f)
	if err := encoder.Decode(config); err != nil {
		return nil, fmt.Errorf("failed to write config file: %v", err)
	}

	repo := &KVMapRepository{
		Config:       config,
		lastModified: time.Now().UnixNano(),
		cfgMutex:     &sync.RWMutex{},
		configFile:   f,
	}

	go repo.fileUpdater()

	return repo, nil
}

type Version struct {
	V uint64
	models.KVDiff
}

func (r *KVMapRepository) GetFullConfig(timestamp int64) (models.KVFull, error) {
	r.cfgMutex.RLock()
	defer r.cfgMutex.RUnlock()

	if timestamp < r.lastModified {
		return models.KVFull{Config: r.Config, LastModified: r.lastModified}, nil
	}
	return models.KVFull{}, models.VersionNotModified
}

func (r *KVMapRepository) UpdConfig(data map[string][]string, jsonData models.KVDiff) error {
	r.cfgMutex.Lock()
	defer r.cfgMutex.Unlock()

	for k, v := range data {
		if len(v) == 0 {
			delete(r.Config, k)
		}
		r.Config[k] = v[0]
	}
	for k, v := range jsonData.Upd {
		r.Config[k] = v
	}
	for _, k := range jsonData.Del {
		delete(r.Config, k)
	}
	r.lastModified = time.Now().UnixNano()

	return nil
}

func (r *KVMapRepository) GetDiffConfig(version uint64) (models.KVDiff, error) {
	updates := make(map[string]interface{})
	delUniq := make(map[string]bool)

	r.verMutex.RLock()
	idx, err := r.checkVersion(version)
	switch err {
	case models.VersionNotModified:
		fallthrough
	case models.OldVersionRequested:
		r.verMutex.RUnlock()
		return models.KVDiff{}, err
	case nil:
	default:
		r.verMutex.RUnlock()
		return models.KVDiff{}, fmt.Errorf("failed to get diff config: %w", err)
	}
	for _, val := range r.Versions[idx:] {
		for k, v := range val.Upd {
			updates[k] = v
		}
		for _, v := range val.Del {
			delUniq[v] = true
		}
	}
	r.verMutex.RUnlock()

	deletes := make([]string, len(delUniq))
	i := 0

	for k := range delUniq {
		deletes[i] = k
		i++
	}

	return models.KVDiff{Upd: updates, Del: deletes}, nil
}

func (r *KVMapRepository) checkVersion(version uint64) (uint64, error) {
	if version == r.LastVersion {
		return 0, models.VersionNotModified
	}

	if version > r.LastVersion {
		return 0, errors.New("bad version revision")
	}

	if version+1 < r.MinVersion {
		return 0, models.OldVersionRequested
	}

	return version + 1 - r.MinVersion, nil
}

func (r KVMapRepository) test() {

}
