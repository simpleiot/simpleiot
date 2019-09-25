package main

import (
	"fmt"
	"image"
	"os"
	"regexp"
	"strconv"

	"github.com/simpleiot/simpleiot/farmation/fonts/agencyfbbold20"
	"github.com/simpleiot/simpleiot/farmation/fonts/tightpixel15"
	"github.com/simpleiot/simpleiot/farmation/isui"
)

var splash struct {
	message  string
	progress int // this is a percent
}

func main() {

	// extract arguments from command
	args := os.Args

	// define argument patterns
	typePattern := regexp.MustCompile(`\w+`)
	valuePattern := regexp.MustCompile(`\s\w+`)

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
		if len(currentArg.typeA) > 2 {
			currentArg.typeA = currentArg.typeA[1:] // chop off leading space
		}
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

	fmt.Println("COLLIN - splash:\nMessage: ", splash.message, "\nProgress: ", splash.progress)

	lcd := image.NewRGBA(image.Rect(0, 0, 128, 64))

	// Logo
	isui.DrawTxtCentered(lcd, "Farmation", 64, 26, agencyfbbold20.Font)

	// Message
	isui.DrawTxtCentered(lcd, splash.message, 64, 36, tightpixel15.Font)

	// Progress bar
	isui.Rect(lcd, 32, 40, 64, 6)
	if splash.progress > 100 || splash.progress < 0 {
		splash.progress = 100
	}
	isui.RectFilled(lcd, 33, 41, splash.progress/100*62, 6)
}
