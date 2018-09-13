package sync

import (
	"errors"
	"flag"
	"io"
	"io/ioutil"
	"log"
	"os"
	"path"
	"path/filepath"
	"strings"
)

const (
	// The paths here are given in relative location, and are expanded at runtime

	// ConfigPath is where all information of drivesync is stored
	ConfigPath = "~/.drivesync"
	// CredentialsFile is the location that credentials are copied to and looks for by default
	CredentialsFile = "credentials.json"
	// TokenFile is the location of where the authentication token is stored
	TokenFile = "token.json"
	// LogFile is the location that logs are outputted to
	LogFile = "drivesync.log"
)

// RunConfig holds all of the run configurations
type RunConfig struct {
	CredentialsFilepath string
	ParentFolder        string
	DriveFolder         string
	Once                bool
}

// SetUp configures the run configuration and returns all of the flags needed to run the program
// should be the first thing called in main since it configures the logging and run-time information
func SetUp() (*RunConfig, error) {
	// first thing done so we capture all the logging
	configureLogging()

	configDir := expandUser(ConfigPath)

	config, err := getRunConfig(configDir)
	if err != nil {
		return nil, err
	}

	os.MkdirAll(configDir, os.ModePerm)

	if config.CredentialsFilepath != path.Join(configDir, TokenFile) {
		err = copyFile(config.CredentialsFilepath, path.Join(configDir, CredentialsFile))
		if err != nil {
			return nil, err
		}
	}

	return config, err
}

// GetTokenSaveLocation returns the save location for the drive auth token
func GetTokenSaveLocation() string {
	configDir := expandUser(ConfigPath)
	return path.Join(configDir, TokenFile)
}

func getRunConfig(configDir string) (*RunConfig, error) {
	config := RunConfig{}

	flag.StringVar(&config.CredentialsFilepath, "credentials", path.Join(configDir, CredentialsFile), "filepath to drive credentials, copies the file to the default location")
	flag.StringVar(&config.ParentFolder, "folder", "", "folder to sync to drive")
	flag.BoolVar(&config.Once, "once", false, "run the sync only once instead of continuously")

	flag.Parse()

	// get the drive name from the basename of the parent folder
	config.DriveFolder = filepath.Base(config.ParentFolder)

	var err error
	if config.ParentFolder == "" {
		err = errors.New("Folder to sync is a required argument")
	}
	return &config, err
}

func configureLogging() {

	fileName := path.Join(expandUser(ConfigPath), LogFile)

	f, err := os.OpenFile(fileName, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		log.Fatalf("error opening file: %v", err)
	}
	// we don't close the file because it should be open for the duration fo the program
	mw := io.MultiWriter(os.Stdout, f)

	log.SetFlags(log.LstdFlags | log.Lshortfile)
	log.SetOutput(mw)
	log.Println("Starting configure process")
}

func copyFile(src string, dest string) error {
	f, err := ioutil.ReadFile(src)
	if err != nil {
		return err
	}
	err = ioutil.WriteFile(dest, f, 0644)
	return err
}

func expandUser(path string) string {

	home := os.Getenv("HOME")
	if path == "~" {
		// In case of "~", which won't be caught by the "else if"
		path = home
	} else if strings.HasPrefix(path, "~/") {
		// Use strings.HasPrefix so we don't match paths like
		// "/something/~/something/"
		path = filepath.Join(home, path[2:])
	}
	return path
}
