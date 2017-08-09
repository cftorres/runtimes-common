package utils

import (
	"archive/tar"
	"compress/gzip"
	"encoding/json"
	"errors"
	"io/ioutil"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"strings"

	"github.com/containers/image/docker"
	"github.com/golang/glog"
	"github.com/docker/docker/api/types/container"
)

var sourceToPrepMap = map[string]Prepper{
	"ID":  IDPrepper{},
	"URL": CloudPrepper{},
	"tar": TarPrepper{},
}

var sourceCheckMap = map[string]func(string) bool{
	"ID":  CheckImageID,
	"URL": CheckImageURL,
	"tar": CheckTar,
}

type Image struct {
	Source  string
	FSPath  string
	EnvVars []string
	History []string
	Layers  []string
}

type ImagePrepper struct {
	Source string
}

type Prepper interface {
	init() error
	getFinalFS() (string, error)
	getHistory() []string
	getEnvVars() []string
}

func (p ImagePrepper) GetImage() (Image, error) {
	img := p.Source

	var prepper Prepper
	for source, check := range sourceCheckMap {
		if check(img) {
			typePrepper := reflect.TypeOf(sourceToPrepMap[source])
			prepper = reflect.New(typePrepper).Interface().(Prepper)
			reflect.ValueOf(prepper).Elem().Field(0).Set(reflect.ValueOf(p))
			break
		}
	}
	if prepper == nil {
		return Image{}, errors.New("Could not determine image source")
	}

	glog.Infof("Starting prep for image %s", p.Source)
	image, err := prep(prepper)
	if err != nil {
		return Image{}, err
	}
	image.Source = img
	glog.Infof("Finished prepping image %s", p.Source)

	return image, nil
}

func getImageFromTar(tarPath string) (string, error) {
	glog.Info("Extracting image tar to obtain image file system")
	path := strings.TrimSuffix(tarPath, filepath.Ext(tarPath))
	err := UnTar(tarPath, path)
	return path, err
}

// CloudPrepper prepares images sourced from a Cloud registry
type CloudPrepper struct {
	ImagePrepper
	imageJSON configJSON
}

func (p CloudPrepper) init() error {
	config, err := p.getConfig()
	if err != nil {
		return err
	}
	p.imageJSON = config
	return nil
}

func (p CloudPrepper) getFinalFS() (string, error) {
	URLPattern := regexp.MustCompile("^.+/(.+(:.+){0,1})$")
	URLMatch := URLPattern.FindStringSubmatch(p.Source)
	path := strings.Replace(URLMatch[1], ":", "", -1)
	ref, err := docker.ParseReference("//" + p.Source)
	if err != nil {
		return "", err
	}

	img, err := ref.NewImage(nil)
	if err != nil {
		glog.Errorf("Error referencing image %s from registry: %s", p.Source, err)
		return "", errors.New("Could not create image root filesystem")
	}
	defer img.Close()

	imgSrc, err := ref.NewImageSource(nil, nil)
	if err != nil {
		glog.Error(err)
		return "", err
	}

	if _, ok := os.Stat(path); ok != nil {
		os.MkdirAll(path, 0777)
	}

	for _, b := range img.LayerInfos() {
		bi, _, err := imgSrc.GetBlob(b)
		if err != nil {
			glog.Error(err)
		}
		gzf, err := gzip.NewReader(bi)
		if err != nil {
			glog.Error(err)
		}
		tr := tar.NewReader(gzf)
		err = unpackTar(tr, path)
		if err != nil {
			glog.Error(err)
		}
	}
	return path, nil
}

type imageHistoryItem struct {
	CreatedBy  string    `json:"created_by"`
}

type configJSON struct {
	Config container.Config `json:"config"`
	History []imageHistoryItem `json:"history"`
}

func (p CloudPrepper) getConfig() (configJSON, error) {
	ref, err := docker.ParseReference("//" + p.Source)
	if err != nil {
		return configJSON{}, err
	}

	img, err := ref.NewImage(nil)
	if err != nil {
		glog.Errorf("Error referencing image %s from registry: %s", p.Source, err)
		return configJSON{}, errors.New("Could not obtain image config")
	}
	defer img.Close()

	configBlob, err := img.ConfigBlob()
	if err != nil {
		glog.Errorf("Error obtaining config blob for image %s from registry: %s", p.Source, err)
		return configJSON{}, errors.New("Could not obtain image config")
	}

	var config configJSON
	err = json.Unmarshal(configBlob, &config)
	if err != nil {
		glog.Errorf("Error with config file struct for image %s: %s", p.Source, err)
		return configJSON{}, errors.New("Could not obtain image config")
	}
	return config, nil
}

