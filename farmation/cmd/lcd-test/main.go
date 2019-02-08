package main

import (
	"os"
	"fmt"
	"log"
	"io/ioutil"
	"time"
	"periph.io/x/periph/conn/gpio"
	"periph.io/x/periph/conn/gpio/gpioreg"
	"periph.io/x/periph/host"
)

// NT7534 lcd controller:
const NT7534_00_SET_LOWER_COLUMN_ADDRESS_BIT byte = 0x00
const NT7534_10_SET_UPPER_COLUMN_ADDRESS_BIT byte = 0x10
const NT7534_20_RESISTOR_RATIO_RANGE_SET_BIT byte = 0x20
const NT7534_28_POWER_CONTROL_SET_BIT  byte       = 0x28
const NT7534_30_PARTIAL_DISPLAY_DUTY_BIT  byte    = 0x30
const NT7534_38_PARTIAL_DISPLAY_BIAS_BIT  byte    = 0x38
const NT7534_40_SET_DISPLAY_START_LINE_BIT  byte  = 0x40
const NT7534_81_CONTRAST_PREFIX  byte             = 0x81
const NT7534_82_PARTIAL_DISPLAY_MODE_OFF byte     = 0x82
const NT7534_83_PARTIAL_DISPLAY_MODE_ON  byte     = 0x83
const NT7534_84_INVERSION_BY_FRAME byte           = 0x84
const NT7534_85_INVERSION_BY_NLINE_PREFIX  byte   = 0x85
const NT7534_A0_SEGMENT_REMAP_NORMAL  byte        = 0xA0
const NT7534_A1_SEGMENT_REMAP_REVERSE   byte      = 0xA1
const NT7534_A2_LCD_BIAS_1_9  byte                = 0xA2
const NT7534_A3_LCD_BIAS_1_7   byte               = 0xA3
const NT7534_A4_ENTIRE_DISPLAY_NORMAL  byte       = 0xA4
const NT7534_A5_ENTIRE_DISPLAY_FORCE_ON  byte     = 0xA5
const NT7534_A6_INVERSION_NORMAL   byte           = 0xA6
const NT7534_A7_INVERSION_INVERTED  byte          = 0xA7
const NT7534_AC_STATIC_INDICATOR_OFF_PREFIX  byte = 0xAC
const NT7534_AD_STATIC_INDICATOR_ON_PREFIX  byte  = 0xAD
const NT7534_00_STATIC_IND_OFF_PARAMETER   byte   = 0x00
const NT7534_01_STATIC_IND_1S_BLINK_PARAMETER  byte = 0x01
const NT7534_02_STATIC_IND_0P5S_BLINK_PARAMETER  byte = 0x02
const NT7534_03_STATIC_IND_ON_PARAMETER   byte    = 0x03
const NT7534_AE_DISPLAY_OFF_SLEEP_YES   byte      = 0xAE
const NT7534_AF_DISPLAY_ON_SLEEP_NO    byte       = 0xAF
const NT7534_B0_SET_PAGE_START_ADDRESS_BIT byte   = 0xB0
const NT7534_C0_COM_DIRECTION_NORMAL   byte       = 0xC0
const NT7534_C8_COM_DIRECTION_REVERSE   byte      = 0xC8
const NT7534_D3_PARTIAL_DISPLAY_START_LINE_PREFIX  byte = 0xD3
const NT7534_E0_START_READ_MODIFY_WRITE byte      = 0xE0
const NT7534_E2_RESET byte                        = 0xE2
const NT7534_E3_NOP   byte                        = 0xE3
const NT7534_E4_OSC_FREQ_31KHZ    byte            = 0xE4
const NT7534_E5_OSC_FREQ_26KHZ     byte           = 0xE5
const NT7534_E6_DC_DC_FREQUENCY_PREFIX byte       = 0xE6
const NT7534_EE_END_READ_MODIFY_WRITE  byte       = 0xEE

const RegSel_Gpio string = "PD26"
const LcdReset_Gpio string = "PA26"


