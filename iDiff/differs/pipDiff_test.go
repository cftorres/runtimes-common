package differs

import (
	"reflect"
	"testing"

	"github.com/GoogleCloudPlatform/runtimes-common/iDiff/utils"
)

func TestGetPythonVersion(t *testing.T) {
	testCases := []struct {
		layerPath        string
		expectedVersions []string
		err              bool
	}{
		{
			layerPath:        "testDirs/pipTests/pythonVersionTests/notAFolder",
			expectedVersions: []string{},
			err:              true,
		},
		{
			layerPath:        "testDirs/pipTests/pythonVersionTests/noLibLayer",
			expectedVersions: []string{},
			err:              true,
		},
		{
			layerPath:        "testDirs/pipTests/pythonVersionTests/noPythonLayer",
			expectedVersions: []string{},
			err:              false,
		},
		{
			layerPath:        "testDirs/pipTests/pythonVersionTests/version2.7Layer",
			expectedVersions: []string{"python2.7"},
			err:              false,
		},
		{
			layerPath:        "testDirs/pipTests/pythonVersionTests/version3.6Layer",
			expectedVersions: []string{"python3.6"},
			err:              false,
		},
		{
			layerPath:        "testDirs/pipTests/pythonVersionTests/2VersionLayer",
			expectedVersions: []string{"python2.7", "python3.6"},
			err:              false,
		},
	}
	for _, test := range testCases {
		version, err := getPythonVersion(test.layerPath)
		if err != nil && !test.err {
			t.Errorf("Got unexpected error: %s", err)
		}
		if err == nil && test.err {
			t.Error("Expected error but got none.")
		}
		if !reflect.DeepEqual(version, test.expectedVersions) {
			t.Errorf("Expected: %s.  Got: %s", test.expectedVersions, version)
		}
	}
}

func TestGetPythonPackages(t *testing.T) {
	testCases := []struct {
		path             string
		expectedPackages map[string]map[string]utils.PackageInfo
	}{
		{
			path:             "testDirs/pipTests/noPackagesTest",
			expectedPackages: map[string]map[string]utils.PackageInfo{},
		},
		{
			path: "testDirs/pipTests/packagesOneLayer",
			expectedPackages: map[string]map[string]utils.PackageInfo{
				"packageone": {"python3.6": {Version: "3.6.9", Size: "4096"}},
				"packagetwo": {"python3.6": {Version: "4.6.2", Size: "4096"}},
				"script1.py": {"python3.6": {}},
				"script2.py": {"python3.6": {}},
			},
		},
		{
			path: "testDirs/pipTests/packagesMultiVersion",
			expectedPackages: map[string]map[string]utils.PackageInfo{
				"packageone": {"python3.6": {Version: "3.6.9", Size: "4096"},
					"python2.7": {Version: "0.1.1", Size: "4096"}},
				"packagetwo": {"python3.6": {Version: "4.6.2", Size: "4096"}},
				"script1.py": {"python3.6": {}},
				"script2.py": {"python3.6": {}},
				"script3.py": {"python2.7": {}},
			},
		},
	}
	for _, test := range testCases {
		packages := getPythonPackages(test.path)
		if !reflect.DeepEqual(packages, test.expectedPackages) {
			t.Errorf("Expected: %s but got: %s", test.expectedPackages, packages)
		}
	}
}
