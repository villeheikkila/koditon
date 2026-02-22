package tui

func nextScreenForActionInput(ctx *appContext, a action, values []string, breadcrumb string) Screen {
	inputs := append([]string(nil), values...)
	if a.UseCityPicker && len(inputs) == 0 {
		return newCityPickerScreen(ctx, a, inputs, breadcrumb)
	}
	if a.BuildInput != nil {
		if screen := a.BuildInput(ctx, a, inputs, breadcrumb); screen != nil {
			return screen
		}
	}
	if len(a.Prompts) > len(inputs) {
		return newPromptScreen(ctx, a, inputs, len(inputs), breadcrumb)
	}
	return newJobScreen(ctx, a, inputs, breadcrumb)
}
