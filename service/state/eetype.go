package state

import (
	"archive/zip"
	"bytes"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"

	"github.com/icon-project/goloop/common/errors"
	"github.com/icon-project/goloop/common/log"
	"github.com/icon-project/goloop/module"
	"github.com/icon-project/goloop/service/scoreresult"
)

type EEType string

const (
	eePython               = "python"
	eeJava                 = "java"
	javaCode               = "code.jar"
	tmpRoot                = "tmp"
	contractPythonRootFile = "package.json"
	tryTmpNum              = 10
)

func (e EEType) InstallMethod() string {
	switch e {
	case eePython:
		return "on_install"
	case eeJava:
		return "<init>"
	}
	log.Errorf("UnexpectedEEType(%s)\n", e)
	return ""
}

func storePython(path string, code []byte, log log.Logger) error {
	basePath, _ := filepath.Split(path)
	var tmpPath string
	var i int
	for i = 0; i < tryTmpNum; i++ {
		tmpPath = filepath.Join(basePath, tmpRoot, path+strconv.Itoa(i))
		if _, err := os.Stat(tmpPath); !os.IsNotExist(err) {
			if err := os.RemoveAll(tmpPath); err != nil {
				break
			}
		} else {
			break
		}
	}
	if i == tryTmpNum {
		return errors.CriticalIOError.Errorf("Fail to create temporary directory")
	}

	if err := os.MkdirAll(tmpPath, 0755); err != nil {
		return errors.WithCode(err, errors.CriticalIOError)
	}
	zipReader, err :=
		zip.NewReader(bytes.NewReader(code), int64(len(code)))
	if err != nil {
		return errors.WithCode(err, errors.CriticalIOError)
	}

	findRoot := false
	scoreRoot := ""
	for _, zipFile := range zipReader.File {
		if info := zipFile.FileInfo(); info.IsDir() {
			continue
		}
		if findRoot == false &&
			filepath.Base(zipFile.Name) == contractPythonRootFile {
			scoreRoot = filepath.Dir(zipFile.Name)
			findRoot = true
		}
		storePath := filepath.Join(tmpPath, zipFile.Name)
		storeDir := filepath.Dir(storePath)
		if _, err := os.Stat(storeDir); os.IsNotExist(err) {
			os.MkdirAll(storeDir, 0755)
		}
		reader, err := zipFile.Open()
		if err != nil {
			return scoreresult.IllegalFormatError.Wrap(err, "Fail to open zip file")
		}
		buf, err := ioutil.ReadAll(reader)
		if err != nil {
			reader.Close()
			return scoreresult.IllegalFormatError.Wrap(err, "Fail to read zip file")
		}
		if err = ioutil.WriteFile(storePath, buf, 0755); err != nil {
			return errors.CriticalIOError.Wrapf(err, "FailToWriteFile(name=%s)", storePath)
		}
		err = reader.Close()
		if err != nil {
			return errors.CriticalIOError.Wrap(err, "Fail to close zip file")
		}
	}
	if findRoot == false {
		os.RemoveAll(tmpPath)
		return scoreresult.IllegalFormatError.Errorf(
			"Root file does not exist(required:%s)\n", contractPythonRootFile)
	}
	contractRoot := filepath.Join(tmpPath, scoreRoot)
	if err := os.Rename(contractRoot, path); err != nil {
		log.Warnf("tmpPath(%s), scoreRoot(%s), err(%s)\n", tmpPath, scoreRoot, err)
		return errors.CriticalIOError.Wrapf(err, "FailToRenameTo(from=%s to=%s)", contractRoot, path)
	}
	if err := os.RemoveAll(tmpPath); err != nil {
		log.Debugf("Failed to remove tmpPath(%s), err(%s)\n", tmpPath, err)
	}
	return nil
}

func storeJava(path string, code []byte, log log.Logger) error {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		if err = os.MkdirAll(path, 0755); err != nil {
			return errors.WithCode(err, errors.CriticalIOError)
		}
	}
	sPath := filepath.Join(path, javaCode)
	if err := ioutil.WriteFile(sPath, code, 0755); err != nil {
		_ = os.RemoveAll(sPath)
		return errors.WithCode(err, errors.CriticalIOError)
	}
	return nil
}

func (e EEType) Store(path string, code []byte, log log.Logger) error {
	var err error
	switch e {
	case eePython:
		err = storePython(path, code, log)
	case eeJava:
		err = storeJava(path, code, log)
	default:
		err = scoreresult.Errorf(module.StatusInvalidParameter,
			"UnexpectedEEType(%v)\n", e)
	}
	return err
}

func (e EEType) String() string {
	return string(e)
}
