package utils

import "os/user"

// GetUsername returns the current username.
func GetUsername() (string, error) {
	user, err := user.Current()
	if err != nil {
		return "", err
	}
	return user.Username, nil
}
