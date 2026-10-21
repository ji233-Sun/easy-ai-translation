package main

import (
	"context"
	"strings"
)

// app is the state the window shows, and what the user does with it. Its
// view reads and changes it on the main thread; translations running in
// other goroutines change it through update.
type app struct {
	client  deepseek
	update  func(func())           // runs a function on the main thread, then redraws: Window.Update
	saveKey func(key string) error // keeps the API key, or forgets it when empty

	input, output string
	apiKey        string
	keyDraft      string // the API key being edited in the settings
	settingsOpen  bool

	busy   bool
	errMsg string
	job    int                // counts translations, so that a stopped one's late pieces are dropped
	cancel context.CancelFunc // stops the translation running
}

// translate translates the input, streaming the translation into the
// output, in place of one already running.
func (a *app) translate() {
	text := strings.TrimSpace(a.input)
	if text == "" {
		return
	}
	if a.apiKey == "" {
		a.errMsg = "请先设置 DeepSeek API Key"
		a.openSettings()
		return
	}
	a.stop()
	ctx, cancel := context.WithCancel(context.Background())
	job, client, key, target, update := a.job, a.client, a.apiKey, targetFor(text), a.update
	a.cancel, a.busy, a.output, a.errMsg = cancel, true, "", ""
	go func() {
		defer cancel()
		err := client.translate(ctx, key, text, target, func(delta string) {
			update(func() {
				if a.job == job {
					a.output += delta
				}
			})
		})
		update(func() {
			if a.job != job {
				return
			}
			a.busy, a.cancel = false, nil
			if err != nil && ctx.Err() == nil {
				a.errMsg = err.Error()
			}
		})
	}()
}

// stop stops the translation running, if any, keeping what it has
// delivered and dropping the rest.
func (a *app) stop() {
	if a.cancel != nil {
		a.cancel()
		a.cancel = nil
	}
	a.job++
	a.busy = false
}

// clear empties the input and the output.
func (a *app) clear() {
	a.stop()
	a.input, a.output, a.errMsg = "", "", ""
}

func (a *app) openSettings() {
	a.keyDraft, a.settingsOpen = a.apiKey, true
}

// saveSettings keeps the API key being edited, and closes the settings.
func (a *app) saveSettings() error {
	key := strings.TrimSpace(a.keyDraft)
	if err := a.saveKey(key); err != nil {
		return err
	}
	a.apiKey, a.settingsOpen, a.errMsg = key, false, ""
	return nil
}
