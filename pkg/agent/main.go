package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/Sirupsen/logrus"
	"github.com/spf13/viper"
)

// GeneralConfig has miscelaneous configuration options
type GeneralConfig struct {
	LogDir   string `toml:"log_dir"`
	LogLevel string `toml:"log_level"`
}

/*ServerConfig  remote server config file*/
type ServerConfig struct {
	CentralConfigServer string `toml:"central_config_server"`
	CentralConfigPort   int    `toml:"central_config_port"`
	ReloadConfig        int    `toml:"reload_config"`
}

//Config has all configurations
type Config struct {
	NodeID  string `toml:"nodeid"`
	tmpdir  string
	General GeneralConfig
	Server  ServerConfig
	//CheckFiles map[string]*CheckFileConfig
	CheckGroup []*CheckGroupConfig
}

var (
	version    string
	commit     string
	branch     string
	buildstamp string
)

var (
	log        = logrus.New()
	getversion bool
	appdir     = os.Getenv("PWD")
	logDir     = filepath.Join(appdir, "log")
	confDir    = filepath.Join(appdir, "conf")
	configFile = filepath.Join(confDir, "lrconf-agent.toml")

	cfg *Config
)

func fatal(v ...interface{}) {
	log.Fatalln(v...)
}

func flags() *flag.FlagSet {
	var f flag.FlagSet
	f.BoolVar(&getversion, "version", getversion, "display de version")
	f.StringVar(&configFile, "config", configFile, "config file")
	f.StringVar(&logDir, "logs", logDir, "log directory")
	f.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage of %s:\n", os.Args[0])
		f.VisitAll(func(flag *flag.Flag) {
			format := "%10s: %s\n"
			fmt.Fprintf(os.Stderr, format, "-"+flag.Name, flag.Usage)
		})
		fmt.Fprintf(os.Stderr, "\nAll settings can be set in config file: %s\n", configFile)
		os.Exit(1)

	}
	return &f
}

func init() {

	//SET Log format
	customFormatter := new(logrus.TextFormatter)
	customFormatter.TimestampFormat = "2006-01-02 15:04:05"
	log.Formatter = customFormatter
	customFormatter.FullTimestamp = true

	// parse first time to see if config file is being specified
	f := flags()
	f.Parse(os.Args[1:])

	if getversion {
		fmt.Printf("lrconf-agent v%s (git: %s ) built at [%s]\n", version, commit, branch)
		os.Exit(0)
	}
	//
	log.Printf("set Default directories : \n   - Exec: %s\n   - Config: %s\n   -Logs: %s\n", appdir, confDir, logDir)
	// now load up config settings
	viper.Set("Verbose", true)
	viper.Set("LogFile", "./log/viper.log")

	if _, err := os.Stat(configFile); err == nil {
		log.Info("no config file set")
		viper.SetConfigFile(configFile)
	} else {
		log.Info("set default config files")

		viper.SetConfigName("lrconf-agent")
		tmpdir := filepath.Join(os.TempDir(), "lrconf-agent")
		//first dir to seach for is the temporal runtime path if have been previously downloaded and after stopped
		viper.AddConfigPath(tmpdir)
		switch runtime.GOOS {
		case "linux":
			viper.AddConfigPath("/opt/lrconf/conf/")
			viper.AddConfigPath("./conf/")
			viper.AddConfigPath(".")
		case "windows":
			viper.AddConfigPath("C:\\Program Files\\lrconf-agent\\conf\\")
			viper.AddConfigPath(".")
		}

	}
	err := viper.ReadInConfig()
	if err != nil {
		log.Errorf("Fatal error config file: %s ", err)
		os.Exit(1)
	}
	//Allocating Config struct
	cfg = new(Config)
	cfg.InitConfig()
	err = viper.Unmarshal(cfg)
	if err != nil {
		log.Errorf("unable to decode into struct, %s ", err)
		os.Exit(1)
	}
	cfg.EndConfig()

	//LOG SETTINGS

	if len(cfg.General.LogDir) > 0 {
		logDir = cfg.General.LogDir
	}
	if len(cfg.General.LogLevel) > 0 {
		l, _ := logrus.ParseLevel(cfg.General.LogLevel)
		log.Level = l
	}
	log.Infof("%+v", cfg)

	// re-read cmd line args to override as indicated
	f = flags()
	f.Parse(os.Args[1:])
	os.Mkdir(logDir, 0755)

	//CHECK IF FILES ARE OK

	for i, g := range cfg.CheckGroup {
		if ok, err := g.InitCheckGroup(); ok != true {
			log.Warningf("Error in config  group %s has errors: %s: ", i, err)
			cfg.CheckGroup = append(cfg.CheckGroup[:i], cfg.CheckGroup[i+1:]...)
		}
		log.Infof("Group Check OK : %s : %s", i, g.CheckID)
	}

}

