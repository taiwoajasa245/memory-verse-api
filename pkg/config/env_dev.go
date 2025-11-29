//go:build dev

package config

func SelectedEnv() string {
	return ".env.development"
}
