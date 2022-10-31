package consensus

import (
	"bufio"
	"encoding/binary"
	"fmt"
	"hash/crc32"
	"io"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/icon-project/goloop/common"
	"github.com/icon-project/goloop/common/errors"
	"github.com/icon-project/goloop/common/log"
)

const (
	configWALBufSize              = 1024 * 4
	configWALFileLimit            = 1024 * 1024 * 2
	configWALTotalLimit           = configWALFileLimit * 4
	configWALHousekeepingInterval = time.Second * 1
	configWALSyncInterval         = time.Second * 1
)

const (
	maxUint64                    = ^uint64(0)
	walPermission    os.FileMode = 0600
	walDirPermission os.FileMode = 0700
	headerLen                    = 8
)

var crc32c = crc32.MakeTable(crc32.Castagnoli)

type WALWriter interface {
	WriteBytes([]byte) (int, error)
	Sync() error
	Close() error
}

type WALReader interface {
	ReadBytes() ([]byte, error)
	// Close closes reader. Multiple call of Close is safe.
	Close() error
	// CloseAndRepair closes and repairs UnexpectedEOF or CorruptedWAL by
	// truncating.
	CloseAndRepair() error
}

type WALConfig struct {
	FileLimit            int64
	TotalLimit           int64
	HousekeepingInterval time.Duration
	SyncInterval         time.Duration
}

type file struct {
	*os.File
}

type walWriter struct {
	mutex            common.Mutex
	id               string
	cfg              WALConfig
	buf              *bufio.Writer
	tail             file
	tailIdx          uint64
	eldestUnsyncData *time.Time

	ticker        *time.Ticker
	tickerStop    chan struct{}
	tickerStopped chan struct{}
}

type walInfo struct {
	headIdx   uint64
	tailIdx   uint64
	totalSize int64
	tailSize  int64
	fileSizes []int64
}

func fileFor(id string, idx uint64) string {
	return fmt.Sprintf("%s_%d", id, idx)
}

func readWALInfo(id string) (*walInfo, error) {
	groupDir := filepath.Dir(id)
	var minIndex, maxIndex uint64 = maxUint64, 0
	var totalSize, tailSize int64 = 0, 0

	dir, err := os.Open(groupDir)
	if err != nil {
		return nil, errors.Wrapf(os.ErrNotExist, "cannot open dir=%v wal=%v err=%v", groupDir, id, err)
	}
	defer func() {
		log.Must(dir.Close())
	}()

	entries, err := dir.Readdir(0)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	prefix := filepath.Base(id) + "_"
	fileSizeFor := make(map[uint64]int64)
	for _, entry := range entries {
		if !strings.HasPrefix(entry.Name(), prefix) {
			continue
		}
		fileSize := entry.Size()
		totalSize += fileSize
		idxStr := entry.Name()[len(prefix):]
		idx, err := strconv.ParseUint(idxStr, 10, 64)
		if err != nil {
			continue
		}
		fileSizeFor[idx] = fileSize
		//log.Printf("file=%v size=%v\n", entry.Name(), entry.Size())
		if maxIndex == 0 || maxIndex < idx {
			maxIndex = idx
			tailSize = fileSize
		}
		if idx < minIndex {
			minIndex = idx
		}
	}

	nFiles := int64(maxIndex) - int64(minIndex) + 1
	if nFiles < 0 {
		nFiles = 0
	}
	fileSizes := make([]int64, nFiles)
	for i := uint64(0); i < uint64(nFiles); i++ {
		fileSizes[i] = fileSizeFor[i+minIndex]
	}

	return &walInfo{
		minIndex,
		maxIndex,
		totalSize,
		tailSize,
		fileSizes,
	}, nil
}

// OpenWALForWrite opens WALWriter. id is in the form /wal/dir/prefix
func OpenWALForWrite(id string, cfg *WALConfig) (WALWriter, error) {
	w := &walWriter{
		id:  id,
		cfg: *cfg,
	}
	if w.cfg.FileLimit <= 0 {
		w.cfg.FileLimit = configWALFileLimit
	}
	if w.cfg.TotalLimit <= 0 {
		w.cfg.TotalLimit = configWALTotalLimit
	}
	if w.cfg.HousekeepingInterval <= time.Duration(0) {
		w.cfg.HousekeepingInterval = configWALHousekeepingInterval
	}
	if w.cfg.SyncInterval <= time.Duration(0) {
		w.cfg.SyncInterval = configWALSyncInterval
	}

	w.buf = bufio.NewWriterSize(&w.tail, configWALBufSize)
	wi, err := readWALInfo(w.id)
	if IsNotExist(err) {
		if err = os.MkdirAll(filepath.Dir(id), walDirPermission); err != nil {
			return nil, errors.WithStack(err)
		}
		wi, err = readWALInfo(w.id)
	}
	if err != nil {
		return nil, err
	}
	w.tail.File, err = os.OpenFile(fileFor(w.id, wi.tailIdx), os.O_CREATE|os.O_WRONLY|os.O_APPEND, walPermission)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	w.tailIdx = wi.tailIdx

	w.mutex.Lock()
	defer w.mutex.Unlock()

	w.startHousekeeping()

	return w, nil
}