//InitConfig default values
func (c *Config) InitConfig() {
	//CHECK IF NODEID
	if len(c.NodeID) == 0 {
		name, _ := os.Hostname()
		c.NodeID = strings.ToLower(name)
		log.Warnf("NODEID not set in initial configuration, has been set as the hostname: %s", c.NodeID)
	} else {
		log.Infof("NODEID set to: %s", c.NodeID)
	}
	//Creating Temporal dir to Download remote config
	c.tmpdir = filepath.Join(os.TempDir(), "lrconf-agent")
	os.Mkdir(cfg.tmpdir, 0755)
}

//EndConfig default values
func (c *Config) EndConfig() {
	//set checkid for each check

}

//DownloadNew to download the new version of this file
func (c *Config) downloadMainConf() (string, error) {
	log.Debugf("Download new config file from server..")
	basename := "lrconf-agent.toml"
	rawURL := "http://" + c.Server.CentralConfigServer + ":" + strconv.Itoa(c.Server.CentralConfigPort) + "/nodes/" + c.NodeID + "/" + basename
	newconf := filepath.Join(c.tmpdir, basename)
	err := downloadFile(rawURL, newconf)
	return newconf, err
}

/*CheckFiles is the main loop to check configuration files */
func CheckFiles(wg *sync.WaitGroup, cfg *Config) {
	//func CheckFiles(wg *sync.WaitGroup, CheckFiles []*CheckFileConfig, Freq int) {
	defer wg.Done()
	Freq := cfg.Server.ReloadConfig
	log.Debugf("init check processes with: %d seconds", Freq)
	s := time.Tick(time.Duration(Freq) * time.Second)
	for {
		log.Debugf("new interation %s", time.Now().String())
		//Reload configuration with viper if config file correctly downloaded
		if newconf, err := cfg.downloadMainConf(); err == nil {
			//download OK
			viper.SetConfigFile(newconf)
			verr := viper.ReadInConfig()
			var newCfg *Config
			if verr != nil {
				log.Errorf("Fatal error config file: %s \n", verr)
			} else {
				newCfg = new(Config)
				newCfg.InitConfig()
				verr = viper.Unmarshal(newCfg)
				if verr != nil {
					log.Warnf("ERROR unable to decode into struct, %v \n", verr)
				} else {
					cfg = newCfg
					cfg.EndConfig()
					log.Infof("Config Successfully reloaded !!")
					Freq2 := cfg.Server.ReloadConfig
					if Freq != Freq2 {
						Freq = Freq2
						log.Infof("reconfiguring check Period to : %d seconds", Freq)
						s = time.Tick(time.Duration(Freq) * time.Second)
					}
				}
			}
			log.Debugf("DATA:%+v", cfg)
		} else {
			log.Warningf(" I can not download the agent conf from remoteserver: %s", err)
		}
		//Check Main Process after config reload
		for i, g := range cfg.CheckGroup {
			changed := 0
			log.Debugf("init review Group: %d : %s ", i, g.CheckID)
			for _, f := range g.File {
				//check if exist
				if exist, _ := f.Exist(); exist == false {
					log.Infof("file %s has been created  current sum [ %s ]", f.Path, f.Sum)
					f.DownloadNew(cfg.NodeID, g.CheckID, cfg.Server)
					changed++
					continue
				}
				lastsum, modified := f.IsModified()
				if modified == true {
					log.Infof("file %s has been modified  last sum [ %s ] current sum [ %s ]", f.Path, lastsum, f.Sum)
					f.Backup()
					f.DownloadNew(cfg.NodeID, g.CheckID, cfg.Server)
				}
			}
			if changed > 0 {
				log.Infof("%d files have been changed in the Group [ %s ] procedd to reload , check , upload log", changed, g.CheckID)
				g.ExecReload()
				g.ExecCheck()
				g.UploadLog(cfg.NodeID, cfg.Server)
			}
		}
	LOOP:
		for {
			select {
			case <-s:
				break LOOP
			}
		}
	}
}

func main() {
	var wg sync.WaitGroup
	/*defer func() {
		//errorLog.Close()
	}()*/
	log.Debug("Init main")
	wg.Add(1)
	go CheckFiles(&wg, cfg)
	wg.Wait()
}
