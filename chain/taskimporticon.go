/*
 * Copyright 2021 ICON Foundation
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package chain

import (
	"encoding/json"
	"fmt"
	"io"
	"mime"
	"net/http"
	"os"
	"path"
	"strconv"
	"strings"

	"github.com/labstack/echo/v4"

	"github.com/icon-project/goloop/block"
	"github.com/icon-project/goloop/common"
	"github.com/icon-project/goloop/common/db"
	"github.com/icon-project/goloop/common/errors"
	"github.com/icon-project/goloop/consensus"
	"github.com/icon-project/goloop/icon/blockv0/lcstore"
	"github.com/icon-project/goloop/icon/lcimporter"
	"github.com/icon-project/goloop/network"
)

const (
	ImportICONTask = "import_icon"
	ImportICONName = "ImportICON"
)

var lcStoreDefaultCacheConfig = lcstore.CacheConfig{
	MaxWorkers: 8,
	MaxBlocks:  32,
}

type importICONParams struct {
	StoreURI    string               `json:"store_uri"`
	ConfigURL   string               `json:"config_url"`
	MaxRPS      int                  `json:"max_rps"`
	CacheConfig *lcstore.CacheConfig `json:"cache_config,omitempty"`
}

type importICONConfig struct {
	Validators  []*common.Address   `json:"validators"`
}

type taskImportICON struct {
	chain  *singleChain
	params *importICONParams

	result resultStore

	dbase db.Database
	sm    *lcimporter.ServiceManager
}

func (t *taskImportICON) String() string {
	return ImportICONName
}

func (t *taskImportICON) DetailOf(s State) string {
	switch s {
	case Started:
		return fmt.Sprintf("%s %s", ImportICONTask, t.sm.GetStatus())
	default:
		return ImportICONTask +" "+s.String()
	}
}

func (t *taskImportICON) Start() error {
	if err := t._import(); err != nil {
		t.result.SetValue(err)
		return err
	}
	return nil
}

func (t *taskImportICON) _prepareDatabase() error {
	cfg := t.chain.cfg
	chainDir := cfg.AbsBaseDir()
	tmpDBDir := path.Join(chainDir, DefaultTmpDBDir)
	dbName := strconv.FormatInt(int64(cfg.NID), 16)
	if dbase, err := db.Open(tmpDBDir, cfg.DBType, dbName); err != nil {
		return err
	} else {
		t.dbase = dbase
		return nil
	}
}

func (t *taskImportICON) _releaseDatabase() {
	if t.dbase != nil {
		t.dbase.Close()
		t.dbase = nil
	}
}


func (t *taskImportICON) _loadConfig() (*importICONConfig, error) {
	_readConfig := func(rc io.ReadCloser) (*importICONConfig, error) {
		defer rc.Close()
		cfg := new(importICONConfig)
		jd := json.NewDecoder(rc)
		jd.DisallowUnknownFields()
		if err := jd.Decode(cfg); err != nil {
			return nil, err
		}
		return cfg, nil
	}

	if strings.HasPrefix(t.params.ConfigURL, "http") {
		client := new(http.Client)
		if resp, err := client.Get(t.params.ConfigURL); err != nil {
			return nil, err
		} else {
			if resp.StatusCode != http.StatusOK {
				return nil, errors.UnknownError.Errorf("ConfigFail(status=%s)", resp.Status)
			}
			if ct, _, err := mime.ParseMediaType(resp.Header.Get(echo.HeaderContentType)); err != nil {
				return nil, err
			} else if ct == echo.MIMEApplicationJSON {
				return _readConfig(resp.Body)
			} else {
				return nil, errors.UnknownError.Errorf("InvalidContentType")
			}
		}
	} else {
		if fd, err := os.Open(t.params.ConfigURL); err != nil {
			return nil, err
		} else {
			return _readConfig(fd)
		}
	}
}

func (t *taskImportICON) _import() (ret error) {
	c := t.chain

	if err := t._prepareDatabase(); err != nil {
		return err
	}
	defer func() {
		if ret != nil {
			t._releaseDatabase()
			t.chain.releaseManagers()
		}
	}()

	// load configurations
	tc, err := t._loadConfig()
	if err != nil {
		return err
	}
	config := &lcimporter.Config{
		Validators:  tc.Validators,
		StoreURI:    t.params.StoreURI,
		MaxRPS:      t.params.MaxRPS,
	}
	if t.params.CacheConfig != nil {
		config.CacheConfig = *t.params.CacheConfig
	} else {
		config.CacheConfig = lcStoreDefaultCacheConfig
	}
	config.BaseDir = c.cfg.AbsBaseDir()
	config.Platform = c.plt
	config.ProxyMgr = c.pm

	// initialize network manager
	pr := network.PeerRoleFlag(c.cfg.Role)
	c.nm = network.NewManager(c, c.nt, c.cfg.SeedAddr, pr.ToRoles()...)

	// initialize service manager
	if sm, err := lcimporter.NewServiceManager(c, t.dbase, config, t); err != nil {
		return err
	} else {
		t.sm = sm
		c.sm = sm
	}

	// initialize block manager
	if bm, err := block.NewManager(c, nil, nil); err != nil {
		return err
	} else {
		c.bm = bm
	}

	// initialize consensus
	WALDir := path.Join(c.cfg.AbsBaseDir(), DefaultWALDir)
	c.cs = consensus.NewConsensus(c, WALDir, nil, nil)

	// start migration
	if err := c.nm.Start(); err != nil {
		return err
	}
	c.sm.Start()
	if err := c.cs.Start(); err != nil {
		return err
	}
	return nil
}

func (t *taskImportICON) Stop() {
	t.result.SetValue(errors.ErrInterrupted)
}

func (t *taskImportICON) prepareConsensus() error {
	chainDir := t.chain.cfg.AbsBaseDir()
	t.chain.releaseDatabase()
	defer t.chain.ensureDatabase()

	walDir := path.Join(chainDir, DefaultWALDir)
	tmpDir := path.Join(chainDir, DefaultTmpDBDir)
	dbDir := path.Join(chainDir, DefaultDBDir)

	if err := os.RemoveAll(walDir); err != nil {
		return errors.Wrapf(err, "FailToRemoveWAL(%s)", walDir)
	}
	if err := os.RemoveAll(dbDir); err != nil {
		return errors.Wrapf(err, "FailToRemoveDB(%s)", dbDir)
	}
	if err := os.Rename(tmpDir, dbDir); err != nil {
		return errors.Wrapf(err, "FailToRenameDB(%s->%s)", tmpDir, dbDir)
	}
	return nil
}

func (t *taskImportICON) Wait() error {
	result := t.result.Wait()
	if t.sm.Finished() {
		result = nil
	}
	t.chain.releaseManagers()
	t._releaseDatabase()
	if result == nil {
		return t.prepareConsensus()
	}
	return result
}

func (t *taskImportICON) OnResult(err error) {
	t.result.SetValue(err)
}

func taskImportIconFactory(c *singleChain, params json.RawMessage) (chainTask, error) {
	p := new(importICONParams)
	if err := json.Unmarshal(params, p); err != nil {
		return nil, err
	}
	return &taskImportICON{
		chain:  c,
		params: p,
	}, nil
}

func init() {
	registerTaskFactory(ImportICONTask, taskImportIconFactory)
}
