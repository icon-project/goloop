package gs

import (
	"archive/zip"
	"bytes"
	"encoding/hex"
	"encoding/json"
	"io"
	"io/ioutil"
	"os"
	"path"
	"regexp"
	"sort"

	"github.com/icon-project/goloop/common/crypto"
	"github.com/icon-project/goloop/common/errors"
	"github.com/icon-project/goloop/common/log"
	"github.com/icon-project/goloop/module"
	"github.com/icon-project/goloop/service/transaction"
)

const (
	GenesisFileName = "genesis.json"
)

type genesisStorageImpl interface {
	Genesis() []byte
	Get(key []byte) ([]byte, error)
}

type genesisStorage struct {
	genesisStorageImpl
	cid, nid int
	height   int64
	gType    module.GenesisType
}

func (gs *genesisStorage) ensureTypeAndIDs() error {
	if gs.cid == 0 {
		if pg, err := newPrunedGenesis(gs.Genesis()); err == nil {
			gs.cid = int(pg.CID.Value)
			gs.nid = int(pg.NID.Value)
			gs.gType = module.GenesisPruned
			gs.height = pg.Height.Value
			return nil
		}
		gtx, err := transaction.NewGenesisTransaction(gs.Genesis())
		if err != nil {
			return err
		}
		gs.cid = gtx.CID()
		gs.nid = gtx.NID()
		gs.height = 0
		gs.gType = module.GenesisNormal
	}
	return nil
}

func (gs *genesisStorage) Type() (module.GenesisType, error) {
	if err := gs.ensureTypeAndIDs(); err != nil {
		return module.GenesisUnknown, err
	}
	return gs.gType, nil
}

func (gs *genesisStorage) CID() (int, error) {
	if err := gs.ensureTypeAndIDs(); err != nil {
		return 0, err
	}
	return gs.cid, nil
}

func (gs *genesisStorage) NID() (int, error) {
	if err := gs.ensureTypeAndIDs(); err != nil {
		return 0, err
	}
	return gs.nid, nil
}

