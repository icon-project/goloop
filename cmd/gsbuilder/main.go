package main

import (
	"archive/zip"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path"
	"regexp"
	"strings"

	"github.com/icon-project/goloop/common/crypto"
)

const (
	GenesisFileName = "genesis.json"
)

type templateContext struct {
	path   string
	writer *zip.Writer
}

func extractContent(js interface{}, contentMap map[string]string, c *templateContext) error {
	switch v := js.(type) {
	case []interface{}:
		for _, v2 := range v {
			if err := extractContent(v2, contentMap, c); err != nil {
				return err
			}
		}
	case map[string]interface{}:
		for k, v2 := range v {
			if str, ok := v2.(string); ok {
				// processing template
				r := regexp.MustCompile("{{(read|hash):([^>]+)}}")
				for {
					m := r.FindStringSubmatch(str)
					if len(m) == 0 {
						break
					}
					key := m[1]
					if key == "read" {
						name := m[2]
						data, err := ioutil.ReadFile(path.Join(c.path, name))
						if err != nil {
							return err
						}
						str = strings.Replace(str, m[0], "0x"+hex.EncodeToString(data), 1)
					} else {
						name := m[2]
						data, err := ioutil.ReadFile(path.Join(c.path, name))
						if err != nil {
							return err
						}
						hash := hex.EncodeToString(crypto.SHA3Sum256(data))
						f, err := c.writer.Create(hash)
						if err != nil {
							return err
						}
						f.Write(data)
						str = strings.Replace(str, m[0], "0x"+hash, 1)
					}
					v[k] = str
				}
			} else {
				if err := extractContent(v2, contentMap, c); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

func buildStorage(storageName string, dir string) map[string]string {
	zipFile, err := os.Create(storageName)
	defer zipFile.Close()
	if err != nil {
		log.Printf("Failed to create file(%s). err = %s\n", storageName, err)
		return nil
	}
	zWriter := zip.NewWriter(zipFile)
	defer zWriter.Close()
	contentMap := make(map[string]string)
	genesis, err := ioutil.ReadFile(path.Join(dir, GenesisFileName))
	if err != nil {
		log.Printf("Failed to read file(%s). %s\n", GenesisFileName, err)
		return nil
	}
	if len(genesis) != 0 {
		gMap := make(map[string]interface{})
		if err := json.Unmarshal(genesis, &gMap); err != nil {
			log.Printf("Failed to unmarshal. err = %s\n", err)
			return nil
		}
		if err := extractContent(gMap, contentMap,
			&templateContext{dir, zWriter}); err != nil {
			log.Printf("Failed to extrace content. err = %s\n", err)
			return nil
		}
		if len(gMap) != 0 {
			genesis, _ = json.Marshal(gMap)
		}
		contentFile, err := zWriter.Create(GenesisFileName)
		if err != nil {
			log.Printf("Failed to create file.\n")
			return nil
		}
		fmt.Printf("genesis = %s\n", genesis)
		contentFile.Write(genesis)
	} else {
		panic("Failed to get genesis")
	}
	return contentMap
}

func main() {
	var dir, gStorage string
	flag.StringVar(&dir, "dir", "storage", "storage directory")
	flag.StringVar(&gStorage, "o", "genesisStorage.zip", "genesis storage file")
	flag.Parse()

	if fi, err := os.Stat(dir); os.IsNotExist(err) {
		log.Printf("Failed to find dir. err = %s\n", err)
		return
	} else if fi.IsDir() == false {
		log.Printf("%s is not a directory.\n", dir)
		return
	}

	contentMap := buildStorage(gStorage, dir)

	if _, err := os.Stat(gStorage); os.IsNotExist(err) {
		fmt.Printf("Failed to create file. %s does not exist. %s\n", gStorage, err)
		return
	}

	for k, v := range contentMap {
		fmt.Printf("%s -> %s\n", k, v)
	}

	fmt.Printf("genesis storage -> %s\n", gStorage)
}
