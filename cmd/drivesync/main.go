package main

import (
	"log"

	"github.com/nparthas/drivesync/sync"
)

func main() {

	config, err := sync.SetUp()
	if err != nil {
		log.Fatalf("The directory to upload must be set from the command line")
	}

	// TODO:: error handle for outdated token

	srv, err := sync.GetService(config.CredentialsFilepath)
	if err != nil {
		log.Fatalf("Unable to retrieve drive client %v", err)
	}

	// get the parent folder id
	parentID, err := srv.GetFolderID(config.DriveFolder)
	if err != nil {
		log.Fatalf("Unable to get list of files %v", err)
	}
	log.Printf("top-level parent folder name, id: %s %s\n", config.DriveFolder, parentID)

	ch := make(chan error)

	for ok := true; ok; ok = config.Once {
		go sync.DoRecursiveSync(srv, config.ParentFolder, parentID, ch)
		// only errors get sent back, fail on the first one
		err = <-ch
		if err != nil {
			log.Fatalf("Could not sync dir %v", err)
		}
		// put some newline so it's easier to read the log file
		log.Printf("finished sync...\n\n\n\n\n\n\n")
	}
}
