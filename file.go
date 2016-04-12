package iris

import (
	"archive/zip"
	"fmt"
	"io"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// directoryExists returns true if a directory(or file) exists, otherwise false
func directoryExists(dir string) bool {
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		return false
	}
	return true
}

// downloadZip downloads a zip file returns the downloaded filename and an error.
//
// An indicator is always shown up to the terminal, so the user will know if (a plugin) try to download something
func downloadZip(zipURL string, newDir string) (string, error) {
	var err error
	var size int64
	finish := false

	go func() {
		i := 0
		print("\n|")
		print("_")
		print("|")

	printer:
		{
			i++

			print("\010\010-")
			time.Sleep(time.Second / 2)
			print("\010\\")
			time.Sleep(time.Second / 2)
			print("\010|")
			time.Sleep(time.Second / 2)
			print("\010/")
			time.Sleep(time.Second / 2)
			print("\010-")
			time.Sleep(time.Second / 2)
			print("|")
			if finish {
				goto ok
			}
			goto printer
		}

	ok:
	}()

	os.MkdirAll(newDir, os.ModeDir)
	tokens := strings.Split(zipURL, "/")
	fileName := newDir + tokens[len(tokens)-1]
	if !strings.HasSuffix(fileName, ".zip") {
		err = fmt.Errorf("Error while creating %s ,is not a zip", fileName)
		return "", err
	}

	output, err := os.Create(fileName)
	if err != nil {
		fmt.Println("Error while creating", fileName, "-", err)
		return "", nil
	}
	defer output.Close()
	response, err := http.Get(zipURL)
	if err != nil {
		fmt.Println("Error while downloading", zipURL, "-", err)
		return "", nil
	}
	defer response.Body.Close()

	size, err = io.Copy(output, response.Body)
	if err != nil {
		fmt.Println("Error while downloading", zipURL, "-", err)
		return "", nil
	}
	finish = true
	println("\010\010\010\010\010\010OK ", size, " bytes downloaded")
	return fileName, nil

}

// unzip extracts a zipped file to the target location
//
// it removes the zipped file after succesfuly completion
// returns a string with the path of the created folder (if any) and an error (if any)
func unzip(archive string, target string) (string, error) {
	reader, err := zip.OpenReader(archive)
	if err != nil {
		return "", err
	}

	if err := os.MkdirAll(target, 0755); err != nil {
		return "", err
	}
	createdFolder := ""
	for _, file := range reader.File {
		path := filepath.Join(target, file.Name)
		if file.FileInfo().IsDir() {
			os.MkdirAll(path, file.Mode())
			if createdFolder == "" {
				// this is the new directory that zip has
				createdFolder = path
			}
			continue
		}

		fileReader, err := file.Open()
		if err != nil {
			return "", err
		}
		defer fileReader.Close()

		targetFile, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, file.Mode())
		if err != nil {
			return "", err
		}
		defer targetFile.Close()

		if _, err := io.Copy(targetFile, fileReader); err != nil {
			return "", err
		}

	}

	reader.Close()
	return createdFolder, nil
}

// removeFile removes a file and returns an error, if any
func removeFile(filePath string) error {
	return os.Remove(filePath)
}

// install is just the flow of: downloadZip -> unzip -> removeFile(zippedFile)
// accepts 2 parameters
//
// first parameter is the remote url file zip
// second parameter is the target directory
// returns a string(installedDirectory) and an error
//
// (string) installedDirectory is the directory which the zip file had, this is the real installation path, you don't need to know what it's because these things maybe change to the future let's keep it to return the correct path.
// the installedDirectory is not empty when the installation is succed, the targetDirectory is not already exists and no error happens
// the installedDirectory is empty when the installation is already done by previous time or an error happens
func install(remoteFileZip string, targetDirectory string) (installedDirectory string, err error) {
	var zipFile string

	zipFile, err = downloadZip(remoteFileZip, targetDirectory)
	if err == nil {
		installedDirectory, err = unzip(zipFile, targetDirectory)
		if err == nil {
			installedDirectory += string(os.PathSeparator)
			removeFile(zipFile)
		}
	}
	return
}

// TypeByExtension returns the MIME type associated with the file extension ext.
// The extension ext should begin with a leading dot, as in ".html".
// When ext has no associated type, TypeByExtension returns "".
//
// Extensions are looked up first case-sensitively, then case-insensitively.
//
// The built-in table is small but on unix it is augmented by the local
// system's mime.types file(s) if available under one or more of these
// names:
//
//   /etc/mime.types
//   /etc/apache2/mime.types
//   /etc/apache/mime.types
//
// On Windows, MIME types are extracted from the registry.
//
// Text types have the charset parameter set to "utf-8" by default.
func TypeByExtension(fullfilename string) (t string) {
	if t = mime.TypeByExtension(filepath.Ext(fullfilename)); t == "" {
		t = ContentBINARY
	}
	return
}

func getParrentDir(targetDirectory string) string {
	lastSlashIndex := strings.LastIndexByte(targetDirectory, os.PathSeparator)
	//check if the slash is at the end , if yes then re- check without the last slash, we don't want /path/to/ , we want /path/to in order to get the /path/ which is the parent directory of the /path/to
	if lastSlashIndex == len(targetDirectory)-1 {
		lastSlashIndex = strings.LastIndexByte(targetDirectory[0:lastSlashIndex], os.PathSeparator)
	}

	parentDirectory := targetDirectory[0:lastSlashIndex]
	return parentDirectory
}