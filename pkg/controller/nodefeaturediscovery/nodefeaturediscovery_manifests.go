package nodefeaturediscovery

import (
	"io/ioutil"
	"os"
	"path/filepath"
)

type AssetsFromFile []byte

var manifests []AssetsFromFile

func filePathWalkDir(root string) ([]string, error) {
	var files []string
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if !info.IsDir() {
			files = append(files, path)
		}
		return nil
	})
	return files, err
}

func GetAssetsFromPath(path string) []AssetsFromFile {

	manifests := []AssetsFromFile{}
	assets := path
	files, err := filePathWalkDir(assets)
	if err != nil {
		panic(err)
	}
	for _, file := range files {
		buffer, err := ioutil.ReadFile(file)
		if err != nil {
			panic(err)
		}
		manifests = append(manifests, buffer)
	}
	return manifests
}
