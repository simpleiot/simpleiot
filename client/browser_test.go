package client

import (
	"bytes"
	"strconv"
	"testing"
)

// Does not err on empty file
func TestReadBrowserConfigEmptyFile(t *testing.T) {
	contents := ""
	file := bytes.NewBufferString(contents)

	_, err := readBrowserConfig(file)
	if err != nil {
		t.Error(err)
	}
}

// Correctly assigns vars
func TestReadBrowserConfigValidFile(t *testing.T) {
	browserURL := "http://localhost:8118"
	rotate := 180
	keyboardScale := true
	fullscreen := true
	defaultDialogs := true
	dialogColor := "#FF0000"
	touchQuirk := true
	retryInterval := 20
	exceptionURL := "http://exception-url"
	ignoreCertErr := true
	disableSandbox := true
	remoteDebugging := "0.0.0.0:9222"
	contents := `
				YOE_KIOSK_BROWSER_URL=` + browserURL + `
				YOE_KIOSK_BROWSER_ROTATE=` + strconv.Itoa(rotate) + `
				YOE_KIOSK_BROWSER_KEYBOARD_SCALE=` + boolToIntString(keyboardScale) + `
				YOE_KIOSK_BROWSER_FULLSCREEN=` + boolToIntString(fullscreen) + `
				YOE_KIOSK_BROWSER_DEFAULT_DIALOGS=` + boolToIntString(defaultDialogs) + `
				YOE_KIOSK_BROWSER_DIALOG_COLOR=` + dialogColor + `
				YOE_KIOSK_BROWSER_TOUCH_QUIRK=` + boolToIntString(touchQuirk) + `
				YOE_KIOSK_BROWSER_RETRY_INTERVAL=` + strconv.Itoa(retryInterval) + `
				YOE_KIOSK_BROWSER_EXCEPTION_URL=` + exceptionURL + `
				YOE_KIOSK_BROWSER_IGNORE_CERT_ERR=` + boolToIntString(ignoreCertErr) + `
				QTWEBENGINE_DISABLE_SANDBOX=` + boolToIntString(disableSandbox) + `
				QTWEBENGINE_REMOTE_DEBUGGING=` + remoteDebugging + `
				`
	file := bytes.NewBufferString(contents)

	config, err := readBrowserConfig(file)

	if err != nil {
		t.Error(err)
	}

	if config.URL != browserURL {
		t.Error("Failed to read YOE_KIOSK_BROWSER_URL")
	}
	if config.Rotate != rotate {
		t.Error("Failed to read YOE_KIOSK_BROWSER_ROTATE")
	}
	if config.KeyboardScale != keyboardScale {
		t.Error("Failed to read YOE_KIOSK_BROWSER_KEYBOARD_SCALE")
	}
	if config.Fullscreen != fullscreen {
		t.Error("Failed to read YOE_KIOSK_BROWSER_FULLSCREEN")
	}
	if config.DefaultDialogs != defaultDialogs {
		t.Error("Failed to read YOE_KIOSK_BROWSER_DEFAULT_DIALOGS")
	}
	if config.DialogColor != dialogColor {
		t.Error("Failed to read YOE_KIOSK_BROWSER_DIALOG_COLOR")
	}
	if config.TouchQuirk != touchQuirk {
		t.Error("Failed to read YOE_KIOSK_BROWSER_TOUCH_QUIRK")
	}
	if config.RetryInterval != retryInterval {
		t.Error("Failed to read YOE_KIOSK_BROWSER_RETRY_INTERVAL")
	}
	if config.ExceptionURL != exceptionURL {
		t.Error("Failed to read YOE_KIOSK_BROWSER_EXCEPTION_URL")
	}
	if config.IgnoreCertErr != ignoreCertErr {
		t.Error("Failed to read YOE_KIOSK_BROWSER_IGNORE_CERT_ERR")
	}
	if config.DisableSandbox != disableSandbox {
		t.Error("Failed to read QTWEBENGINE_DISABLE_SANDBOX")
	}
	if config.DebugPort != remoteDebugging {
		t.Error("Failed to read QTWEBENGINE_REMOTE_DEBUGGING")
	}
}

func TestBrowserConfigValueCheck(t *testing.T) {
	for _, test := range []struct {
		value string
		ok    bool
	}{
		{"", true},
		{"#D91824", true},
		{"1024x600", true},
		{"a\nQTWEBENGINE_DISABLE_SANDBOX=1", false},
		{"a\rb", false},
		{"a\x00b", false},
		{"a\tb", false},
	} {
		err := checkConfigValue(test.value)
		if test.ok && err != nil {
			t.Errorf("%q: unexpected error %v", test.value, err)
		}
		if !test.ok && err == nil {
			t.Errorf("%q: expected error", test.value)
		}
	}
}

func TestBrowserURLCheck(t *testing.T) {
	for _, test := range []struct {
		value string
		ok    bool
	}{
		{"", true},
		{"http://localhost:8080", true},
		{"https://example.com/dashboard?x=1", true},
		{"file:///usr/share/kiosk/index.html", true},
		{"javascript:alert(1)", false},
		{"ftp://example.com", false},
		{"localhost:8080", false},
		{"http://example.com\nX=1", false},
	} {
		err := checkBrowserURL(test.value)
		if test.ok && err != nil {
			t.Errorf("%q: unexpected error %v", test.value, err)
		}
		if !test.ok && err == nil {
			t.Errorf("%q: expected error", test.value)
		}
	}
}
