package differs

import (
	"errors"

	"github.com/GoogleCloudPlatform/runtimes-common/iDiff/utils"
)

type DiffRequest struct {
	Image1    utils.Image
	Image2    utils.Image
	DiffTypes  []Differ
	UseDocker bool
}

type DiffResult interface {
	OutputJSON() error
	OutputText() error
}

type Differ interface {
	Diff(image1, image2 utils.Image, eng bool) (DiffResult, error)
}

var diffs = map[string]Differ{
	"hist":    HistoryDiffer{},
	"history": HistoryDiffer{},
	"file":    FileDiffer{},
	"apt":     AptDiffer{},
	"linux":   AptDiffer{},
	"pip":     PipDiffer{},
	"node":    NodeDiffer{},
}

func (diff DiffRequest) GetDiff() (results map[string]DiffResult, err error) {
	img1 := diff.Image1
	img2 := diff.Image2
	diffs := diff.DiffTypes
	eng := diff.UseDocker

	for _, differ := range diffs {
		differName := reflect.TypeOf(diff).Name()
		if diff, err := differ.Diff(img1, img2, eng); err == nil {
			results[differName] = diff
		} else {
			glog.Errorf("Error getting diff with %s", differName)
		}
	}

	if len(results) == 0 {
		err = errors.New("Could not perform diff on %s and %s", img1, img2) 
	}
	return
}

func GetDiffers(diffNames []string) (differs []Differ, err error) {
	var differs []Differ
	for _, diffName := range diffNames{
		if d, exists := diffs[diffName]; exists {
			differs = append(differs, d)
		} else {
			glog.Errorf("Unknown differ specified", diffName)
		}
	}
	if len(differs) == 0 {
		err = errors.New("No known differs specified")
	}
	return
}
