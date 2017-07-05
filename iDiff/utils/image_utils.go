package utils

import (
	//"fmt"
	//"encoding/json"
	//"encoding/base64"
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/system"
)

// ImageToDir converts an image to an unpacked tar and creates a representation of that directory.
func ImageToDir(img string) (string, string, error) {
	cli, err := client.NewEnvClient()
	if err != nil {
		return "", "", err
	}
	tarPath, err := ImageToTar(cli, img)
	if err != nil {
		return "", "", err
	}
	err = ExtractTar(tarPath)
	if err != nil {
		return "", "", err
	}
	os.Remove(tarPath)
	path := strings.TrimSuffix(tarPath, filepath.Ext(tarPath))
	jsonPath := path + ".json"
	err = DirToJSON(path, jsonPath, false) // TODO: Obtain deep parameter from flag
	if err != nil {
		return "", "", err
	}
	return jsonPath, path, nil
}

// ImageToTar writes an image to a .tar file
func ImageToTar(cli client.APIClient, image string) (string, error) {
	if checkImageID(image) {
		//fmt.Println("NO HERE")
		imgBytes, err := cli.ImageSave(context.Background(), []string{image})
		if err != nil {
			return "", err
		}
		defer imgBytes.Close()
		newpath := image + ".tar"
		return newpath, copyToFile(newpath, imgBytes)
	} else {
		//fmt.Println("HERE")
		/*authConfig := types.AuthConfig{
			Username: "colettet@google.com",
			Password: "password",
		}
		encodedJSON, err := json.Marshal(authConfig)
		if err != nil {
			panic(err)
		}
		authStr := base64.URLEncoding.EncodeToString(encodedJSON)*/
		imgBytes, err := cli.ImagePull(context.Background(), image, types.ImagePullOptions{})
		if err != nil {
			return "", err
		}
		defer imgBytes.Close()
		//io.Copy(os.Stdout, imgBytes)

		d := json.NewDecoder(events)

	    type Event struct {
	        Status         string `json:"status"`
	        Error          string `json:"error"`
	        Progress       string `json:"progress"`
	        ProgressDetail struct {
	            Current int `json:"current"`
	            Total   int `json:"total"`
	        } `json:"progressDetail"`
	    }

	    var events []*Event
	    for {
	    	var event *Event
	        if err := d.Decode(&event); err != nil {
	            if err == io.EOF {
	                break
	            }

	            return "", err
	        }

	        events = append(events, event)
	    }

	    // Latest event for new image
	    // EVENT: {Status:Status: Downloaded newer image for busybox:latest Error: Progress:[==================================================>]  699.2kB/699.2kB ProgressDetail:{Current:699243 Total:699243}}
	    // Latest event for up-to-date image
	    // EVENT: {Status:Status: Image is up to date for busybox:latest Error: Progress: ProgressDetail:{Current:0 Total:0}}
	    if events != nil {
	    	digestStatus = events[len(events)-2].Status
	    	pattern := regexp.MustCompile("^Digest: (sha256[a-z|0-9]{64})$")
			match := pattern.FindStringSubmatch(digestStatus)
	        if match != "" {
	        	tagIndex := strings.LastIndex(":", image)
	        	if tagIndex > 0 {
	        		image = image[:tagIndex]
	        	}
	        	imgBytes, err := cli.ImageSave(context.Background(), []string{image})
				if err != nil {
					return "", err
				}
				defer imgBytes.Close()
				newpath := image[25:32] + image[33:34] + ".tar"
				return newpath, copyToFile(newpath, imgBytes)
	        }
	    }

		return "", errors.New("Could not pull image from URL")
	}
}

// copyToFile writes the content of the reader to the specified file
func copyToFile(outfile string, r io.Reader) error {
	// We use sequential file access here to avoid depleting the standby list
	// on Windows. On Linux, this is a call directly to ioutil.TempFile
	tmpFile, err := system.TempFileSequential(filepath.Dir(outfile), ".docker_temp_")
	if err != nil {
		return err
	}

	tmpPath := tmpFile.Name()

	_, err = io.Copy(tmpFile, r)
	tmpFile.Close()

	if err != nil {
		os.Remove(tmpPath)
		return err
	}

	if err = os.Rename(tmpPath, outfile); err != nil {
		os.Remove(tmpPath)
		return err
	}

	return nil
}

func checkImageID(arg string) bool {
	pattern := regexp.MustCompile("[a-z|0-9]{12}")
	if exp := pattern.FindString(arg); exp != arg {
		return false
	}
	return true
}
