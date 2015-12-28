package utils

import (
	log "github.com/Sirupsen/logrus"
	"github.com/spf13/viper"
)

// AuthorizedUsers contains Tat Authorized Users
var AuthorizedUsers []string

// IsValidTatUser checks user on tat
func IsValidTatUser(tatUsername, tatPassword string) bool {

	if !ArrayContains(AuthorizedUsers, tatUsername) {
		log.Warnf("User %s not authorized from configuration, flag --authorized-users. Authorized:%+v", tatUsername, AuthorizedUsers)
		return false
	}

	if _, err := GetWantBody(
		viper.GetString("url_tat_engine"),
		"/user/me",
		tatUsername,
		tatPassword,
		"",
	); err != nil {
		return false
	}
	return true
}
