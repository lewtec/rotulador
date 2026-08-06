package i18n

import (
	"context"
	"embed"
	"encoding/json"
	"log/slog"
	"net/http"
	"runtime"
	"strconv"
	"strings"
	"sync"

	goi18n "github.com/nicksnyder/go-i18n/v2/i18n"
	"golang.org/x/text/language"
)

//go:embed locales/*.json
var localesFS embed.FS

var (
	bundle        *goi18n.Bundle
	defaultLocal  *goi18n.Localizer
	currentLocale string = "en"

	// Goroutine-local storage for localizers (mold template bridge; removed in templ PR)
	goroutineLocalizers sync.Map // map[uint64]*goi18n.Localizer
)

type localizerKey struct{}

// getGoroutineID returns the current goroutine ID
func getGoroutineID() uint64 {
	var buf [64]byte
	n := runtime.Stack(buf[:], false)
	idField := strings.Fields(strings.TrimPrefix(string(buf[:n]), "goroutine "))[0]
	id, _ := strconv.ParseUint(idField, 10, 64)
	return id
}

func init() {
	bundle = goi18n.NewBundle(language.English)
	bundle.RegisterUnmarshalFunc("json", json.Unmarshal)

	// Load all locale files
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

// SetLanguage sets the current language for translations
func SetLanguage(lang string) {
	currentLocale = lang
	defaultLocal = goi18n.NewLocalizer(bundle, currentLocale)
}

// AddMessage adds a message to the bundle dynamically (useful for YAML config)
func AddMessage(lang, messageID, translation string) error {
	return bundle.AddMessages(language.MustParse(lang), &goi18n.Message{
		ID:    messageID,
		Other: translation,
	})
}

// GetLocalizerFromContext retrieves the localizer from context, or returns default
func GetLocalizerFromContext(ctx context.Context) *goi18n.Localizer {
	if ctx == nil {
		return defaultLocal
	}

	if localizer, ok := ctx.Value(localizerKey{}).(*goi18n.Localizer); ok {
		return localizer
	}
	return defaultLocal
}

// WithLocalizer adds a localizer to the context
func WithLocalizer(ctx context.Context, localizer *goi18n.Localizer) context.Context {
	return context.WithValue(ctx, localizerKey{}, localizer)
}

// GetLocalizerFromRequest creates a localizer based on the Accept-Language header
func GetLocalizerFromRequest(r *http.Request) *goi18n.Localizer {
	acceptLang := r.Header.Get("Accept-Language")

	// Parse Accept-Language header to get preferred languages
	// Format: "en-US,en;q=0.9,pt-BR;q=0.8,pt;q=0.7"
	var langs []string
	if acceptLang != "" {
		parts := strings.Split(acceptLang, ",")
		for _, part := range parts {
			// Remove quality values (;q=0.9)
			lang := strings.TrimSpace(strings.Split(part, ";")[0])
			if lang != "" {
				langs = append(langs, lang)
			}
		}
	}

	// Add default language as fallback
	if len(langs) == 0 {
		langs = []string{currentLocale}
	}

	return goi18n.NewLocalizer(bundle, langs...)
}

// BindLocalizer stores loc for the current goroutine so T works inside mold templates.
// Call the returned function to clear the binding when rendering finishes.
func BindLocalizer(loc *goi18n.Localizer) func() {
	gid := getGoroutineID()
	goroutineLocalizers.Store(gid, loc)
	return func() {
		goroutineLocalizers.Delete(gid)
	}
}

// T translates a message ID using the goroutine-local localizer if available,
// otherwise uses the default localizer
func T(messageID string) string {
	gid := getGoroutineID()
	if loc, ok := goroutineLocalizers.Load(gid); ok {
		if localizer, ok := loc.(*goi18n.Localizer); ok {
			msg, err := localizer.Localize(&goi18n.LocalizeConfig{
				MessageID: messageID,
			})
			if err != nil {
				return messageID
			}
			return msg
		}
	}

	msg, err := defaultLocal.Localize(&goi18n.LocalizeConfig{
		MessageID: messageID,
	})
	if err != nil {
		return messageID
	}
	return msg
}

// LocalizeWithData translates a message with template data
func LocalizeWithData(messageID string, data map[string]interface{}) string {
	msg, err := defaultLocal.Localize(&goi18n.LocalizeConfig{
		MessageID:    messageID,
		TemplateData: data,
	})
	if err != nil {
		return messageID
	}
	return msg
}

// LocalizeWithContext translates a message using the localizer from context
func LocalizeWithContext(ctx context.Context, messageID string) string {
	localizer := GetLocalizerFromContext(ctx)
	msg, err := localizer.Localize(&goi18n.LocalizeConfig{
		MessageID: messageID,
	})
	if err != nil {
		return messageID
	}
	return msg
}

// LocalizeWithContextAndData translates a message with template data using context
func LocalizeWithContextAndData(ctx context.Context, messageID string, data map[string]interface{}) string {
	localizer := GetLocalizerFromContext(ctx)
	msg, err := localizer.Localize(&goi18n.LocalizeConfig{
		MessageID:    messageID,
		TemplateData: data,
	})
	if err != nil {
		return messageID
	}
	return msg
}
