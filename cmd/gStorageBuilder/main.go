package main

import (
	"archive/zip"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path"
	"strings"

	"golang.org/x/crypto/sha3"
)

const (
	GenesisFileName = "genesis.json"
)

func addContent(writer *zip.Writer, dir string, name string) string {
	content, err := ioutil.ReadFile(path.Join(dir, name))
	if err != nil {
		log.Printf("Failed to read file(%s). %s\n", content, err)
		return ""
	}

	hash := fmt.Sprintf("%x", sha3.Sum256(content))

	contentFile, err := writer.Create(hash)
	if err != nil {
		log.Printf("Failed to create file.\n")
		return ""
	}
	contentFile.Write(content)
	return hash
}

func extractContent(js interface{}, contentMap map[string]string, writer *zip.Writer, dirPath string) {
	switch v := js.(type) {
	case []interface{}:
		for _, v2 := range v {
			extractContent(v2, contentMap, writer, dirPath)
		}
	case map[string]interface{}:
		for k, v2 := range v {
			if strings.Compare(k, "@contentId") == 0 {
				delete(v, k)
				if s, ok := v2.(string); ok {
					hash := addContent(writer, dirPath, s)
					v["contentId"] = "hash:" + hash
					contentMap[s] = hash
				}
			} else if strings.Compare(k, "@content") == 0 {
				delete(v, k)
				if s, ok := v2.(string); ok {
					filePath := path.Join(dirPath, s)
					data, err := ioutil.ReadFile(filePath)
					if err != nil {
						fmt.Printf("failed to readfile. err = %s, file = %s\n", err, filePath)
					}
					v["content"] = data
				}
			} else {
				extractContent(v2, contentMap, writer, dirPath)
			}
		}
	}
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
		json.Unmarshal(genesis, &gMap)
		extractContent(gMap, contentMap, zWriter, dir)
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