func (gs *genesisStorage) Height() int64 {
	if err := gs.ensureTypeAndIDs(); err != nil {
		return 0
	}
	return gs.height
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

type templateContext struct {
	path   string
	writer module.GenesisStorageWriter
}

func (c *templateContext) AddData(data []byte) (string, error) {
	key, err := c.writer.WriteData(data)
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(key), nil
}

func writeToZip(writer *zip.Writer, p, n string) error {
	p2 := path.Join(p, n)
	st, err := os.Stat(p2)
	if err != nil {
		return errors.Wrap(err, "writeToZip: FAIL on os.State")
	}
	if !st.IsDir() {
		fd, err := os.Open(p2)
		defer fd.Close()

		if err != nil {
			return errors.Wrapf(err, "writeToZip: fail to open %s", p2)
		}
		zf, err := writer.Create(n)
		if err != nil {
			return errors.Wrapf(err, "writeToZip: fail to create entry %s", n)
		}
		if _, err := io.Copy(zf, fd); err != nil {
			return errors.Wrap(err, "writeToZip: fail to copy")
		}
		return nil
	}

	fis, err := ioutil.ReadDir(p2)
	if err != nil {
		return errors.Wrap(err, "writeToZip: FAIL on ReadDir")
	}
	// make it generate consistent compressed zip file.
	sort.SliceStable(fis, func(i, j int) bool {
		return fis[i].Name() < fis[j].Name()
	})
	for _, fi := range fis {
		if err := writeToZip(writer, p, path.Join(n, fi.Name())); err != nil {
			return err
		}
	}
	return nil
}

func zipDirectory(p string) ([]byte, error) {
	bs := bytes.NewBuffer(nil)
	zw := zip.NewWriter(bs)

	if err := writeToZip(zw, p, ""); err != nil {
		zw.Close()
		return nil, err
	} else {
		zw.Flush()
		zw.Close()
		return bs.Bytes(), nil
	}
}

var regexTemplate = regexp.MustCompile("{{(read|hash|zip|ziphash):([^}]+)}}")

func processTemplate(c *templateContext, s string) (r string, e error) {
	for {
		m := regexTemplate.FindStringSubmatchIndex(s)
		if len(m) == 0 {
			break
		}
		key := s[m[2]:m[3]]
		p := path.Join(c.path, s[m[4]:m[5]])
		switch key {
		case "zip":
			data, err := zipDirectory(p)
			if err != nil {
				return s, err
			}
			s = s[0:m[0]] + "0x" + hex.EncodeToString(data) + s[m[1]:]

		case "read":
			data, err := ioutil.ReadFile(p)
			if err != nil {
				return s, err
			}
			s = s[0:m[0]] + "0x" + hex.EncodeToString(data) + s[m[1]:]

		case "hash":
			data, err := ioutil.ReadFile(p)
			if err != nil {
				return s, err
			}
			hash, err := c.AddData(data)
			if err != nil {
				return s, err
			}
			s = s[0:m[0]] + "0x" + hash + s[m[1]:]

		case "ziphash":
			data, err := zipDirectory(path.Join(p))
			if err != nil {
				return s, err
			}
			hash, err := c.AddData(data)
			if err != nil {
				return s, err
			}
			s = s[0:m[0]] + "0x" + hash + s[m[1]:]

		default:
			return s, errors.IllegalArgumentError.Errorf(
				"Unknown keyword:%q for %q", key, s)
		}
	}
	return s, nil
}

func processContent(c *templateContext, o interface{}) (interface{}, error) {
	switch obj := o.(type) {
	case []interface{}:
		for i, v := range obj {
			if no, err := processContent(c, v); err != nil {
				return nil, err
			} else {
				obj[i] = no
			}
		}
		return obj, nil
	case map[string]interface{}:
		for k, v := range obj {
			if no, err := processContent(c, v); err != nil {
				return nil, err
			} else {
				obj[k] = no
			}
		}
		return obj, nil
	case string:
		return processTemplate(c, obj)
	default:
		return o, nil
	}
}

// WriteGenesisStorageFromPath write genesis data from the template.
// You may specify directory containing genesis.json. Or specify template
// file itself.
func WriteFromPath(w io.Writer, p string) error {
	var genesisDir, genesisTemplate string

	st, err := os.Stat(p)
	if err != nil {
		return err
	}
	if st.IsDir() {
		genesisDir = p
		genesisTemplate = path.Join(p, GenesisFileName)
	} else {
		genesisDir, _ = path.Split(p)
		genesisTemplate = p
	}

	gsw := NewGenesisStorageWriter(w)
	defer gsw.Close()

	// load and decode genesis
	genesis, err := ioutil.ReadFile(genesisTemplate)
	if err != nil {
		return errors.Wrapf(err, "Fail to read %s", genesisTemplate)
	}
	d := json.NewDecoder(bytes.NewBuffer(genesis))
	d.UseNumber()
	var genesisObj map[string]interface{}
	err = d.Decode(&genesisObj)
	if err != nil {
		return errors.Wrapf(err, "Fail to decode %s", genesisTemplate)
	}

	// process genesis template
	_, err = processContent(&templateContext{
		writer: gsw,
		path:   genesisDir,
	}, genesisObj)
	if err != nil {
		return errors.Wrap(err, "Fail to process content")
	}

	// write genesis data at last
	genesis, err = json.Marshal(genesisObj)
	if err != nil {
		return errors.Wrap(err, "Fail to marshal JSON")
	}
	if err := gsw.WriteGenesis(genesis); err != nil {
		return errors.Wrap(err, "Fail to write genesis")
	}
	return nil
}

func NewFromTx(tx []byte) module.GenesisStorage {
	return &genesisStorage{
		genesisStorageImpl: &genesisStorageWithDataDir{
			genesis:  tx,
			dataMap:  nil,
			dataPath: "",
		},
	}
}

func NewFromFile(fd *os.File) (module.GenesisStorage, error) {
	fi, err := fd.Stat()
	if err != nil {
		return nil, err
	}
	return newGenesisStorage(fd, fi.Size())
}

func New(data []byte) (module.GenesisStorage, error) {
	return newGenesisStorage(bytes.NewReader(data), int64(len(data)))
}

func newGenesisStorage(readerAt io.ReaderAt, size int64) (module.GenesisStorage, error) {
	reader, err := zip.NewReader(readerAt, size)
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
				log.Warnf("InvalidFileName(name=%s)", f.Name)
				continue
			}
			m[string(key)] = f
		}
	}
	if genesis == nil {
		return nil, errors.New("IllegalFormatNoGenesis")
	}
	return &genesisStorage{
		genesisStorageImpl: &genesisStorageWithZip{
			genesis: genesis,
			fileMap: m,
		},
	}, nil
}
