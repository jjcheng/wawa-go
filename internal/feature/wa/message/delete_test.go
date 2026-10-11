package feature_wa_message

import (
	"context"
	"errors"
	"net/http"
	"testing"

	dao_wa "github.com/jjcheng/wawa-go/internal/dao/wa"
	"github.com/jjcheng/wawa-go/internal/dto"
	dto_account "github.com/jjcheng/wawa-go/internal/dto/account"
	dto_wa "github.com/jjcheng/wawa-go/internal/dto/wa"
	"github.com/jjcheng/wawa-go/internal/repository"
	"github.com/jjcheng/wawa-go/internal/service"
	"gorm.io/gorm"
)

type deleteMessageRepository struct {
	repository.WAMessageRepository
	message  *dao_wa.Message
	getError error
	deleted  bool
}

func (r *deleteMessageRepository) GetById(context.Context, int32) (*dao_wa.Message, error) {
	return r.message, r.getError
}

func (r *deleteMessageRepository) DeleteById(_ context.Context, id int32) error {
	r.deleted = id == r.message.Id
	return nil
}

type deleteMessageUnitOfWork struct {
	repository.UnitOfWork
	messages   *deleteMessageRepository
	committed  bool
	rolledBack bool
}

func (u *deleteMessageUnitOfWork) WAMessageRepository() repository.WAMessageRepository {
	return u.messages
}

func (u *deleteMessageUnitOfWork) BeginTransaction() repository.UnitOfWork { return u }
func (u *deleteMessageUnitOfWork) CommitTransaction() error {
	u.committed = true
	return nil
}
func (u *deleteMessageUnitOfWork) Rollback() { u.rolledBack = true }

func TestDeleteValidateAndAPISettings(t *testing.T) {
	for _, id := range []int32{-1, 0, 1} {
		request := Delete{Id: id}
		if got := len(request.Validate()) > 0; got != (id <= 0) {
			t.Fatalf("id=%d: unexpected validation result", id)
		}
	}
	settings := (Delete{}).APISettings()
	if settings.Method != http.MethodDelete || settings.Path != "/v1/wa/messages/:id" || !settings.Auth {
		t.Fatalf("unexpected API settings: %+v", settings)
	}
}

func TestDeleteHandle(t *testing.T) {
	user := &dto_account.User{WA: &dto_account.UserWA{
		PhoneNumbers: []dto_wa.PhoneNumber{{DTOBase: dto.DTOBase{Id: 42}}},
	}}
	for _, tc := range []struct {
		name       string
		user       *dto_account.User
		phoneID    int32
		getError   error
		attachment string
		status     int
		deleted    bool
		committed  bool
		rolledBack bool
	}{
		{name: "unauthenticated", status: http.StatusForbidden},
		{name: "no WA access", user: &dto_account.User{}, status: http.StatusUnauthorized},
		{name: "not found", user: user, getError: gorm.ErrRecordNotFound, status: http.StatusNotFound},
		{name: "lookup failed", user: user, getError: errors.New("database unavailable"), status: http.StatusInternalServerError},
		{name: "not assigned", user: user, phoneID: 43, status: http.StatusUnauthorized},
		{name: "no attachment", user: user, phoneID: 42, status: http.StatusOK, deleted: true, committed: true},
		{name: "attachment deletion failed", user: user, phoneID: 42, attachment: "https://external.example/media.mp4", status: http.StatusInternalServerError, deleted: true, rolledBack: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			messages := &deleteMessageRepository{
				message:  &dao_wa.Message{PhoneNumberId: tc.phoneID, AttachmentURL: tc.attachment},
				getError: tc.getError,
			}
			messages.message.Id = 1
			unitOfWork := &deleteMessageUnitOfWork{messages: messages}
			response := (Delete{Id: 1}).Handle(context.Background(), tc.user, &service.Dependencies{
				UnitOfWork: unitOfWork, File: &service.File{},
			})
			if response.StatusCode != tc.status || messages.deleted != tc.deleted ||
				unitOfWork.committed != tc.committed || unitOfWork.rolledBack != tc.rolledBack {
				t.Fatalf("status=%d deleted=%v committed=%v rolledBack=%v", response.StatusCode, messages.deleted, unitOfWork.committed, unitOfWork.rolledBack)
			}
		})
	}
}