func (p CloudPrepper) getHistory() []string {
	history := p.imageJSON.History
	strhistory := make([]string, len(history))
	for i, layer := range history {
		strhistory[i] = layer.CreatedBy
	}
	return strhistory
}

func (p CloudPrepper) getEnvVars() []string {
	return p.imageJSON.Config.Env
}

// IDPrepper prepares images sourced from a local Docker ID
type IDPrepper struct {
	ImagePrepper
	imageConfig container.Config
}

func (p IDPrepper) init() error {
	config, err := p.getConfig()
	if err != nil {
		return err	
	}
	p.imageConfig = config
	return nil
}

func (p IDPrepper) getConfig() (container.Config, error) {
	// check client compatibility with Docker API
	valid, err := ValidDockerVersion()
	if err != nil {
		return container.Config{}, err
	}
	var config container.Config
	if !valid {
		glog.Info("Docker version incompatible with api, shelling out to local Docker client.")
		config, err = getImageConfigCmd(p.Source)
	} else {
		config, err = getImageConfig(p.Source)
	}
	if err != nil {
		return container.Config{}, err
	}
	return config, nil
}

func (p IDPrepper) getFinalFS() (string, error) {
	// check client compatibility with Docker API
	valid, err := ValidDockerVersion()
	if err != nil {
		return "", err
	}
	var tarPath string
	if !valid {
		glog.Info("Docker version incompatible with api, shelling out to local Docker client.")
		tarPath, err = imageToTarCmd(p.Source, p.Source)
	} else {
		tarPath, err = saveImageToTar(p.Source, p.Source)
	}
	if err != nil {
		return "", err
	}

	defer os.Remove(tarPath)
	return getImageFromTar(tarPath)
}

func (p IDPrepper) getHistory() []string {	
	history, err := getHistoryList(p.Source)
	if err != nil {
		glog.Error("Could not obtain image history for %s: %s", p.Source, err)	
	}
	return history
}

func (p IDPrepper) getEnvVars() []string {	
	return p.imageConfig.Env
}

// TarPrepper prepares images sourced from a .tar
type TarPrepper struct {
	ImagePrepper
	imageJSON configJSON
}

func (p TarPrepper) init() error {
	config, err := p.getConfig()
	if err != nil {
		return err
	}
	p.imageJSON = config
	return nil
}

func (p TarPrepper) getConfig() (configJSON, error) {
	tmpDir := strings.TrimSuffix(p.Source, filepath.Ext(p.Source))
	defer os.Remove(tmpDir)
	err := UnTar(p.Source, tmpDir)
	if err != nil {
		return configJSON{}, err
	}
	contents, err := ioutil.ReadDir(tmpDir)
	glog.Error(contents)
	if err != nil {
		glog.Errorf("Could not read image tar contents: %s", err)
		return configJSON{}, errors.New("Could not obtain image config")
	}
	
	var config configJSON
	configList := []string{}
	for _, item := range contents {
		if filepath.Ext(item.Name()) == ".json" && item.Name() != "manifest.json" {
			if len(configList) != 0 {
				// Another <image>.json file has already been processed and the image config obtained is uncertain.
				glog.Error("Multiple possible config sources detected for image at " + p.Source + ", some diff results may be incorrect.")
				break
			}
			fileName := filepath.Join(tmpDir, item.Name())
			file, err := ioutil.ReadFile(fileName)
			if err != nil {
				glog.Errorf("Could not read config file %s: %s", fileName, err)
				return configJSON{}, errors.New("Could not obtain image config")
			}
			var configFile configJSON
			json.Unmarshal(file, &configFile)
			config = configFile
			configList = append(configList, fileName)
		}
	}
/*	if (configJSON{}) == config {
		glog.Warningf("No image config found in tar source %s. Pip differ may be incomplete due to missing PYTHONPATH information.")
		return config, errors.New("Could not obtain image config")
	}*/
	return config, nil
}

func (p TarPrepper) getFinalFS() (string, error) {
	return getImageFromTar(p.Source)
}

func (p TarPrepper) getHistory() []string {
	history := p.imageJSON.History
	strhistory := make([]string, len(history))
	for i, layer := range history {
		strhistory[i] = layer.CreatedBy
	}
	return strhistory
}

func (p TarPrepper) getEnvVars() []string {
	return p.imageJSON.Config.Env
}

func prep(p Prepper) (Image, error) {
	err := p.init()
	if err != nil {
		return Image{}, err
	}
	
	imgFS, err := p.getFinalFS()
	if err != nil {
		return Image{}, err
	}

	return Image{
		FSPath: imgFS,
		History: p.getHistory(),
		EnvVars: p.getEnvVars(),
	}, nil
}
