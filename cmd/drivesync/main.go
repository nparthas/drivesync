package main

import (
	"fmt"
	"io/ioutil"
	"log"

	"github.com/nparthas/drivesync/sync"

	"golang.org/x/oauth2/google"
	"google.golang.org/api/drive/v3"
)

func main() {

	b, err := ioutil.ReadFile("credentials.json")

	if err != nil {
		log.Fatalf("Unable to read credentails file %v", err)
	}

	config, err := google.ConfigFromJSON(b, drive.DriveMetadataReadonlyScope)
	if err != nil {
		log.Fatalf("Unable to parse client secret file %v", err)
	}

	client := sync.GetClient(config)

	srv, err := drive.New(client)
	if err != nil {
		log.Fatalf("Unable to retrieve drive client %v", err)
	}

	// r, err := srv.Files.List().PageSize(10).Fields("nextPageToken, files(id, name)").Do()
	r, err := srv.Files.List().PageSize(50).Fields("nextPageToken, files(id,name)").Q("mimeType = 'application/vnd.google-apps.folder'").Do()
	// r := srv.Files.EmptyTrash()

	if err != nil {
		log.Fatalf("Unable to get list of file %v", err)
	}

	fmt.Printf("\n\n%v\n\n\n", r)

	fmt.Println("Files")
	if len(r.Files) == 0 {
		fmt.Println("no files")
	} else {
		for _, i := range r.Files {
			fmt.Printf("%s (%s)\n", i.Name, i.Id)
		}
	}
}
