package isupdate

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path"
	"runtime"
	"time"

	"github.com/simpleiot/simpleiot/data"
)

// updateApp is meant to be run as a goroutine and sends status to
// out channel
func updateApp(url string, out chan interface{}) {
	log.Println("Starting app update: ", url)

	updateDir := "./"

	if runtime.GOARCH == "arm" {
		updateDir = "/data/update"

		if err := os.RemoveAll(updateDir); err != nil {
			fmt.Println("Error cleaning update dir")
			return
		}

		if err := os.MkdirAll(updateDir, os.ModeDir); err != nil {
			fmt.Println("Error creating update directory")
			return
		}
	}

	filePath := path.Join(updateDir, "is-arm.xz")
	filePathUnzipped := path.Join(updateDir, "is-arm")
	os.Remove(filePath)

	file, err := os.Create(filePath)
	if err != nil {
		log.Println("App update: error creating file: ", err)
		return
	}

	defer file.Close()

	var netClient = &http.Client{
		Timeout: 30 * time.Minute,
	}

	// Get the data
	resp, err := netClient.Get(url)
	if err != nil {
		log.Println("App update: error getting update file: ", err)
		return
	}
	defer resp.Body.Close()

	// copy to file
	_, err = io.Copy(file, resp.Body)

	if err != nil {
		log.Println("App update: error downloading update file: ", err)
		return
	}

	err = exec.Command("xz", "-d", filePath).Run()
	if err != nil {
		log.Println("Error decompressing update: ", err)
		return
	}

	if runtime.GOARCH != "arm" {
		// nothing more to do if not on target device
		log.Println("App update done, not installing on dev machine")
		return
	}

	err = exec.Command("mv", filePathUnzipped, "/usr/bin/is").Run()
	if err != nil {
		log.Println("App update: error installing binary: ", err)
		return
	}

	err = exec.Command("chmod", "755", "/usr/bin/is").Run()
	if err != nil {
		log.Println("App update: error setting mode: ", err)
		return
	}

	log.Println("App update finished, restarting app")

	err = exec.Command("/etc/init.d/isapp", "restart").Start()

	if err != nil {
		log.Println("App update: error restarting app: ", err)
	}
}

// Run goroutine for software update process
func Run(in, out chan interface{}) {
	for {
		select {
		case m := <-in:
			switch m := m.(type) {
			case data.DeviceCmd:
				switch m.Cmd {
				case data.CmdUpdateApp:
					go updateApp(m.Detail, out)
				default:
					log.Print("isupdate mux: unhandled command: ", m.Cmd)
				}
			default:
				log.Printf("isupdate mux: unhandled message of type %T: %+v\r\n", m, m)
			}
		}
	}
}
