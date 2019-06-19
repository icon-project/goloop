package chain

import (
	"archive/zip"
	"bytes"
	"encoding/hex"
	"encoding/json"
	"io"
	"io/ioutil"
	"log"
	"os"
	"path"
	"regexp"
	"sort"

	"github.com/icon-project/goloop/common/crypto"
	"github.com/icon-project/goloop/common/errors"
	"github.com/icon-project/goloop/service/transaction"
	"golang.org/x/crypto/sha3"
)

const (
	GenesisFileName = "genesis.json"
)

type GenesisStorage interface {
	NID() (int, error)
	Genesis() []byte
	Get(key []byte) ([]byte, error)
}

type genesisStorageWithDataDir struct {
	genesis  []byte
	nid      int
	dataPath string
	dataMap  map[string]string
}

func GetNIDForGenesis(g []byte) (int, error) {
	gtx, err := transaction.NewGenesisTransaction(g)
	if err != nil {
		return 0, err
	}
	return gtx.NID(), nil
}

func (gs *genesisStorageWithDataDir) Genesis() []byte {
	return gs.genesis
}

func (gs *genesisStorageWithDataDir) NID() (int, error) {
	if gs.nid == 0 {
		if nid, err := GetNIDForGenesis(gs.Genesis()); err != nil {
			return 0, err
		} else {
			gs.nid = nid
		}
	}
	return gs.nid, nil
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

type genesisStorageWithZip struct {
	genesis []byte
	nid     int
	fileMap map[string]*zip.File
}

func (gs *genesisStorageWithZip) Genesis() []byte {
	return gs.genesis
}

func (gs *genesisStorageWithZip) NID() (int, error) {
	if gs.nid == 0 {
		if nid, err := GetNIDForGenesis(gs.Genesis()); err != nil {
			return 0, err
		} else {
			gs.nid = nid
		}
	}
	return gs.nid, nil
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
	writer *zip.Writer
}

func (c *templateContext) AddData(data []byte) (string, error) {
	hash := hex.EncodeToString(crypto.SHA3Sum256(data))
	f, err := c.writer.Create(hash)
	if err != nil {
		return "", err
	}
	if _, err := f.Write(data); err != nil {
		return "", err
	}
	return hash, nil
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
func WriteGenesisStorageFromPath(w io.Writer, p string) error {
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

	zw := zip.NewWriter(w)
	defer zw.Close()

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
		writer: zw,
		path:   genesisDir,
	}, genesisObj)

	// write genesis data at last
	f, err := zw.Create(GenesisFileName)
	if err != nil {
		return errors.Wrapf(err, "Fail to create %s", GenesisFileName)
	}
	genesis, err = json.Marshal(genesisObj)
	if err != nil {
		return errors.Wrap(err, "Fail to marshal JSON")
	}
	_, err = f.Write(genesis)
	if err != nil {
		return errors.Wrap(err, "Fail to write genesis")
	}
	_ = zw.Flush()
	return nil
}

func NewGenesisStorageFromFile(fd *os.File) (GenesisStorage, error) {
	fi, err := fd.Stat()
	if err != nil {
		return nil, err
	}
	return newGenesisStorage(fd, fi.Size())
}

func NewGenesisStorage(data []byte) (GenesisStorage, error) {
	return newGenesisStorage(bytes.NewReader(data), int64(len(data)))
}

func newGenesisStorage(readerAt io.ReaderAt, size int64) (GenesisStorage, error) {
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
