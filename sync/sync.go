package sync

import (
	"crypto/md5"
	"fmt"
	"io"
	"log"
	"os"
	"path"
	"time"

	drive "google.golang.org/api/drive/v3"
)

//TimeFormat is the format googles uses for their timestamps, we need to use this for all out timestamps
const TimeFormat = time.RFC3339

// DoRecursiveSync completes a recursive sync of a given local directory
// requires the full directory path to work properly
// send the errors to a channel and terminate, we dont care about creating a buffer since one error means terminate
func DoRecursiveSync(srv *DriveService, parentDir string, parentID string, ch chan<- error) {

	sync(srv, parentDir, parentID, ch)
	close(ch)
}

// functions that does the actual syncing, needs to be wrapped so that we can indicate that the
// function is done
func sync(srv *DriveService, parentDir string, parentID string, ch chan<- error) {

	// make get files call in one place to avoid rest calls, expensive
	items, err := srv.GetItemsForFolder(parentID)
	if err != nil {
		log.Printf("%v", err)
		ch <- err
		return
	}
	driveFiles, driveFolders := SeparateFilesAndFolders(items)

	err = syncFiles(srv, parentDir, parentID, driveFiles)
	if err != nil {
		log.Printf("%v", err)
		ch <- err
		return
	}

	err = syncFolders(srv, parentDir, parentID, driveFolders)
	if err != nil {
		log.Printf("%v", err)
		ch <- err
		return
	}

	// get all the folders again in case we created some more
	items, err = srv.GetItemsForFolder(parentID)
	if err != nil {
		log.Printf("%v", err)
		ch <- err
		return
	}
	_, driveFolders = SeparateFilesAndFolders(items)

	for _, folder := range driveFolders {
		sync(srv, path.Join(parentDir, folder.Name), folder.Id, ch)
	}
}

// syncFolders passes over a directory level and syncs all of the files only on that level
// directory must be the full path to the folder
func syncFolders(srv *DriveService, parentDir string, parentID string, driveFolders map[string]*drive.File) error {

	localFolders, err := GetTopLevelFolders(parentDir)
	if err != nil {
		return err
	}

	// same as syncFiles, we keep track of the files we have prossed by setting their hash to true
	processedFolders := make(map[string]bool)

	for _, file := range localFolders {
		name := file.Name()
		if _, ok := driveFolders[name]; !ok {
			err = srv.CreateFolder(name, parentID)
			log.Printf("Creating folder %s in drive\n", name)
			if err != nil {
				return err
			}
		} else {
			processedFolders[name] = true
		}
	}

	for name := range driveFolders {
		if !processedFolders[name] {
			log.Printf("Creating folder %s locally\n", name)
			os.Mkdir(path.Join(parentDir, name), os.ModePerm)
		}
	}

	return nil
}

// syncFiles passes over a directory level and syncs all of the files only on that level
// directory must be the full path to the folder
func syncFiles(srv *DriveService, parentDir string, folderID string, driveFiles map[string]*drive.File) error {

	localFiles, err := GetTopLevelFiles(parentDir)
	if err != nil {
		return err
	}

	// keep track of the drive files we have processed, just care about hash key, not value
	// since the default value is true, we can just get the value from the hash to check if
	// the file was processed
	processedFiles := make(map[string]bool)

	// check if all of the local files are in drive
	for _, file := range localFiles {
		name := file.Name()

		// if the file is not in drive, upload it
		if _, ok := driveFiles[name]; !ok {

			log.Printf("Uploading %s file to drive\n", name)
			r, err := os.Open(path.Join(parentDir, name))
			if err != nil {
				return err
			}
			err = srv.UploadFile(r, name, folderID)
			if err != nil {
				return err
			}
		} else {
			// the file exists in both, do a diff on the check sum if we can download the file
			driveFile := driveFiles[name]
			localMD5, err := computeHashString(path.Join(parentDir, name))
			if err != nil {
				return err
			}

			if driveFile.Capabilities.CanDownload && localMD5 != driveFiles[name].Md5Checksum {

				// files are not equal, take the newer one
				// drive times are in UTC and to the millisecond, local files are not
				localTime := file.ModTime().UTC().Round(time.Second)
				driveTime, err := time.Parse(TimeFormat, driveFile.ModifiedTime)
				if err != nil {
					return err
				}
				driveTime.Round(time.Second)

				if localTime.Before(driveTime) {
					// the drive file is newer, download it
					log.Printf("%s is newer in drive, dowloading...", name)
					err = srv.DownloadFile(driveFile.Id, path.Join(parentDir, name))
					if err != nil {
						return err
					}
				} else {
					// the local file is newer or we can't differentiate the time, upload it
					log.Printf("%s is newer locally, uploading...", name)
					f, err := os.Open(path.Join(parentDir, name))
					if err != nil {
						return err
					}
					defer f.Close()
					err = srv.UpdateFile(f, driveFile.Id)
				}
			} else if !driveFile.Capabilities.CanDownload {
				log.Printf("%s cannot be downloaded, skipping...", name)
			}
			// going to not log files that are the same, muddles logfile

			processedFiles[name] = true
		}

	}

	// all local files are synced, handle the ones in drive but not locally
	for name, file := range driveFiles {
		// checksum is only available for files with binary content, we arn't going to download sheets, docs, etc.
		if file.Capabilities.CanDownload && file.Md5Checksum != "" && !processedFiles[name] {
			log.Printf("Downloading %s from drive\n", name)
			err = srv.DownloadFile(file.Id, path.Join(parentDir, name))
			if err != nil {
				return err
			}
		}
	}

	// all files at this level should now be synced
	return nil
}

// computes a hash on a file, give the entire filepath
func computeHashString(filePath string) (string, error) {
	// compute the hash for the local file to compare
	f, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer f.Close()

	h := md5.New()
	_, err = io.Copy(h, f)
	if err != nil {
		return "", err
	}
	s := fmt.Sprintf("%x", h.Sum(nil))
	return s, nil
}
