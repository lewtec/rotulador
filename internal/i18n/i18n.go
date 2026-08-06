package i18n

import (
	"context"
	"embed"
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"

	goi18n "github.com/nicksnyder/go-i18n/v2/i18n"
	"golang.org/x/text/language"
)

//go:embed locales/*.json
var localesFS embed.FS

var (
	bundle        *goi18n.Bundle
	defaultLocal  *goi18n.Localizer
	currentLocale string = "en"
)

type localizerKey struct{}

func init() {
	bundle = goi18n.NewBundle(language.English)
	bundle.RegisterUnmarshalFunc("json", json.Unmarshal)

	locales := []string{"en", "pt-BR"}
	for _, locale := range locales {
		data, err := localesFS.ReadFile("locales/" + locale + ".json")
		if err != nil {
			slog.Warn("i18n: WARNING - failed to read locale file", "locale", locale, "err", err)
			continue
		}

		_, err = bundle.ParseMessageFileBytes(data, locale+".json")
		if err != nil {
			slog.Warn("i18n: failed to parse locale file", "locale", locale, "err", err)
			continue
		}
	}

	defaultLocal = goi18n.NewLocalizer(bundle, currentLocale)
}

// SetLanguage sets the default language for translations outside a request context.
func SetLanguage(lang string) {
	currentLocale = lang
	defaultLocal = goi18n.NewLocalizer(bundle, currentLocale)
}

// AddMessage adds a message to the bundle dynamically (useful for YAML config).
func AddMessage(lang, messageID, translation string) error {
	return bundle.AddMessages(language.MustParse(lang), &goi18n.Message{
		ID:    messageID,
		Other: translation,
	})
}

// Get returns the localizer from ctx, or the default localizer.
func Get(ctx context.Context) *goi18n.Localizer {
	if ctx == nil {
		return defaultLocal
	}
	if localizer, ok := ctx.Value(localizerKey{}).(*goi18n.Localizer); ok {
		return localizer
	}
	return defaultLocal
}

// With attaches a localizer to ctx.
func With(ctx context.Context, localizer *goi18n.Localizer) context.Context {
	return context.WithValue(ctx, localizerKey{}, localizer)
}

// FromRequest builds a localizer from Accept-Language.
func FromRequest(r *http.Request) *goi18n.Localizer {
	acceptLang := r.Header.Get("Accept-Language")

	var langs []string
	if acceptLang != "" {
		parts := strings.Split(acceptLang, ",")
		for _, part := range parts {
			lang := strings.TrimSpace(strings.Split(part, ";")[0])
			if lang != "" {
				langs = append(langs, lang)
			}
		}
	}
	if len(langs) == 0 {
		langs = []string{currentLocale}
	}
	return goi18n.NewLocalizer(bundle, langs...)
}

// T translates messageID using the localizer on ctx.
func T(ctx context.Context, messageID string) string {
	msg, err := Get(ctx).Localize(&goi18n.LocalizeConfig{MessageID: messageID})
	if err != nil {
		return messageID
	}
	return msg
}

// TData translates messageID with template data using the localizer on ctx.
func TData(ctx context.Context, messageID string, data map[string]interface{}) string {
	msg, err := Get(ctx).Localize(&goi18n.LocalizeConfig{
		MessageID:    messageID,
		TemplateData: data,
	})
	if err != nil {
		return messageID
	}
	return msg
}
