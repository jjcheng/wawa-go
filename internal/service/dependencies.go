package service

import (
	"github.com/jjcheng/wawa-go/internal/repository"
)

type Dependencies struct {
	UnitOfWork      repository.UnitOfWork
	Logger          *Logger
	File            *File
	MessageQueue    *MessageQueue
	Whatsapp        *Whatsapp
	WAMessageStream *WAMessageStream
}

func NewDependencies(unitOfWork repository.UnitOfWork, logger *Logger, file *File, messageQueue *MessageQueue, whatsapp *Whatsapp, waMessageStream *WAMessageStream) *Dependencies {
	return &Dependencies{
		UnitOfWork:      unitOfWork,
		Logger:          logger,
		File:            file,
		MessageQueue:    messageQueue,
		Whatsapp:        whatsapp,
		WAMessageStream: waMessageStream,
	}
}
