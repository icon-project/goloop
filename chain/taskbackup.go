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

package chain

import (
	"archive/zip"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path"
	"sort"
	"sync/atomic"

	"github.com/icon-project/goloop/common"
	"github.com/icon-project/goloop/common/codec"
	"github.com/icon-project/goloop/common/errors"
)

const TemporalBackupFile = ".backup"

type BackupInfo struct {
	NID     common.HexInt32 `json:"nid"`
	CID     common.HexInt32 `json:"cid"`
	Channel string          `json:"channel"`
	Height  int64           `json:"height"`
	Codec   string          `json:"codec"`
}

var backupStates = map[State]string{
	Starting: "backup starting",
	Stopping: "backup stopping",
	Failed:   "backup failed",
	Finished: "backup done",
}

type taskBackup struct {
	chain   *singleChain
	file    string
	extra   []string
	fd      io.WriteCloser
	zw      *zip.Writer
	current int32
	total   int32
	stop    int32
	result  resultStore
}

func (t *taskBackup) String() string {
	return fmt.Sprintf("Backup(file=%s)", path.Base(t.file))
}

func (t *taskBackup) DetailOf(s State) string {
	switch s {
	case Started:
		total := atomic.LoadInt32(&t.total)
		if total > 0 {
			current := atomic.LoadInt32(&t.current)
			return fmt.Sprintf("backup %d/%d", current, total)
		} else if total == 0 {
			return "backup started"
		} else {
			return "backup manual"
		}
	default:
		if ss, ok := backupStates[s]; ok {
			return ss
		} else {
			return s.String()
		}
	}
}

func (t *taskBackup) Start() (ret error) {
	// On manual backup, it just releases the database.
	if t.file == "" {
		atomic.StoreInt32(&t.total, -1)
		t.chain.releaseDatabase()
		return nil
	}
	tmp, err := os.CreateTemp(path.Dir(t.file), TemporalBackupFile)
	if err != nil {
		return errors.Wrap(err, "Fail to make temporal file")
	}
	defer func() {
		if ret != nil {
			tmp.Close()
			os.Remove(tmp.Name())
		}
	}()
	if err := tmp.Chmod(0644); err != nil {
		return err
	}

	t.fd = tmp
	t.zw = zip.NewWriter(tmp)

	if err := writeBackupInfo(t.zw, &BackupInfo{
		NID:     common.HexInt32{Value: int32(t.chain.NID())},
		CID:     common.HexInt32{Value: int32(t.chain.CID())},
		Channel: t.chain.Channel(),
		Height:  t.chain.lastBlockHeight(),
		Codec:   codec.BC.Name(),
	}); err != nil {
		return err
	}

	t.chain.releaseDatabase()

	go func() {
		err := t._backup()
		if err == nil {
			err = os.Rename(tmp.Name(), t.file)
		}
		if err != nil {
			os.Remove(tmp.Name())
		}
		t.result.SetValue(err)
	}()
	return nil
}

func zipWrite(writer *zip.Writer, p, n string, on func(int64) error) error {
	p2 := path.Join(p, n)
	st, err := os.Stat(p2)
	if errors.Is(err, fs.ErrNotExist) {
		return nil
	} else if err != nil {
		return errors.Wrap(err, "writeToZip: FAIL on os.State")
	}
	if st.Mode().IsRegular() {
		fd, err := os.Open(p2)
		defer fd.Close()
		if err != nil {
			return errors.Wrapf(err, "writeToZip: fail to open %s", p2)
		}

		fh, err := zip.FileInfoHeader(st)
		if err != nil {
			return errors.Wrapf(err, "writeToZip: fail to make header for %s", p2)
		}
		fh.Name = n
		fh.Method = zip.Deflate
		zf, err := writer.CreateHeader(fh)
		if err != nil {
			return err
		}
		if err != nil {
			return errors.Wrapf(err, "writeToZip: fail to create entry %s", n)
		}
		if _, err := io.Copy(zf, fd); err != nil {
			return errors.Wrap(err, "writeToZip: fail to copy")
		}
		if err := on(st.Size()); err != nil {
			return err
		}
		return nil
	} else if !st.IsDir() {
		return nil
	}

	fis, err := os.ReadDir(p2)
	if err != nil {
		return errors.Wrap(err, "writeToZip: FAIL on ReadDir")
	}
	// make it generate consistent compressed zip file.
	sort.SliceStable(fis, func(i, j int) bool {
		return fis[i].Name() < fis[j].Name()
	})
	for _, fi := range fis {
		if err := zipWrite(writer, p, path.Join(n, fi.Name()), on); err != nil {
			return err
		}
	}
	return nil
}

