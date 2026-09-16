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
}

func NewDependencies(unitOfWork repository.UnitOfWork, logger *Logger, file *File, messageQueue *MessageQueue, ably *Ably, whatsapp *Whatsapp) *Dependencies {
	return &Dependencies{
		UnitOfWork:   unitOfWork,
		Logger:       logger,
		File:         file,
		MessageQueue: messageQueue,
		Ably:         ably,
		Whatsapp:     whatsapp,
	}
}
