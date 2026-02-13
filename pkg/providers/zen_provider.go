package providers

func init() {
	RegisterProvider(providerRegistration{
		Name:          "zen",
		ModelPrefixes: []string{"zen/"},
		Creator:       zenCreator,
	})
}
