package common

import (
	"crypto/rand"
	"crypto/sha256"
	"fmt"
	"math/big"
	"net/http"
	"os"
	"reflect"
	"strings"

	"github.com/fsnotify/fsnotify"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/spf13/viper"
)

const (
	randomLength = 32
)

func init() {
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnixMs
	log.Logger = zerolog.New(os.Stderr)
	log.Logger = log.With().Logger()
}

var characterRunes = []byte("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789")

// RandomString returns a random string length of argument n.
func RandomString(n int) (string, error) {
	b := make([]byte, n)
	for i := range b {
		num, err := rand.Int(rand.Reader, big.NewInt(int64(len(characterRunes))))
		if err != nil {
			return "", err
		}
		b[i] = characterRunes[num.Int64()]
	}

	return string(b), nil
}

// RandomToken returns random sha256 string.
func RandomToken() (string, error) {
	hash := sha256.New()
	r, err := RandomString(randomLength)
	if err != nil {
		return "", err
	}
	hash.Write([]byte(r))
	bs := hash.Sum(nil)
	return fmt.Sprintf("%x", bs), nil
}

// IsHTTPS is a helper function that evaluates the http.Request
// and returns True if the Request uses HTTPS. It is able to detect,
// using the X-Forwarded-Proto, if the original request was HTTPS and
// routed through a reverse proxy with SSL termination.
func IsHTTPS(r *http.Request) bool {
	switch {
	case r.URL.Scheme == "https":
		return true
	case r.TLS != nil:
		return true
	case strings.HasPrefix(r.Proto, "HTTPS"):
		return true
	case r.Header.Get("X-Forwarded-Proto") == "https":
		return true
	default:
		return false
	}
}

// MinUint calculates Min from a, b.
func MinUint(a, b uint) uint {
	if a < b {
		return a
	}
	return b
}

// EnsureDot ensures that string has ending dot.
func EnsureDot(input string) string {
	if !strings.HasSuffix(input, ".") {
		return fmt.Sprintf("%s.", input)
	}
	return input
}

// RemoveDot removes suffix dot from string if it exists.
func RemoveDot(input string) string {
	if strings.HasSuffix(input, ".") {
		return input[:len(input)-1]
	}
	return input
}

// LoadAndListenConfig loads config file to struct and listen changes in it.
func LoadAndListenConfig(path string, obj interface{}, onUpdate func(oldObj interface{})) error {
	v := viper.New()
	v.SetConfigFile(path)
	if err := v.ReadInConfig(); err != nil {
		return fmt.Errorf("unable to read config: %w", err)
	}
	if err := v.Unmarshal(&obj); err != nil {
		return fmt.Errorf("unable to marshal config: %w", err)
	}
	log.Info().
		Str("path", v.ConfigFileUsed()).
		Msg("config loaded")
	v.WatchConfig()
	v.OnConfigChange(func(e fsnotify.Event) {
		log.Info().
			Str("path", e.Name).
			Msg("config reloaded")
		oldObj := reflect.Indirect(reflect.ValueOf(obj)).Interface()
		if err := v.Unmarshal(&obj); err != nil {
			log.Fatal().
				Str("path", e.Name).
				Msgf("unable to marshal config: %v", err)
		}
		if onUpdate != nil {
			onUpdate(oldObj)
		}
	})
	return nil
}
