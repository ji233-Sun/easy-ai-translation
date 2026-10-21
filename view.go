package main

import (
	"runtime"
	"strings"
	"unicode/utf8"

	"github.com/egoist/mygo/ui"
)

const (
	titleBarHeight = 52 // the height of the inset title bar
	apiKeysURL     = "https://platform.deepseek.com/api_keys"
)

// translateShortcut is how the platform writes the shortcut that translates.
var translateShortcut = func() string {
	if runtime.GOOS == "darwin" {
		return "⌘↩"
	}
	return "Ctrl+Enter"
}()

func icon(shapes string) *ui.SVG {
	return ui.MustParseSVG([]byte(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round">` + shapes + `</svg>`))
}

// Icons from Lucide (ISC license).
var (
	settingsIcon = icon(`<path d="M12.22 2h-.44a2 2 0 0 0-2 2v.18a2 2 0 0 1-1 1.73l-.43.25a2 2 0 0 1-2 0l-.15-.08a2 2 0 0 0-2.73.73l-.22.38a2 2 0 0 0 .73 2.73l.15.1a2 2 0 0 1 1 1.72v.51a2 2 0 0 1-1 1.74l-.15.09a2 2 0 0 0-.73 2.73l.22.38a2 2 0 0 0 2.73.73l.15-.08a2 2 0 0 1 2 0l.43.25a2 2 0 0 1 1 1.73V20a2 2 0 0 0 2 2h.44a2 2 0 0 0 2-2v-.18a2 2 0 0 1 1-1.73l.43-.25a2 2 0 0 1 2 0l.15.08a2 2 0 0 0 2.73-.73l.22-.39a2 2 0 0 0-.73-2.73l-.15-.08a2 2 0 0 1-1-1.74v-.5a2 2 0 0 1 1-1.74l.15-.09a2 2 0 0 0 .73-2.73l-.22-.38a2 2 0 0 0-2.73-.73l-.15.08a2 2 0 0 1-2 0l-.43-.25a2 2 0 0 1-1-1.73V4a2 2 0 0 0-2-2z"/><circle cx="12" cy="12" r="3"/>`)
	copyIcon     = icon(`<rect width="14" height="14" x="8" y="8" rx="2" ry="2"/><path d="M4 16c-1.1 0-2-.9-2-2V4c0-1.1.9-2 2-2h10c1.1 0 2 .9 2 2"/>`)
	clearIcon    = icon(`<path d="M18 6 6 18"/><path d="m6 6 12 12"/>`)
	arrowIcon    = icon(`<path d="M5 12h14"/><path d="m12 5 7 7-7 7"/>`)
	alertIcon    = icon(`<circle cx="12" cy="12" r="10"/><path d="M12 8v4"/><path d="M12 16h.01"/>`)
)

func (a *app) view(c *ui.Context) {
	vibrant := c.Vibrancy()
	if vibrant {
		// The window's material shows wherever the view draws no
		// background: around the panes, and a little through them.
		c.Root().Background(ui.Transparent)
	} else {
		c.Root().Background(c.Theme().Surface)
	}
	c.OnShortcut(ui.Cmd, ui.KeyEnter, a.translate)
	c.OnShortcut(ui.Cmd, ui.KeyComma, a.openSettings)
	ui.Column(c).Fill().Children(func() {
		a.titleBar(c)
		ui.Row(c).Grow(1).MinHeight(0).Padding(0, 16).Gap(6).AlignItems(ui.Stretch).Children(func() {
			a.sourcePane(c, vibrant)
			ui.Column(c).Justify(ui.Center).Children(func() {
				ui.Icon(c, arrowIcon).Size(16, 16).TextColor(c.Theme().TextMuted)
			})
			a.resultPane(c, vibrant)
		})
		a.footer(c)
	})
	a.settings(c)
}

// titleBar is the room of the hidden title bar: it drags the window, beside
// the window controls inset over it.
func (a *app) titleBar(c *ui.Context) {
	bar := c.TitleBar()
	ui.Row(c).Height(titleBarHeight).Shrink(0).Padding(0, bar.Right+12, 0, bar.Left+16).AlignItems(ui.Center).DragWindow().Children(func() {
		ui.Text(c, "Easy Translate").FontSize(15).Bold().SingleLine()
		ui.Spacer(c)
		iconButton(c, settingsIcon, "设置").OnClick(a.openSettings)
	})
}

func (a *app) sourcePane(c *ui.Context, vibrant bool) {
	source := "自动检测"
	if strings.TrimSpace(a.input) != "" {
		source = "其他语言"
		if targetFor(a.input) == english {
			source = "中文"
		}
	}
	pane(c, vibrant).Children(func() {
		paneHeader(c, source, func() {
			if a.input != "" && iconButton(c, clearIcon, "清空").Clicked() {
				a.clear()
			}
		})
		textPane(c.Key("input"), &a.input).Placeholder("输入中文，或其他语言的文字…").AutoFocus()
	})
}

func (a *app) resultPane(c *ui.Context, vibrant bool) {
	target, placeholder := "译文", "译文会显示在这里"
	if strings.TrimSpace(a.input) != "" {
		target = targetFor(a.input).name
	}
	if a.busy {
		placeholder = "正在翻译…"
	}
	pane(c, vibrant).Children(func() {
		paneHeader(c, target, func() {
			if a.output != "" && iconButton(c, copyIcon, "复制").Clicked() {
				c.WriteClipboard(a.output)
				c.Toast("已复制译文")
			}
		})
		textPane(c.Key("output"), &a.output).ReadOnly(true).Placeholder(placeholder)
	})
}

// footer shows how the translation goes, and the button that starts or
// stops it.
func (a *app) footer(c *ui.Context) {
	t := c.Theme()
	ui.Row(c).Height(60).Shrink(0).Padding(0, 20).Gap(8).AlignItems(ui.Center).Children(func() {
		switch {
		case a.errMsg != "":
			ui.Icon(c, alertIcon).Size(14, 14).TextColor(t.Danger)
			ui.Text(c, a.errMsg).FontSize(12).TextColor(t.Danger).Grow(1).MinWidth(0).MaxLines(2)
		case a.busy:
			ui.Spinner(c).Label("正在翻译")
			ui.Text(c, "正在翻译…").FontSize(12).TextColor(t.TextMuted).Grow(1)
		default:
			ui.Textf(c, "%d 字符", utf8.RuneCountInString(a.input)).FontSize(12).TextColor(t.TextMuted).Grow(1)
		}
		if a.busy {
			ui.Button(c, "停止").OnClick(a.stop)
			return
		}
		ui.PrimaryButton(c, "").Label("翻译").Gap(8).Disabled(strings.TrimSpace(a.input) == "").OnClick(a.translate).Children(func() {
			ui.Text(c, "翻译").SingleLine()
			ui.Text(c, translateShortcut).SingleLine().Opacity(0.7)
		})
	})
}

// settings is the dialog that sets the DeepSeek API key.
func (a *app) settings(c *ui.Context) {
	t := c.Theme()
	ui.Modal(c, &a.settingsOpen, func() {
		ui.Column(c).Width(380).Gap(16).Children(func() {
			ui.Column(c).Gap(4).Children(func() {
				ui.Text(c, "DeepSeek API Key").FontSize(15).Bold()
				ui.Text(c, "仅保存在本机的系统钥匙串中，只用于请求 DeepSeek 翻译。").FontSize(12).TextColor(t.TextMuted)
			})
			save := ui.TextInput(c, &a.keyDraft).Password().Placeholder("sk-…").Label("API Key").AutoFocus().Submitted()
			ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
				ui.Link(c, "获取 API Key", apiKeysURL).FontSize(12)
				ui.Spacer(c)
				if ui.Button(c, "取消").Clicked() {
					a.settingsOpen = false
				}
				if ui.PrimaryButton(c, "保存").Clicked() {
					save = true
				}
			})
			if !save {
				return
			}
			if err := a.saveSettings(); err != nil {
				c.Toast("保存失败：" + err.Error())
				return
			}
			c.Toast("API Key 已保存")
		})
	})
}

