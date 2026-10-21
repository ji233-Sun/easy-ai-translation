package main

import (
	"errors"
	"log"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
)

// apiKeyName is the key the DeepSeek API key is kept under in the system's
// secret store: the login Keychain on macOS.
const apiKeyName = "deepseek-api-key"

func loadAPIKey() (string, error) {
	key, err := mygo.Secrets.Get(apiKeyName)
	if errors.Is(err, mygo.ErrSecretNotFound) {
		return "", nil
	}
	return string(key), err
}

func saveAPIKey(key string) error {
	if key == "" {
		return mygo.Secrets.Delete(apiKeyName)
	}
	return mygo.Secrets.Set(apiKeyName, []byte(key))
}

func main() {
	a := &app{
		client:  deepseek{baseURL: "https://api.deepseek.com"},
		saveKey: saveAPIKey,
	}
	mygo.App.WhenReady(func() {
		win := mygo.NewWindow(mygo.WindowOptions{
			Title:     "Easy Translate",
			Width:     880,
			Height:    560,
			MinWidth:  620,
			MinHeight: 400,
			// Opens where the user left it last time.
			StateKey: "main",
			// Hide the title bar, the traffic lights inset over the view,
			// and show the frosted material of macOS behind it.
			TitleBarStyle:  mygo.TitleBarHiddenInset,
			TitleBarHeight: titleBarHeight,
			Vibrancy:       mygo.VibrancySidebar,
			Content:        ui.View(a.view),
		})
		a.update = win.Update
		go func() {
			key, err := loadAPIKey()
			win.Update(func() {
				switch {
				case err != nil:
					a.errMsg = "无法读取钥匙串：" + err.Error()
				case key == "":
					a.openSettings() // the first launch
				default:
					a.apiKey = key
				}
			})
		}()
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}