func main() {

	//gpio support:
	if _, err := host.Init(); err != nil {
		log.Fatal(err)
		fmt.Println("GPIO setup failed.")
	}
	//This is a horrid hack:
//	nCS0 := gpioreg.ByName("PC31")
//	if nCS0 == nil {
//		log.Fatal(nCS0)
//		fmt.Println("Failed to get GPIO for nCS0 pin.")
//	}
	
//	nCS0.Out(gpio.Low)

	//Try to grab the gpio for the register address sel and nRST:
	RegSel := gpioreg.ByName(RegSel_Gpio)
	if RegSel == nil {
		log.Fatal(RegSel)
		fmt.Println("GPIO register bad number for RegSel")
	}
	LcdReset := gpioreg.ByName(LcdReset_Gpio)
	if LcdReset == nil {
		log.Fatal(LcdReset)
		fmt.Println("GPIO register bad number for LcdReset")
	}

	//display is write only
	_, err := os.Open("/dev/spidev1.0")
	if err != nil {
		log.Fatal(err)
		fmt.Println("Failed to open spidev device file")
	}

	//set up display controller
//	cmdOp := make([]byte,12)  //all commands are single bytes
	cmdOp := make([]byte,8)  //all commands are single bytes
				// EXCEPT for the contrast adjustment
// dev kit way:
//	cmdOp[0] = NT7534_E2_RESET
//	cmdOp[1] = NT7534_A1_SEGMENT_REMAP_REVERSE // set ADC
//	cmdOp[2] = NT7534_C0_COM_DIRECTION_NORMAL  // set common scan
//	cmdOp[3] = NT7534_A2_LCD_BIAS_1_9	// set lcd bias
//	cmdOp[4] = NT7534_20_RESISTOR_RATIO_RANGE_SET_BIT | 0x4
//	cmdOp[5] = NT7534_81_CONTRAST_PREFIX
//	cmdOp[6] = NT7534_28_POWER_CONTROL_SET_BIT | 0x07
//	cmdOp[7] = 24  //raw contrast value
//	cmdOp[8] = NT7534_40_SET_DISPLAY_START_LINE_BIT 
//	cmdOp[9] = NT7534_AF_DISPLAY_ON_SLEEP_NO
//	cmdOp[10] = NT7534_A6_INVERSION_NORMAL
//	cmdOp[11] = NT7534_A4_ENTIRE_DISPLAY_NORMAL

//  my way:
	cmdOp[0] = NT7534_E2_RESET

	//set command register
	err = RegSel.Out(gpio.High)
	if err != nil {
		log.Fatal(err)
		fmt.Println("Failed to set gpio for RegSel")
	}

	// hardware reset of LCD display
	err = LcdReset.Out(gpio.High)
	if err != nil {
		log.Fatal(err)
		fmt.Println("Failed to set gpio for LcdREset")
	}

	time.Sleep(100 * time.Millisecond)
	LcdReset.Out(gpio.Low)
	time.Sleep(10 * time.Millisecond)
	LcdReset.Out(gpio.High)
	time.Sleep(10 * time.Millisecond)
	
	err = ioutil.WriteFile("/dev/spidev1.0",cmdOp[0:1],666)
	if err != nil {
		log.Fatal(err)
		fmt.Println("Failed to write to spidev")
	}

	time.Sleep(10 * time.Millisecond)

//	for i, _ := range cmdOp {
//		err = ioutil.WriteFile("/dev/spidev1.0",cmdOp[i:i+1],666)
//		if err != nil {
//			log.Fatal(err)
//			fmt.Println("Failed to write to spidev")
//		}
//		fmt.Println("Sent command %x: \n",cmdOp[i:i+1])
//		time.Sleep(1 * time.Millisecond)
//	}

	cmdOp[0] = NT7534_A2_LCD_BIAS_1_9
	cmdOp[1] = NT7534_A0_SEGMENT_REMAP_NORMAL 
	cmdOp[2] = NT7534_C8_COM_DIRECTION_REVERSE 
	cmdOp[3] = 0x24  //OR 0x24 as above?
	cmdOp[4] = 0x2F  //all internal blocks on
	cmdOp[5] = NT7534_81_CONTRAST_PREFIX
	cmdOp[6] = 20
	cmdOp[7] = NT7534_40_SET_DISPLAY_START_LINE_BIT

	err = ioutil.WriteFile("/dev/spidev1.0",cmdOp,666)
	if err != nil {
		log.Fatal(err)
		fmt.Println("Failed to write to spidev")
	}

	fmt.Println("Initialization of LCD display complete.")

	//fill display memory, hopefully to initialize it
	for page :=0; page < 8; page++ {
		cmdOp[0] = 0xB0 + (byte) (page & 0x0F)
		cmdOp[1] = 0  //lower column address nibble
		cmdOp[2] = 0x10  //upper column nibble
		RegSel.Out(gpio.Low)
		err = ioutil.WriteFile("/dev/spidev1.0",cmdOp[0:3],666)
		if err != nil {
			log.Fatal(err)
			fmt.Println("Failed to write to spidev")
		}
		time.Sleep(1 * time.Millisecond) // have to wait for spi write to complete before twiddling A0
		RegSel.Out(gpio.High)
		cmdOp[0] = 0xFF
		for column := 0; column < 128; column++ {
			err = ioutil.WriteFile("/dev/spidev1.0",cmdOp[0:1],666)
			if err != nil {
				log.Fatal(err)
				fmt.Println("Failed to write to spidev")
			}
		}
		time.Sleep(1 * time.Millisecond) // have to wait for spi write to complete before twiddling A0
	}

	//Display testing functions:
	//First, turn all pixels on:
	cmdOp[0] = NT7534_A5_ENTIRE_DISPLAY_FORCE_ON
	cmdOp[1] = NT7534_A4_ENTIRE_DISPLAY_NORMAL

	RegSel.Out(gpio.Low)  //point at cmd port
	for {
		ioutil.WriteFile("/dev/spidev1.0",cmdOp[0:1],666)
		fmt.Println("Set all dots on")
		time.Sleep(1000 * time.Millisecond)
		ioutil.WriteFile("/dev/spidev1.0",cmdOp[1:2],666)
		fmt.Println("Set all dots off")
		time.Sleep(1000 * time.Millisecond)
	}
}
