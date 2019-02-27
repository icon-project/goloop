package chain

import (
	"archive/zip"
	"bytes"
	"encoding/hex"
	"io"
	"io/ioutil"
	"log"
	"os"
	"path"

	"github.com/icon-project/goloop/common/crypto"
	"github.com/pkg/errors"
	"golang.org/x/crypto/sha3"
)

const (
	GenesisFileName = "genesis.json"
)

type GenesisStorage interface {
	Genesis() []byte
	Get(key []byte) ([]byte, error)
}

type genesisStorageWithDataDir struct {
	genesis  []byte
	dataPath string
	dataMap  map[string]string
}

func (gs *genesisStorageWithDataDir) Genesis() []byte {
	return gs.genesis
}

func (gs *genesisStorageWithDataDir) Get(key []byte) ([]byte, error) {
	if gs.dataMap == nil {
		return nil, nil
	}
	sKey := string(key)
	if f, ok := gs.dataMap[sKey]; ok {
		p := path.Join(gs.dataPath, f)
		if bs, err := ioutil.ReadFile(p); err != nil {
			return nil, err
		} else {
			hash := crypto.SHA3Sum256(bs)
			if bytes.Equal(hash, key) {
				return bs, nil
			} else {
				return nil, errors.New("Invalid data")
			}
		}
	} else {
		return nil, nil
	}
}

const (
	GenesisChunkSize = 1024 * 10
)

func SHA3Sum256WithReadCloser(rc io.ReadCloser) ([]byte, error) {
	s := sha3.New256()
	buf := make([]byte, GenesisChunkSize)
	for {
		r, err := rc.Read(buf)
		if err == io.EOF {
			break
		}
		if err != nil {
			rc.Close()
			return nil, err
		}
		s.Write(buf[0:r])
	}
	if err := rc.Close(); err != nil {
		return nil, err
	}
	return s.Sum([]byte{}), nil
}

func NewGenesisStorageWithDataDir(genesis []byte, p string) (GenesisStorage, error) {
	if p == "" {
		return &genesisStorageWithDataDir{
			genesis:  genesis,
			dataMap:  nil,
			dataPath: "",
		}, nil
	}
	items, err := ioutil.ReadDir(p)
	if err != nil {
		return nil, err
	}
	m := make(map[string]string)
	for _, info := range items {
		if info.IsDir() {
			continue
		}
		f, err := os.Open(path.Join(p, info.Name()))
		if err != nil {
			return nil, err
		}
		hash, err := SHA3Sum256WithReadCloser(f)
		if err != nil {
			return nil, err
		}
		m[string(hash)] = info.Name()
	}
	return &genesisStorageWithDataDir{
		genesis:  genesis,
		dataMap:  m,
		dataPath: p,
	}, nil
}

type genesisStorageWithZip struct {
	genesis []byte
	fileMap map[string]*zip.File
}

func (gs *genesisStorageWithZip) Genesis() []byte {
	return gs.genesis
}

func readAllOfZipFile(f *zip.File) ([]byte, error) {
	rc, err := f.Open()
	if err != nil {
		return nil, err
	}
	bs, err := ioutil.ReadAll(rc)
	if err != nil {
		return nil, err
	}
	rc.Close()
	return bs, nil
}

func (gs *genesisStorageWithZip) Get(key []byte) ([]byte, error) {
	f, ok := gs.fileMap[string(key)]
	if !ok {
		return nil, nil
	}

	bs, err := readAllOfZipFile(f)
	if err != nil {
		return nil, err
	}

	hash := crypto.SHA3Sum256(bs)
	if bytes.Equal(hash, key) {
		return bs, nil
	} else {
		return nil, errors.Errorf("InvalidData(hash=<%x>,key=<%x>", hash, key)
	}
}

func NewGenesisStorage(data []byte) (GenesisStorage, error) {
	reader, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return nil, err
	}
	var genesis []byte
	m := make(map[string]*zip.File)
	for _, f := range reader.File {
		if f.Name == GenesisFileName {
			genesis, err = readAllOfZipFile(f)
			if err != nil {
				return nil, err
			}
		} else {
			key, err := hex.DecodeString(f.Name)
			if err != nil {
				log.Printf("InvalidFileName(fname=%s)", f.Name)
				continue
			}
			m[string(key)] = f
		}
	}
	if genesis == nil {
		return nil, errors.New("IllegalFormatNoGenesis")
	}
	return &genesisStorageWithZip{
		genesis: genesis,
		fileMap: m,
	}, nil
}