// pane is a rounded panel, which the window's material shows through a
// little.
func pane(c *ui.Context, vibrant bool) ui.Element {
	t := c.Theme()
	fill, border := t.Background, t.Border
	if vibrant {
		fill, border = fill.Alpha(0.55), border.Alpha(0.6)
	}
	return ui.Column(c).Grow(1).Basis(0).MinWidth(0).Radius(12).Background(fill).Border(1, border).
		Shadow(0, 1, 2, 0, ui.RGBA(0, 0, 0, 0.05))
}

// paneHeader shows a pane's language, and its actions at the end.
func paneHeader(c *ui.Context, lang string, actions func()) {
	t := c.Theme()
	ui.Row(c).Height(44).Shrink(0).Padding(0, 8, 0, 12).Gap(2).AlignItems(ui.Center).Children(func() {
		ui.Text(c, lang).FontSize(12).FontWeight(600).TextColor(t.Accent).SingleLine().
			Padding(3, 10).Radius(999).Background(t.Accent.Alpha(0.12))
		ui.Spacer(c)
		actions()
	})
}

// textPane is a text area filling the rest of a pane, without a frame of
// its own.
func textPane(c *ui.Context, text *string) ui.Element {
	return ui.TextAreaBase(c, text).Grow(1).MinHeight(0).FillWidth().Padding(0, 14, 14).FontSize(15).LineHeight(1.5)
}

// iconButton is a borderless button showing an icon, as in a toolbar.
func iconButton(c *ui.Context, svg *ui.SVG, label string) ui.Element {
	t := c.Theme()
	b := ui.ButtonBase(c).Size(28, 28).Radius(6).Center().TextColor(t.TextMuted).Label(label).Tooltip(label)
	if b.Hovered() {
		b.Background(t.SurfaceHover).TextColor(t.Text)
	}
	return b.Children(func() { ui.Icon(c, svg).Size(16, 16) })
}
