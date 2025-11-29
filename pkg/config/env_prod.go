//go:build prod

package config

func SelectedEnv() string {
	return ".env.production"
}
