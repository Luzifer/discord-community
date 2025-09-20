package modules

import (
	"sync"

	"github.com/Luzifer/discord-community/pkg/attributestore"
	"github.com/Luzifer/discord-community/pkg/config"
	"github.com/bwmarrin/discordgo"
	"github.com/pkg/errors"
	"github.com/robfig/cron/v3"
)

var (
	moduleRegister     = map[string]ModuleInitFn{}
	moduleRegisterLock sync.RWMutex
)

type (
	// Module contains the interface to implement when writing modules
	Module interface {
		ID() string
		Initialize(args ModuleInitArgs) error
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
		panic(errors.Errorf("duplicate module register %q", name))
	}

	moduleRegister[name] = modInit
}
