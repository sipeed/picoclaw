package main

import "testing"

func TestSetLanguage(t *testing.T) {
	t.Cleanup(func() {
		currentLang = LanguageEnglish
	})

	tests := []struct {
		name string
		lang string
		want Language
	}{
		{name: "English", lang: "en", want: LanguageEnglish},
		{name: "English POSIX", lang: "en_US.UTF-8", want: LanguageEnglish},
		{name: "Simplified Chinese", lang: "zh-CN", want: LanguageChinese},
		{name: "Simplified Chinese POSIX", lang: "zh_CN.UTF-8", want: LanguageChinese},
		{name: "Traditional Chinese", lang: "zh-TW", want: LanguageTraditionalChinese},
		{name: "Traditional Chinese POSIX", lang: "zh_TW.UTF-8", want: LanguageTraditionalChinese},
		{name: "Other Chinese POSIX region", lang: "zh_HK.UTF-8", want: LanguageChinese},
		{name: "Unknown", lang: "fr", want: LanguageEnglish},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			SetLanguage(tt.lang)
			if got := GetLanguage(); got != tt.want {
				t.Fatalf("GetLanguage() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestTraditionalChineseTranslations(t *testing.T) {
	previous := currentLang
	t.Cleanup(func() {
		currentLang = previous
	})

	SetLanguage("zh-TW")

	if got, want := T(MenuOpen), "開啟主控台"; got != want {
		t.Fatalf("T(MenuOpen) = %q, want %q", got, want)
	}
	if got, want := T(MenuVersion, "1.2.3"), "版本 : 1.2.3"; got != want {
		t.Fatalf("T(MenuVersion) = %q, want %q", got, want)
	}
	if got, want := T(DocUrl), "https://docs.picoclaw.io/zh-TW/docs/"; got != want {
		t.Fatalf("T(DocUrl) = %q, want %q", got, want)
	}
}
