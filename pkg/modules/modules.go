// Package modules provides module registration and shared module state handling.
package modules

import (
	"fmt"
	"sync"

	"github.com/bwmarrin/discordgo"
	"github.com/robfig/cron/v3"

	"github.com/Luzifer/discord-community/pkg/attributestore"
	"github.com/Luzifer/discord-community/pkg/config"
)

type (
	// Module contains the interface to implement when writing modules
	Module interface {
		// ID returns the configured module instance ID.
		ID() string
		// Initialize stores runtime dependencies and validates module attributes.
		Initialize(args ModuleInitArgs) error
		// Setup performs module setup after initialization.
		Setup() error
	}

	// ModuleInitArgs define the arguments a module is passed during its Initialize
	ModuleInitArgs struct {
		ID    string
		Attrs attributestore.ModuleAttributeStore

		Crontab *cron.Cron
		Discord *discordgo.Session
		Config  *config.File
		Store   *MetaStore
	}

	// ModuleInitFn creates a new Module instance when called
	ModuleInitFn func() Module
)

var (
	moduleRegister     = make(map[string]ModuleInitFn)
	moduleRegisterLock sync.RWMutex
)

// GetModuleByName spawns a new instance of a Module when called
func GetModuleByName(name string) Module {
	moduleRegisterLock.RLock()
	defer moduleRegisterLock.RUnlock()

	mif, ok := moduleRegister[name]
	if !ok {
		return nil
	}

	return mif()
}

// RegisterModule registers a new named module for use with GetModuleByName
func RegisterModule(name string, modInit ModuleInitFn) {
	moduleRegisterLock.Lock()
	defer moduleRegisterLock.Unlock()

	if _, ok := moduleRegister[name]; ok {
		panic(fmt.Errorf("duplicate module register %q", name))
	}

	moduleRegister[name] = modInit
}
