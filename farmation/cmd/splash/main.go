package main

import (
	"flag"
	"fmt"
	"image"
	"log"
	"os"
	"regexp"
	"strconv"

	"github.com/simpleiot/simpleiot/farmation/fonts/tightpixel15"
	"github.com/simpleiot/simpleiot/farmation/isio"
	"github.com/simpleiot/simpleiot/farmation/islcd"
	"github.com/simpleiot/simpleiot/farmation/isui"
)

var splash struct {
	message  string
	progress int // this is a percent
}

func main() {

	flagInit := flag.Bool("init", false, "initialize display")
	flag.Parse()

	// extract arguments from command
	args := os.Args

	// define argument patterns
	typePattern := regexp.MustCompile(`\w+`)
	valuePattern := regexp.MustCompile(`\s.+`)

	// store current argument
	var currentArg struct {
		typeA string
		value string
	}

	// use each argument to update splash status
	for i, arg := range args {
		if i == 0 { // skip original command -- just look at flags
			continue
		}

		currentArg.typeA = string(typePattern.FindString(arg))
		currentArg.value = string(valuePattern.FindString(arg))
		if len(currentArg.value) > 2 {
			currentArg.value = currentArg.value[1:] // chop off leading space
		}

		switch currentArg.typeA {
		case "MSG":
			splash.message = currentArg.value
		case "PROGRESS":
			splash.progress, _ = strconv.Atoi(currentArg.value)
		}
	}

	lcd, err := islcd.NewLcd()

	if err != nil {
		log.Println("Error opening LCD", err)
		os.Exit(-1)
	}

	err = isio.GpioInit()
	if err != nil {
		log.Println("Error initializing GPIO: ", err)
		os.Exit(-1)
	}

	if *flagInit {
		fmt.Println("Initializing")
		err := lcd.Init()
		if err != nil {
			log.Println("Error initializing LCD: ", err)
			os.Exit(-1)
		}
	}

	img := image.NewRGBA(image.Rect(0, 0, 128, 64))
	isui.Clear(img)

	file := "IS_logo_injector.png"
	// Logo
	err = isui.DrawPng(img, file, 26, 0)
	if err != nil {
		s := fmt.Sprintf("error drawing %s: %s", file, err)
		fmt.Println(s)
	}

	// Message
	isui.DrawTxtCentered(img, splash.message, 64, 54, tightpixel15.Font)

	// Progress bar
	isui.Rect(img, 32, 45, 64, 6)
	if splash.progress > 100 || splash.progress < 0 {
		splash.progress = 100
	}
	progressPerc := float64(splash.progress) * 0.01
	isui.RectFilled(img, 33, 46, int(progressPerc*63), 4)

	lcd.Write(isui.ImageToBlt(0, 0, img, false).Data)
}
