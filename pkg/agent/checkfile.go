package main

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
	//	"path/filepath"
	"crypto/md5"
	"errors"
	"io"
	"io/ioutil"
	"net/http"
	"strconv"
	"time"
	//"net/url"
	"path/filepath"
)

/*
[checkfile]
file_path = "/opt/collectd/plugin_apache.conf"
file_sum = "5HGWGGEGWHHE55W5W5"
file_owner = "apache"
file_mode = "0644"
check_cmd = "/usr/sbin/httpd -t -c {{.src}}"
reload_cmd = "/usr/sbin/service apache reload"
upload_file_after_cmd= "/opt/collectd/logs/collectd.log"
*/

/*CheckFileConfig  is type for file checking*/
type CheckFileConfig struct {
	FilePath           string `toml:"file_path"`
	FileSum            string `toml:"file_sum"`
	FileOwner          string `toml:"file_owner"`
	FileMode           string `toml:"file_mode"`
	CheckCmd           string `toml:"check_cmd"`
	ReloadCmd          string `toml:"reload_cmd"`
	UploadFileAfterCmd string `toml:"upload_file_after_cmd"`
}

//private basic methods for upload/download files from Server

//https://matt.aimonetti.net/posts/2013/07/01/golang-multipart-file-upload-example/
func (f *CheckFileConfig) uploadFile() error {
	log.Debugf("Doing uploading file... for file: %s", f.FilePath)
	return nil
}

//https://www.socketloop.com/tutorials/golang-download-file-example
func downloadFile(rawURL string, dest string) error {
	log.Debugf("Doing downloading file  %s... from URL : %s", dest, rawURL)
	//dirname := filepath.Dir(dest)

	//tmpfile := filepath.Join(os.TempDir(), basename)
	file, err := os.Create(dest)
	log.Debugf("New File created at: %s", dest)
	if err != nil {
		fmt.Println(err)
		panic(err)
	}
	defer file.Close()

	//http redirect detection
	check := http.Client{
		CheckRedirect: func(r *http.Request, via []*http.Request) error {
			r.URL.Opaque = r.URL.Path
			return nil
		},
	}
	//get file
	resp, err := check.Get(rawURL)
	if err != nil {
		log.Errorln(err)
	}
	defer resp.Body.Close()
	log.Infof("Download Status %s", resp.Status)
	if resp.Status != "200 OK" {
		log.Errorf("Error on download File %s [ERROR: %s]", rawURL, resp.Status)
		return errors.New("Error on download " + resp.Status)
	}
	size, err := io.Copy(file, resp.Body)

	if err != nil {
		log.Errorf("Error on Copy file to %s", dest)
	}
	basename := filepath.Base(dest)
	log.Infof("%s with %v bytes downloaded", basename, size)
	return nil
}

func execCmd(cmd string) (string, error) {
	var c *exec.Cmd
	switch runtime.GOOS {
	case "linux":
		c = exec.Command("bash", "-c", cmd)
	case "windows":
		c = exec.Command("cmd", "/C", cmd)
	}
	out, err := c.Output()
	return string(out), err
}

//Public exported Methods

//InitCheck check if mínimal data is set..
func (f *CheckFileConfig) InitCheck() (bool, error) {
	//minimal data should be FilePath and FileSum Default Ownser will be current execution user and default filemode will be 0644 for Linux.
	//File is not needed existing in that case in the next ckeck iteration will be dowloaded
	if len(f.FilePath) == 0 {
		return false, errors.New("Needed Filepath parameter not found")
	}

	if len(f.FileSum) == 0 {
		return false, errors.New("Needed FileSum parameter not found")
	}

	return true, nil
}

//Backup  for saving old versio config files
func (f *CheckFileConfig) Backup() error {
	timename := time.Now().Format("2006-01-02_15:04:05")
	backupPath := f.FilePath + "." + timename
	log.Debugf("Doing backup... for file: %s to : %s", f.FilePath, backupPath)
	in, err := os.Open(f.FilePath)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.Create(backupPath)
	if err != nil {
		return err
	}
	defer func() {
		cerr := out.Close()
		if err == nil {
			err = cerr
		}
	}()
	if _, err = io.Copy(out, in); err != nil {
		return err
	}
	err = out.Sync()
	return nil
}

//Exist to check if current SUM is
func (f *CheckFileConfig) Exist() (bool, error) {
	log.Debugf("Checking if file: %s exist...", f.FilePath)
	if _, err := os.Stat(f.FilePath); os.IsNotExist(err) {
		return false, err
	}
	return true, nil
}

//IsModified to check if current SUM is
func (f *CheckFileConfig) IsModified() (string, bool) {
	log.Infof("Check File modification... for file: %s", f.FilePath)
	//at this point file should exist
	data, err := ioutil.ReadFile(f.FilePath)
	log.Debug(string(data))
	if err != nil {
		return f.FilePath, false
	}
	currentsum := md5.Sum(data)
	currentsumStr := fmt.Sprintf("%x", currentsum)
	log.Debugf("CURRENT SUM: %s", currentsumStr)
	if currentsumStr != f.FileSum {
		return currentsumStr, true
	}
	return f.FileSum, false
}

//DownloadNew to download the new version of this file
func (f *CheckFileConfig) DownloadNew(nodeid string, server ServerConfig) error {
	log.Debugf("Download new file version from server... for file: %s", f.FilePath)
	basename := filepath.Base(f.FilePath)
	rawURL := "http://" + server.CentralConfigServer + ":" + strconv.Itoa(server.CentralConfigPort) + "/" + nodeid + "/" + basename
	downloadFile(rawURL, f.FilePath)
	return nil
}

//UploadLog upload
func (f *CheckFileConfig) UploadLog(nodeid string) error {
	log.Debugf("Uploading log file to server... related to file: %s", f.FilePath)
	return nil
}

//ExecCheck upload
func (f *CheckFileConfig) ExecCheck() (bool, error) {
	log.Debugf("Executing Check Command related to file: %s", f.FilePath)
	out, err := execCmd(f.CheckCmd)
	log.Infof("CMD OUT: %s", out)
	if err != nil {
		log.Error(err)
		return false, err
	}
	return true, nil
}

//ExecReload upload
func (f *CheckFileConfig) ExecReload() (bool, error) {
	log.Debugf("Executing Reload Command related to file: %s", f.FilePath)
	out, err := execCmd(f.ReloadCmd)
	log.Infof("CMD OUT: %s", out)
	if err != nil {
		log.Error(err)
		return false, err
	}
	return true, nil
}
