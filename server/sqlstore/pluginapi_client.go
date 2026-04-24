package sqlstore

import (
	"database/sql"

	"github.com/mattermost/mattermost/server/public/plugin"
	"github.com/mattermost/mattermost/server/public/pluginapi"
)

// StoreAPI is the interface exposing the underlying database, provided by pluginapi.
type StoreAPI interface {
	GetMasterDB() (*sql.DB, error)
	DriverName() string
}

// PluginAPIClient wraps the store and plugin API interfaces needed by the SQL store.
type PluginAPIClient struct {
	Store StoreAPI
	API   plugin.API
}

// NewClient creates a PluginAPIClient from a pluginapi.Client and the raw plugin API.
func NewClient(client *pluginapi.Client, api plugin.API) PluginAPIClient {
	return PluginAPIClient{
		Store: client.Store,
		API:   api,
	}
}