func (w *walWriter) WriteBytes(payload []byte) (int, error) {
	w.mutex.Lock()
	defer w.mutex.Unlock()

	crc := crc32.Checksum(payload, crc32c)
	payloadLen := len(payload)
	frameLen := headerLen + payloadLen

	frame := make([]byte, frameLen)
	binary.BigEndian.PutUint32(frame[0:4], crc)
	binary.BigEndian.PutUint32(frame[4:headerLen], uint32(payloadLen))
	copy(frame[headerLen:], payload)
	//log.Printf("wal write crc=%x payloadLen:%v payload:%x\n", crc, payloadLen, payload)
	n, err := w.buf.Write(frame)
	if err == nil && w.eldestUnsyncData == nil {
		now := time.Now()
		w.eldestUnsyncData = &now
	}
	return n, err
}

func WALWriteObject(w WALWriter, v interface{}) error {
	bs, err := msgCodec.MarshalToBytes(v)
	if err != nil {
		return err
	}
	_, err = w.WriteBytes(bs)
	return err
}

func (w *walWriter) Sync() error {
	w.mutex.Lock()
	defer w.mutex.Unlock()

	return w.sync()
}

func (w *walWriter) sync() error {
	if err := w.buf.Flush(); err != nil {
		return errors.WithStack(err)
	}
	if err := w.tail.Sync(); err != nil {
		return err
	}
	w.eldestUnsyncData = nil
	return nil
}

func (w *walWriter) startHousekeeping() {
	w.ticker = time.NewTicker(w.cfg.HousekeepingInterval)
	w.tickerStop = make(chan struct{})
	w.tickerStopped = make(chan struct{})
	go w.housekeep(w.ticker.C, w.tickerStop, w.tickerStopped)
}

func (w *walWriter) stopHousekeeping() {
	w.mutex.Lock()
	defer w.mutex.Unlock()

	w.ticker.Stop()
	w.mutex.CallAfterUnlock(func() {
		w.tickerStop <- struct{}{}
		<-w.tickerStopped
	})
}

func (w *walWriter) Close() error {
	w.stopHousekeeping()

	w.mutex.Lock()
	defer w.mutex.Unlock()

	err := w.sync()
	if err != nil {
		return err
	}
	err = w.tail.File.Close()
	if err != nil {
		return errors.WithStack(err)
	}
	return nil
}

func (w *walWriter) housekeep(t <-chan time.Time, s <-chan struct{}, d chan<- struct{}) {
	for {
		select {
		case <-t:
			w.doHousekeeping()
		case <-s:
			d <- struct{}{}
			return
		}
	}
}

func (w *walWriter) Shift() error {
	w.mutex.Lock()
	defer w.mutex.Unlock()

	return w.shift()
}

func (w *walWriter) shift() error {
	err := w.sync()
	if err != nil {
		return err
	}
	err = w.tail.File.Close()
	if err != nil {
		return errors.WithStack(err)
	}
	w.tail.File, err = os.OpenFile(fileFor(w.id, w.tailIdx+1), os.O_CREATE|os.O_WRONLY|os.O_APPEND, walPermission)
	if err != nil {
		return errors.WithStack(err)
	}
	w.tailIdx++
	return nil
}

func (w *walWriter) doHousekeeping() {
	wi, err := readWALInfo(w.id)
	if err != nil {
		panic(err)
	}

	w.mutex.Lock()
	defer w.mutex.Unlock()

	//log.Printf("wi.tailSize=%v cfg.FileLimit=%v\n", wi.tailSize, w.cfg.FileLimit)
	if wi.tailSize > w.cfg.FileLimit {
		err := w.shift()
		if err != nil {
			panic(err)
		}
	} else {
		eud := w.eldestUnsyncData
		if eud != nil && eud.Add(w.cfg.SyncInterval).Before(time.Now()) {
			log.Must(w.sync())
		}
	}
	//log.Printf("wi.totalSize=%v cfg.TotalLimit=%v\n", wi.totalSize, w.cfg.TotalLimit)
	for wi.totalSize > w.cfg.TotalLimit {
		pth := fileFor(w.id, wi.headIdx)
		fInfo, err := os.Stat(pth)
		if err != nil {
			err = errors.WithStack(err)
			panic(err)
		}
		err = os.Remove(pth)
		if err != nil {
			err = errors.WithStack(err)
			panic(err)
		}
		wi.totalSize -= fInfo.Size()
		wi.headIdx++
	}
}

