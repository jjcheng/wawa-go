package service

import (
	"github.com/jjcheng/wawa-go/internal/repository"
)

type Dependencies struct {
	UnitOfWork   repository.UnitOfWork
	Logger       *Logger
	File         *File
	MessageQueue *MessageQueue
	Ably         *Ably
	Whatsapp     *Whatsapp
	Facebook     *Facebook
	AuthCache    *AuthCache
	Cache        *Cache
	Cloudflare   *Cloudflare
	TypeSafe     *TypeSafe
	LLM          *LLM
}

func NewDependencies(unitOfWork repository.UnitOfWork, logger *Logger) *Dependencies {
	return &Dependencies{
		UnitOfWork:   unitOfWork,
		Logger:       logger,
		File:         NewFileService(logger),
		MessageQueue: NewMessageQueue(logger),
		Ably:         NewAbly(logger),
		Whatsapp:     NewWhatsapp(logger),
		Facebook:     NewFacebook(logger),
		AuthCache:    NewAuthCache(),
		Cache:        NewCache(unitOfWork),
		Cloudflare:   NewCloudflare(logger),
		TypeSafe:     NewTypeSafe(logger),
		LLM:          NewLLM(logger),
	}
}
