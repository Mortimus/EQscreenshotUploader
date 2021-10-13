package main

import (
	"errors"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"runtime"
	"sync"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/fsnotify/fsnotify"
	"github.com/getlantern/systray"
)

func main() {
	file, err := os.OpenFile(configuration.Log.Path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0666)
	if err != nil {
		log.Fatal(err)
	}
	log.SetFlags(log.Ldate | log.Lshortfile)
	log.SetOutput(file)
	// Create a new Discord session using the provided bot token.
	discord, err := discordgo.New("Bot " + getDiscordToken())
	if err != nil {
		log.Fatalf("Error creating Discord session: %v", err)
	}
	defer discord.Close()
	var shots screenShots
	shots.Init()
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Fatal(err)
	}
	defer watcher.Close()

	// done := make(chan bool)
	go func() {
		for {
			select {
			case event, ok := <-watcher.Events:
				if !ok {
					return
				}
				if event.Op&fsnotify.Write == fsnotify.Write {
					err := shots.Upload(event.Name, configuration.Main.ScreenshotExtension, discord)
					if err != nil {
						log.Println(err)
					}
				}
			case err, ok := <-watcher.Errors:
				if !ok {
					return
				}
				log.Println("error:", err)
			}
		}
	}()

	err = watcher.Add(shots.Folder)
	if err != nil {
		log.Fatal(err)
	}
	systray.Run(onReady, onExit)
	// <-done
}

func onReady() {
	pIcon, err := ioutil.ReadFile("Everquest.ico")
	if err != nil {
		log.Fatal(err)
	}
	systray.SetIcon(pIcon)
	systray.SetTitle("EQscreenshotUploader")
	systray.SetTooltip("Everquest Screenshot Monitor")
	if runtime.GOOS == "windows" {
		mConfig := systray.AddMenuItem("Configuration", "Open configuration for editing")
		go func() { // Todo: we need to move this as it can only be ran once here
			<-mConfig.ClickedCh
			ex, err := os.Executable()
			if err != nil {
				panic(err)
			}
			exPath := filepath.Dir(ex)
			log.Printf("Editing config at %s\n", filepath.Join(exPath, configPath))
			cmd := exec.Command("rundll32.exe", "url.dll,FileProtocolHandler", filepath.Join(exPath, configPath))
			cmd.Start()
		}()
		mLog := systray.AddMenuItem("Open Log", "View Log")
		go func() { // Todo: we need to move this as it can only be ran once here
			<-mLog.ClickedCh
			ex, err := os.Executable()
			if err != nil {
				panic(err)
			}
			exPath := filepath.Dir(ex)
			log.Printf("Opening log at %s\n", filepath.Join(exPath, configuration.Log.Path))
			cmd := exec.Command("rundll32.exe", "url.dll,FileProtocolHandler", filepath.Join(exPath, configuration.Log.Path))
			cmd.Start()
		}()
		startup := systray.AddMenuItem("Autostart", "Launch EQscreenshotUploader when Windows starts")
		startup.AddSubMenuItem("On", "Automatically launch with Windows")
		startup.AddSubMenuItem("Off", "Disable launching with Windows")
	}
	mQuit := systray.AddMenuItem("Exit", "Stop monitoring screenshots")
	go func() {
		<-mQuit.ClickedCh
		log.Printf("Quitting")
		systray.Quit()
	}()
	// Sets the icon of a menu item. Only available on Mac and Windows.
	// mQuit.SetIcon(icon.Data)
}

func onExit() {
	// clean up here
	os.Exit(0)
}

func findPath() string {
	// check config file for path
	log.Printf("Looking for everquest screenshots at %s\n", configuration.Everquest.ScreenshotPath)
	if _, err := os.Stat(configuration.Everquest.ScreenshotPath); !os.IsNotExist(err) {
		log.Printf("Using screenshot folder %s\n", configuration.Everquest.ScreenshotPath)
		return configuration.Everquest.ScreenshotPath
	}

	// check appdata C:\Users\<USER_NAME>\AppData\Local\VirtualStore\Program Files (x86)\Sony\EverQuest\Screenshots
	user, err := user.Current()
	if err != nil {
		log.Println(err)
	} else {
		appDataPath := filepath.Join(user.HomeDir, "AppData", "Local", "VirtualStore", "Program Files (x86)", "Sony", "EverQuest", "Screenshots")
		log.Printf("Looking for everquest screenshots at %s\n", appDataPath)
		if _, err := os.Stat(appDataPath); !os.IsNotExist(err) {
			log.Printf("Using screenshot folder %s\n", appDataPath)
			return appDataPath
		}
	}
	// check if steam path exists
	const steamPath = `C:\Program Files (x86)\Steam\steamapps\common\Everquest F2P\Screenshots`
	log.Printf("Looking for everquest screenshots at %s\n", steamPath)
	if _, err := os.Stat(steamPath); !os.IsNotExist(err) {
		log.Printf("Using screenshot folder %s\n", steamPath)
		return steamPath
	}
	// check other path

	// Path not found
	return ""
}

type screenShots struct {
	known  map[string]interface{}
	lock   sync.Mutex
	Folder string
}

func (s *screenShots) Init() {
	s.known = make(map[string]interface{})
	s.Folder = findPath()
	err := s.LoadExisting(configuration.Main.ScreenshotExtension)
	if err != nil {
		log.Fatal(err)
	}
}

func (s *screenShots) add(path string) {
	s.lock.Lock()
	defer s.lock.Unlock()
	s.known[path] = nil
}

func (s *screenShots) exists(path string) bool {
	s.lock.Lock()
	defer s.lock.Unlock()
	_, ok := s.known[path]
	return ok
}

func (s *screenShots) Upload(path string, ext string, discord *discordgo.Session) error {
	if filepath.Ext(path) != ext {
		return errors.New("wrong file extension " + filepath.Ext(path) + " expected " + ext)
	}
	if !s.exists(path) {
		s.add(path)
		log.Printf("Uploading %s\n", path)
		seconds := time.Second * time.Duration(configuration.Main.UploadDelay)
		log.Printf("Waiting %.1f seconds before uploading\n", seconds.Seconds())
		time.Sleep(seconds)
		file, err := os.Open(path)
		if err != nil {
			return err
		}
		defer file.Close()
		discord.ChannelFileSend(getDiscordChannel(), filepath.Base(path), file)
		return nil
	}
	return errors.New("ignoring " + path)
}

func (s *screenShots) LoadExisting(ext string) error {
	files, err := ioutil.ReadDir(s.Folder)
	if err != nil {
		return err
	}

	for _, file := range files {
		if filepath.Ext(file.Name()) == ext {
			s.add(filepath.Join(s.Folder, file.Name()))
		}
	}
	return nil
}
