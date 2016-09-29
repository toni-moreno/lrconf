package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
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
	NodeID     string `toml:"nodeid"`
	tmpdir     string
	General    GeneralConfig
	Server     ServerConfig
	CheckFiles map[string]*CheckFileConfig
}

var (
	log = logrus.New()

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

	//
	log.Printf("set Default directories : \n   - Exec: %s\n   - Config: %s\n   -Logs: %s\n", appdir, confDir, logDir)

	// parse first time to see if config file is being specified
	f := flags()
	f.Parse(os.Args[1:])
	// now load up config settings
	viper.Set("Verbose", true)
	viper.Set("LogFile", "./log/viper.log")

	if _, err := os.Stat(configFile); err == nil {
		log.Info("no config file set")
		viper.SetConfigFile(configFile)
	} else {
		log.Info("set default config files")

		viper.SetConfigName("lrconf-agent")
		viper.AddConfigPath("/opt/lrconf/conf/")
		viper.AddConfigPath("./conf/")
		viper.AddConfigPath(".")

	}
	err := viper.ReadInConfig()
	if err != nil {
		panic(fmt.Errorf("Fatal error config file: %s \n", err))
	}
	//Allocating Config struct
	cfg = new(Config)
	cfg.InitConfig()
	err = viper.Unmarshal(cfg)
	if err != nil {
		panic(fmt.Errorf("unable to decode into struct, %v \n", err))
	}

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

	for id, f := range cfg.CheckFiles {
		if ok, err := f.InitCheck(); ok != true {
			log.Warningf("Error in config file file %s has errors: %s: ", id, err)
			delete(cfg.CheckFiles, id)
		}
		log.Infof("File Check OK : %s : %s", id, f.FilePath)
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

//DownloadNew to download the new version of this file
func (c *Config) downloadMainConf() (string, error) {
	log.Debugf("Download new config file from server..")
	basename := "lrconf-agent.toml"
	rawURL := "http://" + c.Server.CentralConfigServer + ":" + strconv.Itoa(c.Server.CentralConfigPort) + "/" + c.NodeID + "/" + basename
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
			err := viper.ReadInConfig()
			var newCfg *Config
			if err != nil {
				log.Errorf("Fatal error config file: %s \n", err)
			} else {
				newCfg = new(Config)
				err = viper.Unmarshal(newCfg)
				if err != nil {
					log.Warnf("ERROR unable to decode into struct, %v \n", err)
				} else {
					cfg = newCfg
					cfg.InitConfig()
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
		}
		//Check Main Process after config reload
		for id, f := range cfg.CheckFiles {
			//log.Debug("DATA: %+v", *f)
			log.Debugf("init review file: %s with path %s", id, f.FilePath)
			//check if file exist
			if exist, _ := f.Exist(); exist == false {
				log.Infof("file %s has been created  current sum [ %s ]", f.FilePath, f.FileSum)
				f.DownloadNew(cfg.NodeID, cfg.Server)
				f.ExecReload()
				f.ExecCheck()
				f.UploadLog(cfg.NodeID)
				continue
			}
			lastsum, modified := f.IsModified()
			if modified == true {
				log.Infof("file %s has been modified  last sum [ %s ] current sum [ %s ]", f.FilePath, lastsum, f.FileSum)
				f.Backup()
				f.DownloadNew(cfg.NodeID, cfg.Server)
				f.ExecReload()
				f.ExecCheck()
				f.UploadLog(cfg.NodeID)
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
	//go CheckFiles(&wg, cfg.CheckFiles, cfg.Server.ReloadConfig)
	wg.Wait()
}