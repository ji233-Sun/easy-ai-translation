package main

import (
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
)

// The view runs without a window in tests, which click and type as a user
// would: it asks for an API key first, then streams the translation of
// what is typed.
func TestView(t *testing.T) {
	var req chatRequest
	srv := fakeDeepSeek(t, &req, "Hello", ", world")
	updates := make(chan func(), 16)
	var saved string
	a := &app{
		client:  deepseek{baseURL: srv.URL},
		update:  func(fn func()) { updates <- fn },
		saveKey: func(key string) error { saved = key; return nil },
	}
	tt := ui.NewTester(a.view, 880, 560)
	tt.SetVibrancy(true)

	tt.Type("你好，世界")
	if a.input != "你好，世界" || !tt.HasText("中文") || !tt.HasText("English") {
		t.Fatalf("input %q, texts %q", a.input, tt.Texts())
	}

	// Without a key, translating asks for one.
	tt.Key(ui.Cmd, ui.KeyEnter)
	if !a.settingsOpen {
		t.Fatalf("the settings are closed after translating without a key; texts %q", tt.Texts())
	}
	tt.Type("sk-test")
	if err := tt.Click("保存"); err != nil {
		t.Fatal(err)
	}
	if saved != "sk-test" || a.apiKey != "sk-test" || a.settingsOpen {
		t.Fatalf("saved %q, key %q, settings open %v", saved, a.apiKey, a.settingsOpen)
	}

	if err := tt.Click("翻译"); err != nil {
		t.Fatal(err)
	}
	for a.busy {
		select {
		case fn := <-updates:
			fn()
		case <-time.After(5 * time.Second):
			t.Fatal("the translation did not finish")
		}
	}
	tt.Frame()
	if a.output != "Hello, world" || a.errMsg != "" || !tt.HasText("复制") {
		t.Fatalf("output %q, error %q, texts %q", a.output, a.errMsg, tt.Texts())
	}

	if err := tt.Click("复制"); err != nil {
		t.Fatal(err)
	}
	if tt.Clipboard() != "Hello, world" {
		t.Errorf("clipboard %q after copying", tt.Clipboard())
	}
	if err := tt.Click("清空"); err != nil {
		t.Fatal(err)
	}
	if a.input != "" || a.output != "" {
		t.Errorf("input %q and output %q after clearing", a.input, a.output)
	}
}
