package app

func NormalizeLocale(locale string) string {
	for _, supportedLocale := range UranusInstance.Config.SupportedLanguages {
		if locale == supportedLocale {
			return locale
		}
	}

	return "de"
}
