package translate

import (
	"log"

	"github.com/bregydoc/gtranslate"
)

func TranslateText(text string, targetLang string) string {
	if targetLang == "zh" || targetLang == "" {
		return text // Already in Chinese
	}

	translated, err := gtranslate.TranslateWithParams(
		text,
		gtranslate.TranslationParams{
			From: "zh",
			To:   targetLang,
		},
	)

	if err != nil {
		log.Printf("Translation error (to %s): %v", targetLang, err)
		return text // Fallback to original
	}

	return translated
}
