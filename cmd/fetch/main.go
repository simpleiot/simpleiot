// test download program
package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"path"
	"time"

	"github.com/cavaliercoder/grab"
)

// test app to evaluate the grab http client for downloading OS updates
// we've had trouble downloading over Cat-M connections where the download
// will stall, so the this app is used to debug these downloads.

func httpDownload(url, fn string) {

	file, err := os.Create(fn)
	if err != nil {
		log.Println("App update: error creating file:", err)
		return
	}

	defer file.Close()

	var netClient = &http.Client{
		Timeout: 30 * time.Minute,
	}

	// Get the data
	resp, err := netClient.Get(url)
	if err != nil {
		log.Println("App update: error getting update file:", err)
		return
	}
	defer resp.Body.Close()

	// copy to file
	_, err = io.Copy(file, resp.Body)

	if err != nil {
		log.Println("App update: error downloading update file:", err)
		return
	}

	log.Println("Download finished")

}

func grabDownload(url, fn string) {
	file, err := os.Create(fn)
	if err != nil {
		log.Println("App update: error creating file:", err)
		return
	}

	defer file.Close()

	client := grab.NewClient()
	req, _ := grab.NewRequest(fn, url)
	// ...
	resp := client.Do(req)

	t := time.NewTicker(time.Second)
	defer t.Stop()

	for {
		select {
		case <-t.C:
			fmt.Printf("%.02f%% complete, %.02f B/sec\n",
				resp.Progress()*100,
				resp.BytesPerSecond())

		case <-resp.Done:
			if err := resp.Err(); err != nil {
				log.Println("Error downloading:", err)
			}
			return
		}
	}
}

func usage() {
	fmt.Println("Usage: ")
	flag.PrintDefaults()
	os.Exit(-1)
}

func main() {

	flagGohttp := flag.Bool("gohttp", false, "use Go http client")
	flagGrab := flag.Bool("grab", false, "use grab http client")
	flagURL := flag.String("url", "", "URL of file to download")

	flag.Parse()

	if *flagURL == "" {
		log.Println("Must set URL")
	}

	if !*flagGohttp && !*flagGrab {
		log.Println("Must set -gohttp or -grab option", *flagGohttp, *flagGrab)
		usage()
	}

	url, err := url.Parse(*flagURL)

	if err != nil {
		log.Println("Error parsing url:", err)
		os.Exit(-1)
	}

	urlPath := url.EscapedPath()
	_, fn := path.Split(urlPath)

	if *flagGohttp {
		httpDownload(url.String(), fn)
	}

	if *flagGrab {
		grabDownload(url.String(), fn)
	}
}
