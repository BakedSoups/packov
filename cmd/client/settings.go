package main

type clientSettings struct {
	MouseAim      bool
	Controller    bool
	DamageNumbers bool
	ScreenShake   bool
}

func defaultSettings() clientSettings {
	return clientSettings{
		MouseAim:      true,
		Controller:    true,
		DamageNumbers: true,
		ScreenShake:   true,
	}
}
