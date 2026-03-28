package config

import (
	"os"
	"reflect"
	"strconv"
	"strings"
)

func applyProviderEnvOverrides(cfg *Config) {
	v := reflect.ValueOf(&cfg.Providers).Elem()
	t := v.Type()
	for i := range t.NumField() {
		name := strings.ToUpper(strings.Split(t.Field(i).Tag.Get("json"), ",")[0])
		applyProviderStructEnv(v.Field(i), t.Field(i).Type, name)
	}
}

func applyProviderStructEnv(v reflect.Value, t reflect.Type, name string) {
	if t.Kind() == reflect.Struct {
		for i := range t.NumField() {
			fv, ft := v.Field(i), t.Field(i)
			if ft.Type.Kind() == reflect.Struct {
				applyProviderStructEnv(fv, ft.Type, name)
				continue
			}
			envKey := strings.ReplaceAll(ft.Tag.Get("env"), "{{.Name}}", name)
			if envKey == "" {
				continue
			}
			raw, ok := os.LookupEnv(envKey)
			if !ok {
				continue
			}
			switch fv.Kind() {
			case reflect.String:
				fv.SetString(raw)
			case reflect.Int:
				if n, err := strconv.Atoi(raw); err == nil {
					fv.SetInt(int64(n))
				}
			case reflect.Bool:
				if b, err := strconv.ParseBool(raw); err == nil {
					fv.SetBool(b)
				}
			}
		}
	}
}
