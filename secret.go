package main

import (
	"fmt"
	"syscall"

	log "github.com/sirupsen/logrus"
	"github.com/zalando/go-keyring"
	"golang.org/x/term"
)

func promptForSecret(secret string) string {
	service := "tired"
	var secretValue string
	var err error

	secretValue, err = keyring.Get(service, secret)

	if err == nil && secretValue != "" {
		clog.WithFields(log.Fields{"secret": secret}).Info("Got secret value from keyring.")
		return secretValue
	}

	fmt.Printf("New '%v' value: ", secret)
	bytespw, _ := term.ReadPassword(int(syscall.Stdin))
	secretValue = string(bytespw)
	fmt.Print("\n")

	err = keyring.Set(service, secret, secretValue)

	if err != nil {
		clog.WithFields(log.Fields{"secret": secret, "error": err}).Fatal("Can't save password to keyring.")
	}

	clog.WithFields(log.Fields{"secret": secret}).Info("Secret saved to keyring.")
	return secretValue
}
