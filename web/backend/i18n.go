package main

import (
	"fmt"
	"os"
	"strings"
)

// Language represents the supported languages
type Language string

const (
	LanguageEnglish            Language = "en"
	LanguageChinese            Language = "zh"
	LanguageTraditionalChinese Language = "zh-TW"
)

// current language (default: English)
var currentLang Language = LanguageEnglish

// TranslationKey represents a translation key used for i18n
type TranslationKey string

const (
	AppTooltip         TranslationKey = "AppTooltip"
	MenuOpen           TranslationKey = "MenuOpen"
	MenuOpenTooltip    TranslationKey = "MenuOpenTooltip"
	MenuAbout          TranslationKey = "MenuAbout"
	MenuAboutTooltip   TranslationKey = "MenuAboutTooltip"
	MenuVersion        TranslationKey = "MenuVersion"
	MenuVersionTooltip TranslationKey = "MenuVersionTooltip"
	MenuGitHub         TranslationKey = "MenuGitHub"
	MenuDocs           TranslationKey = "MenuDocs"
	MenuRestart        TranslationKey = "MenuRestart"
	MenuRestartTooltip TranslationKey = "MenuRestartTooltip"
	MenuQuit           TranslationKey = "MenuQuit"
	MenuQuitTooltip    TranslationKey = "MenuQuitTooltip"
	Exiting            TranslationKey = "Exiting"
	DocUrl             TranslationKey = "DocUrl"
)

// Translation tables
// Chinese translations intentionally contain Han script
//
//nolint:gosmopolitan
var translations = map[Language]map[TranslationKey]string{
	LanguageEnglish: {
		AppTooltip:         "%s - Web Console",
		MenuOpen:           "Open Console",
		MenuOpenTooltip:    "Open PicoClaw console in browser",
		MenuAbout:          "About",
		MenuAboutTooltip:   "About PicoClaw",
		MenuVersion:        "Version: %s",
		MenuVersionTooltip: "Current version number",
		MenuGitHub:         "GitHub",
		MenuDocs:           "Documentation",
		MenuRestart:        "Restart Service",
		MenuRestartTooltip: "Restart Gateway service",
		MenuQuit:           "Quit",
		MenuQuitTooltip:    "Exit PicoClaw",
		Exiting:            "Exiting PicoClaw...",
		DocUrl:             "https://docs.picoclaw.io/docs/",
	},
	LanguageChinese: {
		AppTooltip:         "%s - Web Console",
		MenuOpen:           "打开控制台",
		MenuOpenTooltip:    "在浏览器中打开 PicoClaw 控制台",
		MenuAbout:          "关于",
		MenuAboutTooltip:   "关于 PicoClaw",
		MenuVersion:        "版本: %s",
		MenuVersionTooltip: "当前版本号",
		MenuGitHub:         "GitHub",
		MenuDocs:           "文档",
		MenuRestart:        "重启服务",
		MenuRestartTooltip: "重启核心服务",
		MenuQuit:           "退出",
		MenuQuitTooltip:    "退出 PicoClaw",
		Exiting:            "正在退出 PicoClaw...",
		DocUrl:             "https://docs.picoclaw.io/zh-Hans/docs/",
	},
	LanguageTraditionalChinese: {
		AppTooltip:         "%s - Web 主控台",
		MenuOpen:           "開啟主控台",
		MenuOpenTooltip:    "在瀏覽器中開啟 PicoClaw 主控台",
		MenuAbout:          "關於",
		MenuAboutTooltip:   "關於 PicoClaw",
		MenuVersion:        "版本 : %s",
		MenuVersionTooltip: "目前的版本號碼",
		MenuGitHub:         "GitHub",
		MenuDocs:           "說明文件",
		MenuRestart:        "重新啟動服務",
		MenuRestartTooltip: "重新啟動閘道服務",
		MenuQuit:           "結束",
		MenuQuitTooltip:    "結束 PicoClaw",
		Exiting:            "正在結束 PicoClaw...",
		DocUrl:             "https://docs.picoclaw.io/zh-TW/docs/",
	},
}

// SetLanguage sets the current language
func SetLanguage(lang string) {
	lang = strings.ToLower(strings.TrimSpace(lang))

	locale := lang
	if idx := strings.IndexByte(locale, '.'); idx > 0 {
		locale = locale[:idx]
	}
	if locale == "zh-tw" || locale == "zh_tw" {
		currentLang = LanguageTraditionalChinese
		return
	}

	// Extract language code before first underscore or dot
	// e.g., "en_US.UTF-8" -> "en", "zh_CN" -> "zh"
	if idx := strings.IndexAny(lang, "_."); idx > 0 {
		lang = lang[:idx]
	}

	if lang == "zh" || lang == "zh-cn" || lang == "chinese" {
		currentLang = LanguageChinese
	} else {
		currentLang = LanguageEnglish
	}
}

// GetLanguage returns the current language
func GetLanguage() Language {
	return currentLang
}

// T translates a key to the current language
func T(key TranslationKey, args ...any) string {
	if trans, ok := translations[currentLang][key]; ok {
		if len(args) > 0 {
			return fmt.Sprintf(trans, args...)
		}
		return trans
	}
	return string(key)
}

// Initialize i18n from environment variable
func init() {
	if lang := os.Getenv("LANG"); lang != "" {
		SetLanguage(lang)
	}
}
