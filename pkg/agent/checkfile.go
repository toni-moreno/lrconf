package main

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	//	"path/filepath"
	"crypto/md5"
	"crypto/sha1"
	"crypto/sha256"
	"crypto/sha512"
	"errors"
	"io"
	"io/ioutil"
	"mime/multipart"
	"net/http"
	"strconv"
	"time"
	//"net/url"
	"os/user"
	"path/filepath"
)

/*
[[checkgroup]]
check_cmd = "/usr/sbin/httpd -t -c {{.src}}"
reload_cmd = "/usr/sbin/service apache reload"
upload_file_after_cmd= "/opt/collectd/logs/collectd.log"
[[checkgroup.file]]
path = "/opt/collectd/plugin_apache.conf"
sum = "5HGWGGEGWHHE55W5W5"
owner = "apache"
mode = "0644"
*/

/*FileCfg type to handle files*/
type FileCfg struct {
	Path        string `toml:"path"`
	Sum         string `toml:"sum"`
	SumType     string `toml:"sum"`
	Owner       string `toml:"file_owner"`
	Mode        string `toml:"file_mode"`
	CheckAction string `toml:"check_action"`
}

/*CheckGroupConfig  for group files*/
type CheckGroupConfig struct {
	CheckID            string
	CheckCmd           string `toml:"check_cmd"`
	ReloadCmd          string `toml:"reload_cmd"`
	UploadFileAfterCmd string `toml:"upload_file_after_cmd"`
	GroupOwner         string `toml:"groupowner"`
	GroupMode          string `toml:"groupmode"`
	File               []*FileCfg
}

//private basic methods for upload/download files from Server

//https://matt.aimonetti.net/posts/2013/07/01/golang-multipart-file-upload-example/

func newfileUploadRequest(uri string, params map[string]string, paramName, path string) (*http.Request, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile(paramName, filepath.Base(path))
	if err != nil {
		return nil, err
	}
	_, err = io.Copy(part, file)

	for key, val := range params {
		_ = writer.WriteField(key, val)
	}
	err = writer.Close()
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", uri, body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	return req, err
}

func uploadFile(rawURL string, path string, extraParams map[string]string) error {
	log.Debugf("Doing uploading file... for file: %s", path)

	request, err := newfileUploadRequest(rawURL, extraParams, "file", path)
	if err != nil {
		log.Error(err)
		return err
	}
	client := &http.Client{}
	resp, err := client.Do(request)
	if err != nil {
		log.Error(err)
		return err
	}

	body := &bytes.Buffer{}
	_, err = body.ReadFrom(resp.Body)
	if err != nil {
		log.Error(err)
	}
	resp.Body.Close()
	log.Debugf("UPLOAD STATUS CODE: %v", resp.StatusCode)
	log.Debugf("UPLOAD RESP HEADER: %v", resp.Header)
	log.Debugf("UPLOAD BODY: %v", body)

	return nil
}

//https://www.socketloop.com/tutorials/golang-download-file-example
func downloadFile(rawURL string, dest string) error {
	log.Debugf("Doing downloading file  %s... from URL : %s", dest, rawURL)
	//dirname := filepath.Dir(dest)

	//tmpfile := filepath.Join(os.TempDir(), basename)

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
		return err
	}
	defer resp.Body.Close()

	log.Infof("Download Status %s", resp.Status)
	if resp.Status != "200 OK" {
		log.Errorf("Error on download File %s [ERROR: %s]", rawURL, resp.Status)
		return errors.New("Error on download " + resp.Status)
	}

	file, err := os.Create(dest)
	log.Debugf("New File created at: %s", dest)
	if err != nil {
		fmt.Println(err)
		panic(err)
	}
	defer file.Close()

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

