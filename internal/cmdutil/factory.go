// Package cmdutil holds helpers shared by all commands.
package cmdutil

import (
	"io"
	"os"

	"github.com/spf13/viper"

	"github.com/coupler-io/coupler-io-cli/internal/client"
	"github.com/coupler-io/coupler-io-cli/internal/config"
	"github.com/coupler-io/coupler-io-cli/internal/output"
)

type IOStreams struct {
	In     io.Reader
	Out    io.Writer
	ErrOut io.Writer
}

// Factory carries the dependencies commands need, so tests can swap them.
type Factory struct {
	// Viper holds the resolved settings. Each Factory owns its instance, so
	// tests never share configuration state.
	Viper     *viper.Viper
	IOStreams *IOStreams
	// ConfigErr is the error from LoadConfig. Commands that do not need the
	// config file (configure, completion) still run when it is set.
	ConfigErr      error
	ClientOverride *client.Client // for testing
}

func NewFactory() *Factory {
	return &Factory{
		Viper: viper.New(),
		IOStreams: &IOStreams{
			In:     os.Stdin,
			Out:    os.Stdout,
			ErrOut: os.Stderr,
		},
	}
}

// LoadConfig reads ~/.coupler.yaml and COUPLER_* env into f.Viper and records
// any failure in f.ConfigErr.
func (f *Factory) LoadConfig() {
	f.ConfigErr = config.Load(f.Viper)
}

// NewClient builds an API client from the resolved token and API URL.
func (f *Factory) NewClient() (*client.Client, error) {
	if f.ClientOverride != nil {
		return f.ClientOverride, nil
	}
	token, err := config.RequireAPIToken(f.Viper)
	if err != nil {
		return nil, err
	}
	return client.New(token, client.WithBaseURL(f.Viper.GetString(config.KeyAPIURL))), nil
}

// Format returns the selected output format, defaulting to table.
func (f *Factory) Format() output.Format {
	format, err := output.ParseFormat(f.Viper.GetString(config.KeyOutput))
	if err != nil {
		return output.FormatTable
	}
	return format
}
