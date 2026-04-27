package auth

import "strings"

type OAuthClientMetadata struct {
	DisplayName         string
	LogoURL             string
	IsFirstParty        bool
	ShowInConnectedApps bool
}

var oauthClientMetadataByID = map[string]OAuthClientMetadata{
	"koditon-apple": {
		DisplayName:         "Koditon iPhone App",
		IsFirstParty:        true,
		ShowInConnectedApps: false,
	},
	"koditon-cli": {
		DisplayName:         "Koditon CLI",
		IsFirstParty:        true,
		ShowInConnectedApps: true,
	},
	"koditon-web": {
		DisplayName:         "Koditon Web",
		IsFirstParty:        true,
		ShowInConnectedApps: false,
	},
}

func OAuthClientMetadataForID(clientID string) (OAuthClientMetadata, bool) {
	metadata, ok := oauthClientMetadataByID[strings.TrimSpace(clientID)]
	return metadata, ok
}
