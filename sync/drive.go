package sync

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strings"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/drive/v3"
	"google.golang.org/api/googleapi"
)

// external

// DriveService is a drive.Service wrapper used to make high-level calls to
type DriveService struct {
	service *drive.Service
}

// GetService returns a drive service to make calls to, returns and error if unable to create a service
func GetService(configFilePath string) (*DriveService, error) {

	b, err := ioutil.ReadFile(configFilePath)

	if err != nil {
		return nil, err
	}

	config, err := google.ConfigFromJSON(b, drive.DriveScope)
	if err != nil {
		return nil, err
	}

	client := getClient(config)
	srv, err := drive.New(client)

	return &DriveService{srv}, err
}

// GetFolderID retrieves the id of the folder, if the call does not succeed, returns and empty string with the error
func (srv DriveService) GetFolderID(folderName string) (string, error) {
	fieldString := "files(name, id)"
	qString := fmt.Sprintf("mimeType = 'application/vnd.google-apps.folder' and trashed = false and name = '%s'", folderName)

	r, err := srv.service.Files.List().PageSize(1).Fields(googleapi.Field(fieldString)).Q(qString).Do()

	if err != nil {
		return "", err
	}

	var id string
	if len(r.Files) != 0 {
		id = r.Files[0].Id
	}

	return id, nil
}

// GetItemsForFolder returns a map of [name]= File (partial response of id, name, mimeType, modifiedTime, parents, fileCapabilities)
// for all files (inclduding folders) at the top level for a specific folder id, if no files are found, returns an empty list
// returns nil, err for and issue with the request
func (srv DriveService) GetItemsForFolder(folderID string) (map[string]*drive.File, error) {

	fieldString := "nextPageToken, files(id, name, mimeType, modifiedTime, parents, capabilities, md5Checksum)"
	qString := fmt.Sprintf("'%s' in parents and trashed = false", folderID)

	r, err := srv.service.Files.List().PageSize(50).Fields(googleapi.Field(fieldString)).Q(qString).Do()

	if err != nil {
		return nil, err
	}

	// create map and has with name so that differences in file content can be found easily
	files := make(map[string]*drive.File)
	for _, i := range r.Files {
		files[i.Name] = i
	}

	nextPageToken := r.NextPageToken
	for nextPageToken != "" {
		r, err := srv.service.Files.List().PageSize(30).Fields(googleapi.Field(fieldString)).Q(qString).PageToken(nextPageToken).Do()
		if err != nil {
			return nil, err
		}

		nextPageToken = r.NextPageToken
		for _, i := range r.Files {
			files[i.Name] = i
		}
	}
	return files, nil
}

// DownloadFile downloads an object from drive with the given name
// function closes the body
func (srv DriveService) DownloadFile(fileID string, dest string) error {

	r, err := srv.service.Files.Get(fileID).AcknowledgeAbuse(true).Download()
	if err != nil {
		if strings.Contains(err.Error(), "invalidAbuseAcknowledgment") {
			// there is a case where drive will throw an error for no reason, this api call is suppoed to be fixed
			// but sometimes gives awn invalidAbuseAcknowledgment when that error shouldn't be used anymore
			r, err = srv.service.Files.Get(fileID).Download()
		}
		if err != nil {
			return err
		}
	}

	// if there is an error, the request will automatically close the body
	defer r.Body.Close()
	buf := new(bytes.Buffer)
	buf.ReadFrom(r.Body)
	err = ioutil.WriteFile(dest, buf.Bytes(), os.FileMode(0644))
	if err != nil {
		return err
	}
	return nil
}

// UploadFile uploads the contents of a file from a path matching the basename and with the specified parent id
func (srv DriveService) UploadFile(content io.Reader, fileName string, parentID string) error {

	fileMetadata := &drive.File{
		Name:    fileName,
		Parents: []string{parentID},
	}

	_, err := srv.service.Files.Create(fileMetadata).Media(content).Do()
	return err
}

// CreateFolder creates a folder tied to a specific parent Id
func (srv DriveService) CreateFolder(folderName string, parentID string) error {

	fileMetaData := &drive.File{
		Name:     folderName,
		Parents:  []string{parentID},
		MimeType: "application/vnd.google-apps.folder",
	}

	_, err := srv.service.Files.Create(fileMetaData).Do()
	return err
}

// UpdateFile updates the metadata and content of a file that already exists in drive, used for a file that has been locally modified
// here, we are attaching a modified time so that we don't keep downloading the same file
func (srv DriveService) UpdateFile(content io.Reader, fileID string) error {

	fileMetadata := &drive.File{}

	_, err := srv.service.Files.Update(fileID, fileMetadata).Media(content).Do()
	return err
}

// SeparateFilesAndFolders splits a slice of files into mime types of files and folders respectively
func SeparateFilesAndFolders(items map[string]*drive.File) (map[string]*drive.File, map[string]*drive.File) {

	files := make(map[string]*drive.File)
	folders := make(map[string]*drive.File)

	// only two types, either a file or a folder
	for name, file := range items {
		if file.MimeType == "application/vnd.google-apps.folder" {
			folders[name] = file
		} else {
			files[name] = file
		}
	}
	return files, folders
}

// internal

// retrieves a token, saves the token and returns a guaranteed client
func getClient(config *oauth2.Config) *http.Client {
	tokenFile := GetTokenSaveLocation()
	tok, err := tokenFromFile(tokenFile)

	if err != nil {
		tok = getTokenFromWeb(config)
		saveToken(tokenFile, tok)
	}
	return config.Client(context.Background(), tok)
}

// Request a token from the web, then returns the retrieved token.
func getTokenFromWeb(config *oauth2.Config) *oauth2.Token {
	authURL := config.AuthCodeURL("state-token", oauth2.AccessTypeOffline)
	fmt.Printf("Go to the following link in your browser then type the "+
		"authorization code: \n%v\n\n", authURL)

	var authCode string

	fmt.Printf("Auth code: ")
	if _, err := fmt.Scan(&authCode); err != nil {
		log.Fatalf("Unable to read authorization code %v", err)
	}

	tok, err := config.Exchange(oauth2.NoContext, authCode)
	if err != nil {
		log.Fatalf("Unable to retrieve token from web %v", err)
	}
	return tok
}

// loads a token from a local file
func tokenFromFile(file string) (*oauth2.Token, error) {
	f, err := os.Open(file)
	defer f.Close()
	if err != nil {
		return nil, err
	}

	tok := &oauth2.Token{}
	err = json.NewDecoder(f).Decode(tok)
	return tok, err
}

// saves a token to a local file path
func saveToken(path string, token *oauth2.Token) {
	log.Printf("Saving token to %s\n", path)
	f, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0600)
	defer f.Close()

	if err != nil {
		log.Fatalf("Unable to cache oath token %v", err)
	}
	json.NewEncoder(f).Encode(token)
}
