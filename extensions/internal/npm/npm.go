// Package npm implements functions for iteracting with the npm registry
package npm

import (
	"context"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/askgitdev/askgit/extensions/options"
	"github.com/pkg/errors"
	"go.riyazali.net/sqlite"
	"go.uber.org/zap"
)

const (
	BaseURL = "https://registry.npmjs.org"
)

type Client struct {
	httpClient *http.Client
	logger     *zap.Logger
}

// NewClient creates a new API client from an *http.Client. Pass nil to use http.DefaultClient
func NewClient(httpClient *http.Client, logger *zap.Logger) *Client {
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	if logger == nil {
		logger = zap.NewNop()
	}
	return &Client{httpClient, logger}
}

// GetPackage makes an HTTP request to https://registry.npmjs.org/<<packageName>> and returns the JSON response
func (c *Client) GetPackage(ctx context.Context, packageName string) ([]byte, error) {
	path := fmt.Sprintf("%s/%s", BaseURL, packageName)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}

	c.logger.Sugar().Infof("making GET request: %s", path)

	res, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}

	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}

	return body, nil
}

// GetPackageVersion makes an HTTP request to https://registry.npmjs.org/<<packageName>>/<<version>> and returns the JSON response
func (c *Client) GetPackageVersion(ctx context.Context, packageName, version string) ([]byte, error) {
	path := fmt.Sprintf("%s/%s/%s", BaseURL, packageName, version)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}

	c.logger.Sugar().Infof("making GET request: %s", path)

	res, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}

	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}

	return body, nil
}

// Register registers npm API related functionality as a SQLite extension
func Register(ext *sqlite.ExtensionApi, opt *options.Options) (_ sqlite.ErrorCode, err error) {
	var fns = map[string]sqlite.Function{
		"npm_get_package": &GetPackage{NewClient(opt.NPMHttpClient, opt.Logger)},
	}

	for name, fn := range fns {
		if err = ext.CreateFunction(name, fn); err != nil {
			return sqlite.SQLITE_ERROR, errors.Wrapf(err, "failed to register %q function", name)
		}
	}

	return sqlite.SQLITE_OK, nil
}