type walReader struct {
	files       []*os.File
	reader      io.Reader
	validOffset int64
	id          string
	wi          *walInfo
}

func OpenWALForRead(id string) (WALReader, error) {
	wi, err := readWALInfo(id)
	if err != nil {
		return nil, err
	}
	if wi.headIdx > wi.tailIdx {
		return nil, errors.Wrapf(os.ErrNotExist, "no file for wal %v", id)
	}
	files := make([]*os.File, wi.tailIdx-wi.headIdx+1)
	readers := make([]io.Reader, len(files))

	defer func() {
		for _, f := range files {
			if f != nil {
				log.Must(f.Close())
			}
		}
	}()

	for i := 0; i < len(files); i++ {
		files[i], err = os.Open(fileFor(id, wi.headIdx+uint64(i)))
		if err != nil {
			return nil, errors.WithStack(err)
		}
		readers[i] = files[i]
	}
	w := &walReader{}
	w.reader = bufio.NewReaderSize(io.MultiReader(readers...), configWALBufSize)
	w.files = files
	files = nil
	w.id = id
	w.wi = wi
	return w, nil
}

func (w *walReader) ReadBytes() ([]byte, error) {
	header := make([]byte, headerLen)
	_, err := io.ReadAtLeast(w.reader, header, headerLen)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	crc := binary.BigEndian.Uint32(header[0:4])
	payloadLen := binary.BigEndian.Uint32(header[4:headerLen])
	payload := make([]byte, payloadLen)
	_, err = io.ReadAtLeast(w.reader, payload, int(payloadLen))
	if err != nil {
		return nil, errors.WithStack(err)
	}

	actualCRC := crc32.Checksum(payload, crc32c)
	//log.Printf("wal read crc=%x ccrc=%x payloadLen:%v payload:%x\n", crc, actualCRC, payloadLen, payload)
	if actualCRC != crc {
		return nil, errors.Wrapf(errCorruptedWAL, "bad crc: read:%x actural:%x payloadLen:%v payload:%x", crc, actualCRC, payloadLen, payload)
	}

	w.validOffset += int64(headerLen + payloadLen)
	return payload, nil
}

func WALReadObject(w WALReader, v interface{}) ([]byte, error) {
	bs, err := w.ReadBytes()
	if err != nil {
		return nil, err
	}
	return msgCodec.UnmarshalFromBytes(bs, v)
}

func (w *walReader) Close() error {
	for _, f := range w.files {
		err := f.Close()
		if err != nil {
			return errors.WithStack(err)
		}
	}
	w.files = nil
	return nil
}

func (w *walReader) CloseAndRepair() error {
	if err := w.Close(); err != nil {
		return err
	}

	left := w.validOffset
	idx := w.wi.headIdx
	for _, s := range w.wi.fileSizes {
		if left <= s {
			if left < s {
				err := os.Truncate(fileFor(w.id, idx), left)
				if err != nil {
					return errors.WithStack(err)
				}
			}
			for i := idx + 1; i <= w.wi.tailIdx; i++ {
				if err := os.Remove(fileFor(w.id, idx)); err != nil {
					return errors.WithStack(err)
				}
			}
			return nil
		}
		left -= s
		idx++
	}
	return nil
}

func IsCorruptedWAL(err error) bool {
	return errors.Is(err, errCorruptedWAL)
}

var errCorruptedWAL = errors.New("errCorruptedWAL")

func IsNotExist(err error) bool {
	return errors.Is(err, os.ErrNotExist)
}

func IsEOF(err error) bool {
	return errors.Is(err, io.EOF)
}

func IsUnexpectedEOF(err error) bool {
	return errors.Is(err, io.ErrUnexpectedEOF)
}

type WALManager interface {
	OpenForRead(id string) (WALReader, error)
	OpenForWrite(id string, cfg *WALConfig) (WALWriter, error)
}

type walManager struct {
}

var defaultWALManager = &walManager{}

func (wm *walManager) OpenForRead(id string) (WALReader, error) {
	return OpenWALForRead(id)
}

func (wm *walManager) OpenForWrite(id string, cfg *WALConfig) (WALWriter, error) {
	return OpenWALForWrite(id, cfg)
}

func ResetWAL(height int64, dir string, voteListBytes []byte) error {
	wm := defaultWALManager
	var err error
	if err = os.RemoveAll(dir); err != nil {
		return err
	}
	if voteListBytes == nil {
		return nil
	}
	ww, err := wm.OpenForWrite(path.Join(dir, configCommitWALID), &WALConfig{
		FileLimit:  configCommitWALDataSize,
		TotalLimit: configCommitWALDataSize * 3,
	})
	defer func() {
		log.Must(ww.Close())
	}()
	if err != nil {
		return err
	}
	if _, err = ww.WriteBytes(voteListBytes); err != nil {
		return err
	}
	return nil
}