//InitCheckGroup check if mínimal data is set..
func (g *CheckGroupConfig) InitCheckGroup() (bool, error) {
	//minimal data should be FilePath and FileSum Default Ownser will be current execution user and default filemode will be 0644 for Linux.
	//File is not needed existing in that case in the next ckeck iteration will be dowloaded
	// Init Group
	u, _ := user.Current()

	if len(g.GroupOwner) == 0 {
		g.GroupOwner = u.Username
	}
	if len(g.GroupMode) == 0 {
		g.GroupMode = "0755"
	}
	//Init files
	for _, f := range g.File {

		if len(f.Path) == 0 {
			return false, errors.New("Needed Path parameter not found")
		}
		if len(f.Sum) == 0 {
			return false, errors.New("Needed FileSum parameter not found")
		}
		if len(f.SumType) == 0 {
			f.SumType = "MD5"
		}
		if len(f.Owner) == 0 {
			f.Owner = g.GroupOwner
		}
		if len(f.Mode) == 0 {
			f.Mode = g.GroupMode
		}
		if len(f.CheckAction) == 0 {
			f.CheckAction = "change"
		}
	}

	return true, nil
}

//Backup  for saving old versio config files
func (f *FileCfg) Backup() error {
	timename := time.Now().Format("2006-01-02_15:04:05")
	backupPath := f.Path + "." + timename
	log.Debugf("Doing backup... for file: %s to : %s", f.Path, backupPath)
	in, err := os.Open(f.Path)
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
func (f *FileCfg) Exist() (bool, error) {
	log.Debugf("Checking if file: %s exist...", f.Path)
	if _, err := os.Stat(f.Path); os.IsNotExist(err) {
		return false, err
	}
	return true, nil
}

//IsModified to check if current SUM is
func (f *FileCfg) IsModified() (string, bool) {
	log.Infof("Check File modification... for file: %s", f.Path)
	//at this point file should exist
	data, err := ioutil.ReadFile(f.Path)
	log.Debug(string(data))
	if err != nil {
		return f.Path, false
	}

	var currentsumStr string
	switch f.SumType {
	case "MD5":
		currentsum := md5.Sum(data)
		currentsumStr = fmt.Sprintf("%x", currentsum)
	case "SHA1":
		currentsum := sha1.Sum(data)
		currentsumStr = fmt.Sprintf("%x", currentsum)
	case "SHA256":
		currentsum := sha256.Sum256(data)
		currentsumStr = fmt.Sprintf("%x", currentsum)
	case "SHA512":
		currentsum := sha512.Sum512(data)
		currentsumStr = fmt.Sprintf("%x", currentsum)
	}

	log.Debugf("CURRENT SUM: %s", currentsumStr)
	if currentsumStr != f.Sum {
		return currentsumStr, true
	}
	return f.Sum, false
}

//DownloadNew to download the new version of this file
func (f *FileCfg) DownloadNew(nodeid string, groupid string, server ServerConfig) error {
	log.Debugf("Download new file version from server... for file: %s", f.Path)
	basename := filepath.Base(f.Path)
	rawURL := "http://" + server.CentralConfigServer + ":" + strconv.Itoa(server.CentralConfigPort) + "/nodes/" + nodeid + "/" + groupid + "/" + basename
	downloadFile(rawURL, f.Path)
	return nil
}

//UploadLog upload
func (g *CheckGroupConfig) UploadLog(nodeid string, server ServerConfig) error {
	log.Debugf("Uploading log file to server... related to Group: %s", g.CheckID)
	if len(g.UploadFileAfterCmd) == 0 {
		return nil
	}

	extraParams := map[string]string{
		"nodeid":  nodeid,
		"checkid": g.CheckID,
	}

	rawURL := "http://" + server.CentralConfigServer + ":" + strconv.Itoa(server.CentralConfigPort) + "/upload/"

	uploadFile(rawURL, g.UploadFileAfterCmd, extraParams)
	return nil
}

//ExecCheck upload
func (g *CheckGroupConfig) ExecCheck() (bool, error) {
	log.Debugf("Executing Check Command related to Group: %s", g.CheckID)
	out, err := execCmd(g.CheckCmd)
	log.Infof("CMD OUT: %s", out)
	if err != nil {
		log.Error(err)
		return false, err
	}
	return true, nil
}

//ExecReload upload
func (g *CheckGroupConfig) ExecReload() (bool, error) {
	log.Debugf("Executing Reload Command related to Group: %s", g.CheckID)
	out, err := execCmd(g.ReloadCmd)
	log.Infof("CMD OUT: %s", out)
	if err != nil {
		log.Error(err)
		return false, err
	}
	return true, nil
}