func countFiles(p string) (int, error) {
	st, err := os.Stat(p)
	if errors.Is(err, fs.ErrNotExist) {
		return 0, nil
	} else if err != nil {
		return 0, err
	}
	if !st.IsDir() {
		return 1, nil
	}
	fis, err := os.ReadDir(p)
	if err != nil {
		return 0, err
	}
	cnt := 0
	for _, fi := range fis {
		if fi.IsDir() {
			if c, err := countFiles(path.Join(p, fi.Name())); err != nil {
				return 0, err
			} else {
				cnt += c
			}
		} else if fi.Type().IsRegular() {
			cnt += 1
		}
	}
	return cnt, nil
}

func (t *taskBackup) _countFiles(chainDir string, names []string) (int, error) {
	count := 0
	for _, name := range names {
		if cnt, err := countFiles(path.Join(chainDir, name)); err != nil {
			return 0, err
		} else {
			count += cnt
		}
	}
	return count, nil
}

func (t *taskBackup) _isInterrupted() bool {
	return atomic.LoadInt32(&t.stop) != 0
}

func (t *taskBackup) OnWrite(int64) error {
	if t._isInterrupted() {
		return errors.ErrInterrupted
	}
	atomic.AddInt32(&t.current, 1)
	return nil
}

func (t *taskBackup) _backup() error {
	defer t.chain.ensureDatabase()
	defer t.fd.Close()
	defer t.zw.Close()

	names := append([]string{
		DefaultWALDir, DefaultDBDir, DefaultContractDir,
	}, t.extra...)

	chainDir := t.chain.cfg.AbsBaseDir()
	if cnt, err := t._countFiles(chainDir, names); err != nil {
		return err
	} else {
		t.total = int32(cnt)
	}

	for _, name := range names {
		if err := zipWrite(t.zw, chainDir, name, t.OnWrite); err != nil {
			return err
		}
	}

	return nil
}

func (t *taskBackup) Stop() {
	if t.file == "" {
		// if it's manual backup we need to recover database
		// and awake waiter.
		t.chain.ensureDatabase()
		t.result.SetValue(nil)
	}
	atomic.StoreInt32(&t.stop, 1)
}

func (t *taskBackup) Wait() error {
	return t.result.Wait()
}

func newTaskBackup(chain *singleChain, file string, extra []string) chainTask {
	return &taskBackup{
		chain: chain,
		file:  file,
		extra: extra,
	}
}

func writeBackupInfo(zw *zip.Writer, info *BackupInfo) error {
	bs, err := json.Marshal(info)
	if err != nil {
		return err
	}
	return zw.SetComment(string(bs))
}

func GetBackupInfoOf(f string) (*BackupInfo, error) {
	fd, err := os.Open(f)
	if err != nil {
		return nil, err
	}
	defer fd.Close()

	st, err := fd.Stat()
	if err != nil {
		return nil, err
	}
	reader, err := zip.NewReader(fd, st.Size())
	if err != nil {
		return nil, err
	}
	return ReadBackupInfo(reader)
}

func ReadBackupInfo(zr *zip.Reader) (*BackupInfo, error) {
	info := new(BackupInfo)
	if err := json.Unmarshal([]byte(zr.Comment), info); err != nil {
		return nil, err
	}
	return info, nil
}
