/*
 * Copyright 2020 ICON Foundation
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

package node

import (
	"archive/zip"
	"fmt"
	"io"
	"os"
	"path"
	"sync"

	"github.com/icon-project/goloop/chain"
	"github.com/icon-project/goloop/common/codec"
	"github.com/icon-project/goloop/common/errors"
)

const (
	RestoreDirectoryPrefix = ".restore"
)

type RestoreState int

const (
	RestoreNone RestoreState = iota
	RestoreStarted
	RestoreFailed
	RestoreSuccess
	RestoreStopping
)

func (s RestoreState) String() string {
	switch s {
	case RestoreNone:
		return "none"
	case RestoreStarted:
		return "started"
	case RestoreFailed:
		return "failed"
	case RestoreSuccess:
		return "success"
	case RestoreStopping:
		return "stopping"
	default:
		return "unknown"
	}
}

type RestoreStatus struct {
	File      string
	Overwrite bool
	State     string
	Error     error
}

type RestoreManager struct {
	lock      sync.Mutex
	file      string
	channel   string
	overwrite bool

	state   RestoreState
	current int
	total   int
	lastErr error
}

func (m *RestoreManager) Start(node *Node, file string, baseDir string, overwrite bool) (ret error) {
	m.lock.Lock()
	defer m.lock.Unlock()

	switch m.state {
	case RestoreFailed, RestoreSuccess:
		m._setStateInLock(RestoreNone, nil)
	case RestoreNone:
	case RestoreStarted, RestoreStopping:
		return errors.InvalidStateError.Errorf(
			"StillRestoring(%s)", path.Base(m.file))
	}

	tmpDir, err := os.MkdirTemp(baseDir, RestoreDirectoryPrefix)
	if err != nil {
		return err
	}
	defer func() {
		if ret != nil {
			os.RemoveAll(tmpDir)
		}
	}()

	zr, err := zip.OpenReader(file)
	if err != nil {
		return errors.IllegalArgumentError.Wrapf(err,
			"ZipOpenFailure(backup=%s)", file)
	}
	defer func() {
		if ret != nil {
			zr.Close()
		}
	}()

	info, err := chain.ReadBackupInfo(&zr.Reader)
	if err != nil {
		return errors.IllegalArgumentError.Wrap(err,
			"InvalidBackupInfo")
	}

	if info.Codec != codec.BC.Name() {
		return errors.IllegalArgumentError.Errorf(
			"IncompatibleCodec(backup=%s,system=%s)",
			info.Codec, codec.BC.Name())
	}

	if err := node.CanAdd(int(info.CID.Value), int(info.NID.Value), info.Channel, overwrite); err != nil {
		return err
	}

	go func() {
		if err := m._restore(node, zr, tmpDir, overwrite); err != nil {
			node.logger.Debugf("Restore failed err=%+v", err)
			if errors.InterruptedError.Equals(err) {
				m._setState(RestoreNone, nil)
			} else {
				m._setState(RestoreFailed, err)
			}
		} else {
			m._setState(RestoreSuccess, nil)
		}
	}()

	m.file = file
	m.overwrite = overwrite
	m.state = RestoreStarted
	m.current = 0
	m.total = len(zr.File)
	return nil
}

func (m *RestoreManager) _onRestored(idx int) error {
	m.lock.Lock()
	defer m.lock.Unlock()

	if m.state != RestoreStarted {
		return errors.ErrInterrupted
	}
	m.current = idx + 1
	return nil
}

func (m *RestoreManager) GetStatus() *RestoreStatus {
	m.lock.Lock()
	defer m.lock.Unlock()

	switch m.state {
	case RestoreNone:
		return nil
	case RestoreStarted:
		return &RestoreStatus{
			File:      m.file,
			Overwrite: m.overwrite,
			State:     fmt.Sprintf("started %d/%d", m.current, m.total),
		}
	default:
		return &RestoreStatus{
			File:      m.file,
			Overwrite: m.overwrite,
			State:     m.state.String(),
			Error:     m.lastErr,
		}
	}
}

func (m *RestoreManager) _setState(s RestoreState, e error) {
	m.lock.Lock()
	defer m.lock.Unlock()

	m._setStateInLock(s, e)
}

func (m *RestoreManager) _setStateInLock(s RestoreState, e error) {
	m.state = s
	m.lastErr = e
	if s == RestoreNone {
		m.file = ""
		m.total = 0
		m.current = 0
	}
}

func zipExtract(file *zip.File, tmpDir string) (ret error) {
	rc, err := file.Open()
	if err != nil {
		return err
	}
	defer rc.Close()

	target := path.Join(tmpDir, file.Name)
	mode := file.Mode()
	if mode.IsDir() {
		return os.MkdirAll(target, mode.Perm())
	}

	if !file.Mode().IsRegular() {
		return nil
	}

	if err := os.MkdirAll(path.Dir(target), 0755); err != nil {
		return err
	}

	fd, err := os.OpenFile(target,
		os.O_CREATE|os.O_EXCL|os.O_RDWR|os.O_TRUNC, mode.Perm())
	if err != nil {
		return err
	}
	defer fd.Close()

	_, err = io.Copy(fd, rc)

	return err
}

func (m *RestoreManager) _restore(node *Node, zr *zip.ReadCloser, tmpDir string, overwrite bool) (ret error) {
	defer func() {
		if ret != nil {
			os.RemoveAll(tmpDir)
		}
	}()
	defer zr.Close()

	for idx, file := range zr.File {
		if err := zipExtract(file, tmpDir); err != nil {
			return err
		}
		if err := m._onRestored(idx); err != nil {
			return err
		}
	}

	return node.restoreChain(tmpDir, overwrite)
}

func (m *RestoreManager) Stop() error {
	m.lock.Lock()
	defer m.lock.Unlock()

	switch m.state {
	case RestoreFailed, RestoreSuccess:
		m._setStateInLock(RestoreNone, nil)
		return nil
	case RestoreStarted:
		m._setStateInLock(RestoreStopping, nil)
		return nil
	default:
		return errors.InvalidStateError.Errorf("UnableToStop(state=%s)", m.state.String())
	}
}
