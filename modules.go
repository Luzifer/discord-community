package main

import (
	"sync"

	"github.com/bwmarrin/discordgo"
	"github.com/pkg/errors"
	"github.com/robfig/cron/v3"
)

var (
	moduleRegister     = map[string]moduleInitFn{}
	moduleRegisterLock sync.RWMutex
)

type (
	module interface {
		Initialize(crontab *cron.Cron, discord *discordgo.Session, attrs moduleAttributeStore) error
		Setup() error
	}

	moduleInitFn func() module
)

func GetModuleByName(name string) module {
	moduleRegisterLock.RLock()
	defer moduleRegisterLock.RUnlock()

	mif, ok := moduleRegister[name]
	if !ok {
		return nil
	}

	return mif()
}

func RegisterModule(name string, modInit moduleInitFn) {
	moduleRegisterLock.Lock()
	defer moduleRegisterLock.Unlock()

	if _, ok := moduleRegister[name]; ok {
		panic(errors.Errorf("duplicate module register %q", name))
	}

	moduleRegister[name] = modInit
}
