package main

import (
	"archive/zip"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

const (
	URL_VERSION = "https://storage.googleapis.com/appengine-sdks/featured/VERSION"
	URL_SDK     = "https://storage.googleapis.com/appengine-sdks/featured/go_appengine_sdk_linux_amd64-%s.zip"
	TEMP_FILE   = "/tmp/go_appengine.zip"
)

var (
	version, install string
)

func init() {
	pwd, err := os.Getwd()
	if err != nil {
		log.Fatal(err)
	}
	err = getVersion()
	if err != nil {
		log.Fatal(err)
	}

	flag.StringVar(&version, "version", version, "Version of App Engine SDK")
	flag.StringVar(&install, "install", pwd, "Directory to install sdk")
}

func main() {
	flag.Parse()

	log.Println("Google Appengine SDK Manager")
	log.Println("Using version " + version)

	local, err := verifyVersion()
	if err != nil {
		log.Fatal(err)
	}
	if local == "" {
		log.Printf("No versions found in %s/\n", install)
	} else if local == version {
		log.Printf("You are using the version %s in %s\n", local, install)
		log.Println("Aborting")
		return
	} else {
		log.Printf("Found version %s installed in %s\n", local, install)
		log.Println("Backup your old version")
		err = os.Rename(install+"/go_appengine", install+"/go_appengine-"+local)
		if err != nil {
			log.Fatal(err)
		}
	}

	log.Println("Downloading...")
	download()
	log.Println("Extracting new version in " + install)
	unzip(TEMP_FILE)
	log.Println("Done")
}

func getVersion() error {
	resp, err := http.Get(URL_VERSION)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	version, err = readVersion(resp.Body)
	if err != nil {
		return err
	}
	return nil
}

func verifyVersion() (string, error) {
	file, err := os.Open(install + "/go_appengine/VERSION")
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil
		}
		return "", err
	}

	v, err := readVersion(file)
	if err != nil {
		return "", err
	}
	return v, nil
}

func readVersion(reader io.ReadCloser) (string, error) {
	b, err := ioutil.ReadAll(reader)
	if err != nil {
		return "", err
	}
	lines := strings.Split(string(b), "\n")
	for _, l := range lines {
		s := strings.Split(l, ":")
		if s[0] == "release" {
			reg := regexp.MustCompile("[^0-9//.]+")
			return reg.ReplaceAllString(s[1], ""), nil
		}
	}
	return "", fmt.Errorf("Not found")
}

func download() error {
	resp, err := http.Get(fmt.Sprintf(URL_SDK, version))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	out, err := os.Create(TEMP_FILE)
	if err != nil {
		return err
	}
	defer out.Close()

	n, err := io.Copy(out, resp.Body)
	if err != nil {
		return err
	}
	log.Printf("Total of bytes %v\n", n)

	return unzip(TEMP_FILE)
}

func unzip(zipfile string) error {
	reader, err := zip.OpenReader(zipfile)
	if err != nil {
		return err
	}
	defer reader.Close()

	for _, f := range reader.Reader.File {
		zipped, err := f.Open()
		if err != nil {
			return err
		}
		defer zipped.Close()

		// get the individual file name and extract the current directory
		path := filepath.Join(install, f.Name)

		if f.FileInfo().IsDir() {
			os.MkdirAll(path, f.Mode())
			//fmt.Println("Creating directory", path)
		} else {
			writer, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE, f.Mode())
			if err != nil {
				return err
			}

			defer writer.Close()

			if _, err = io.Copy(writer, zipped); err != nil {
				return err
			}
			//fmt.Println("Decompressing : ", path)
		}
	}
	return nil
}
