package feature_cli

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"strings"

	"github.com/google/uuid"
	"github.com/jjcheng/wawa-go/internal/cfg"
	"github.com/jjcheng/wawa-go/internal/helper"
	"github.com/jjcheng/wawa-go/internal/service"
	"github.com/jjcheng/wawa-go/internal/setup"
	"gorm.io/gorm"
)

// BackfillEncryption fills missing encrypted and keyed-hash columns without
// clearing legacy plaintext columns. It is safe to run repeatedly.
func BackfillEncryption(ctx context.Context) error {
	keys := cfg.Default().Site.GlobalKeys
	if keys == nil {
		return errors.New("encryption keys are not configured")
	}
	uow, err := setup.SetupDatabase(cfg.Default().Database.DSN(), service.NewLogger())
	if err != nil {
		return err
	}
	db := uow.DB().WithContext(ctx)

	backfillers := []func(*gorm.DB) (int, error){
		backfillUsers,
		backfillCustomers,
		backfillPhoneNumbers,
		backfillBusinessPortfolios,
		backfillMessages,
		backfillMessageStatuses,
		backfillCampaignRecipients,
	}
	total := 0
	for _, backfill := range backfillers {
		updated, err := backfill(db)
		if err != nil {
			return err
		}
		total += updated
	}
	log.Printf("encryption backfill completed: %d rows updated", total)
	return nil
}

func BackfillMessages(ctx context.Context) error {
	keys := cfg.Default().Site.GlobalKeys
	if keys == nil {
		return errors.New("encryption keys are not configured")
	}
	uow, err := setup.SetupDatabase(cfg.Default().Database.DSN(), service.NewLogger())
	if err != nil {
		return err
	}
	updated, err := backfillMessages(uow.DB().WithContext(ctx))
	if err != nil {
		return err
	}
	log.Printf("message encryption backfill completed: %d rows updated", updated)
	return nil
}

func encryptBackfillValue(value, aad string) (string, error) {
	if strings.TrimSpace(value) == "" {
		return "", nil
	}
	keys := cfg.Default().Site.GlobalKeys
	encrypted, err := helper.EncryptSecret([]byte(value), keys, aad)
	if err != nil {
		return "", err
	}
	encoded, err := json.Marshal(encrypted)
	if err != nil {
		return "", err
	}
	return string(encoded), nil
}

func hashBackfillValue(value string) (string, error) {
	if strings.TrimSpace(value) == "" {
		return "", nil
	}
	return helper.HashSecretHex(value, cfg.Default().Site.GlobalKeys.HMACKey)
}

func applyBackfillUpdate(db *gorm.DB, table string, id int32, values map[string]any) error {
	if len(values) == 0 {
		return nil
	}
	if err := db.Table(table).Where("id = ?", id).Updates(values).Error; err != nil {
		return fmt.Errorf("update %s id %d: %w", table, id, err)
	}
	return nil
}

type userBackfillRow struct {
	Id                   int32
	EncryptionID         string
	PhoneNumber          string
	PhoneNumberHash      string
	PhoneNumberEncrypted string
	Email                string
	EmailHash            string
	EmailEncrypted       string
}

func backfillUsers(db *gorm.DB) (int, error) {
	var rows []userBackfillRow
	if err := db.Raw("SELECT id, encryption_id, phone_number, phone_number_hash, phone_number_encrypted, email, email_hash, email_encrypted FROM account.users").Scan(&rows).Error; err != nil {
		return 0, err
	}
	updated := 0
	for _, row := range rows {
		values := map[string]any{}
		if row.EncryptionID == "" {
			row.EncryptionID = uuid.NewString()
			values["encryption_id"] = row.EncryptionID
		}
		phoneHash := row.PhoneNumberHash
		if row.PhoneNumber != "" {
			var err error
			phoneHash, err = hashBackfillValue(row.PhoneNumber)
			if err != nil {
				return updated, err
			}
			values["phone_number_hash"] = phoneHash
		}
		if row.PhoneNumber != "" {
			encrypted, err := encryptBackfillValue(row.PhoneNumber, "account.users:phone_number:"+row.EncryptionID)
			if err != nil {
				return updated, err
			}
			values["phone_number_encrypted"] = encrypted
		}
		emailHash := row.EmailHash
		if row.Email != "" {
			var err error
			emailHash, err = hashBackfillValue(row.Email)
			if err != nil {
				return updated, err
			}
			values["email_hash"] = emailHash
		}
		if row.Email != "" {
			encrypted, err := encryptBackfillValue(row.Email, "account.users:email:"+row.EncryptionID)
			if err != nil {
				return updated, err
			}
			values["email_encrypted"] = encrypted
		}
		if err := applyBackfillUpdate(db, "account.users", row.Id, values); err != nil {
			return updated, err
		}
		if len(values) > 0 {
			updated++
		}
	}
	return updated, nil
}

type customerBackfillRow struct {
	Id                           int32
	Token                        string
	PhoneNumber                  string
	PhoneNumberHash              string
	PhoneNumberEncrypted         string
	WAId                         string
	WAIdHash                     string
	WAIdEncrypted                string
	ImportedPhoneNumber          string
	ImportedPhoneNumberHash      string
	ImportedPhoneNumberEncrypted string
	AdditionalData               []byte
	AdditionalDataEncrypted      string
}

func backfillCustomers(db *gorm.DB) (int, error) {
	var rows []customerBackfillRow
	if err := db.Raw("SELECT id, token, phone_number, phone_number_hash, phone_number_encrypted, wa_id, wa_id_hash, wa_id_encrypted, imported_phone_number, imported_phone_number_hash, imported_phone_number_encrypted, additional_data, additional_data_encrypted FROM customer.customers").Scan(&rows).Error; err != nil {
		return 0, err
	}
	updated := 0
	for _, row := range rows {
		values := map[string]any{}
		if row.Token == "" {
			row.Token = uuid.NewString()
			values["token"] = row.Token
		}
		if row.PhoneNumber != "" {
			hash, err := hashBackfillValue(row.PhoneNumber)
			if err != nil {
				return updated, err
			}
			values["phone_number_hash"] = hash
		}
		if row.PhoneNumber != "" {
			encrypted, err := encryptBackfillValue(row.PhoneNumber, "customer.customers:phone_number:"+row.Token)
			if err != nil {
				return updated, err
			}
			values["phone_number_encrypted"] = encrypted
		}
		if row.WAId != "" {
			hash, err := hashBackfillValue(row.WAId)
			if err != nil {
				return updated, err
			}
			values["wa_id_hash"] = hash
		}
		if row.WAId != "" {
			encrypted, err := encryptBackfillValue(row.WAId, "customer.customers:wa_id:"+row.Token)
			if err != nil {
				return updated, err
			}
			values["wa_id_encrypted"] = encrypted
		}
		if row.ImportedPhoneNumber != "" {
			hash, err := hashBackfillValue(row.ImportedPhoneNumber)
			if err != nil {
				return updated, err
			}
			values["imported_phone_number_hash"] = hash
		}
		if row.ImportedPhoneNumber != "" {
			encrypted, err := encryptBackfillValue(row.ImportedPhoneNumber, "customer.customers:imported_phone_number:"+row.Token)
			if err != nil {
				return updated, err
			}
			values["imported_phone_number_encrypted"] = encrypted
		}
		if len(row.AdditionalData) > 0 && string(row.AdditionalData) != "null" {
			encrypted, err := encryptBackfillValue(string(row.AdditionalData), "customer.customers:additional_data:"+row.Token)
			if err != nil {
				return updated, err
			}
			values["additional_data_encrypted"] = encrypted
		}
		if err := applyBackfillUpdate(db, "customer.customers", row.Id, values); err != nil {
			return updated, err
		}
		if len(values) > 0 {
			updated++
		}
	}
	return updated, nil
}

type phoneBackfillRow struct {
	Id                                                                                                                                           int32
	MetaPhoneNumberId, DisplayPhoneNumber, DisplayPhoneNumberEncrypted, WAId, WAIdHash, WAIdEncrypted, RegistrationPin, RegistrationPinEncrypted string
}

func backfillPhoneNumbers(db *gorm.DB) (int, error) {
	var rows []phoneBackfillRow
	if err := db.Raw("SELECT id, meta_phone_number_id, display_phone_number, display_phone_number_encrypted, wa_id, wa_id_hash, wa_id_encrypted, registration_pin, registration_pin_encrypted FROM wa.phone_numbers").Scan(&rows).Error; err != nil {
		return 0, err
	}
	updated := 0
	for _, row := range rows {
		values := map[string]any{}
		if row.DisplayPhoneNumber != "" {
			encrypted, err := encryptBackfillValue(row.DisplayPhoneNumber, "wa.phone_numbers:display_phone_number:"+row.MetaPhoneNumberId)
			if err != nil {
				return updated, err
			}
			values["display_phone_number_encrypted"] = encrypted
		}
		if row.WAId != "" {
			hash, err := hashBackfillValue(row.WAId)
			if err != nil {
				return updated, err
			}
			values["wa_id_hash"] = hash
		}
		if row.WAId != "" {
			encrypted, err := encryptBackfillValue(row.WAId, "wa.phone_numbers:wa_id:"+row.MetaPhoneNumberId)
			if err != nil {
				return updated, err
			}
			values["wa_id_encrypted"] = encrypted
		}
		if row.RegistrationPin != "" {
			encrypted, err := encryptBackfillValue(row.RegistrationPin, "wa.phone_numbers:registration_pin:"+row.MetaPhoneNumberId)
			if err != nil {
				return updated, err
			}
			values["registration_pin_encrypted"] = encrypted
		}
		if err := applyBackfillUpdate(db, "wa.phone_numbers", row.Id, values); err != nil {
			return updated, err
		}
		if len(values) > 0 {
			updated++
		}
	}
	return updated, nil
}

type portfolioBackfillRow struct {
	Id                                                         int32
	MetaBusinessPortfolioId, AccessToken, AccessTokenEncrypted string
}

func backfillBusinessPortfolios(db *gorm.DB) (int, error) {
	var rows []portfolioBackfillRow
	if err := db.Raw("SELECT id, meta_business_portfolio_id, access_token, access_token_encrypted FROM wa.business_portfolios").Scan(&rows).Error; err != nil {
		return 0, err
	}
	updated := 0
	for _, row := range rows {
		if row.AccessToken != "" {
			encrypted, err := encryptBackfillValue(row.AccessToken, "wa.business_portfolios:access_token:"+row.MetaBusinessPortfolioId)
			if err != nil {
				return updated, err
			}
			if err := applyBackfillUpdate(db, "wa.business_portfolios", row.Id, map[string]any{"access_token_encrypted": encrypted}); err != nil {
				return updated, err
			}
			updated++
		}
	}
	return updated, nil
}

type messageBackfillRow struct {
	Id               int32
	Token            string
	Payload          []byte
	PayloadEncrypted string
}

func backfillMessages(db *gorm.DB) (int, error) {
	var rows []messageBackfillRow
	if err := db.Raw("SELECT id, token, payload, payload_encrypted FROM wa.messages").Scan(&rows).Error; err != nil {
		return 0, err
	}
	updated := 0
	for _, row := range rows {
		values := map[string]any{}
		if row.Token == "" {
			row.Token = uuid.NewString()
			values["token"] = row.Token
		}
		if len(row.Payload) > 0 && string(row.Payload) != "null" {
			encrypted, err := encryptBackfillValue(string(row.Payload), "wa.messages:payload:"+row.Token)
			if err != nil {
				return updated, err
			}
			values["payload_encrypted"] = encrypted
		}
		if err := applyBackfillUpdate(db, "wa.messages", row.Id, values); err != nil {
			return updated, err
		}
		if len(values) > 0 {
			updated++
		}
	}
	return updated, nil
}

type statusBackfillRow struct {
	Id               int32
	MessageId        int32
	WAMessageId      string
	EncryptionID     string
	Payload          []byte
	PayloadEncrypted string
}

func backfillMessageStatuses(db *gorm.DB) (int, error) {
	var rows []statusBackfillRow
	if err := db.Raw("SELECT id, message_id, wa_message_id, encryption_id, payload, payload_encrypted FROM wa.message_status").Scan(&rows).Error; err != nil {
		return 0, err
	}
	updated := 0
	for _, row := range rows {
		values := map[string]any{}
		if row.EncryptionID == "" {
			row.EncryptionID = uuid.NewString()
			values["encryption_id"] = row.EncryptionID
		}
		if len(row.Payload) > 0 && string(row.Payload) != "null" {
			encrypted, err := encryptBackfillValue(string(row.Payload), "wa.message_status:payload:"+row.EncryptionID)
			if err != nil {
				return updated, err
			}
			values["payload_encrypted"] = encrypted
		}
		if err := applyBackfillUpdate(db, "wa.message_status", row.Id, values); err != nil {
			return updated, err
		}
		if len(values) > 0 {
			updated++
		}
	}
	return updated, nil
}

type recipientBackfillRow struct {
	Id, CampaignId, CustomerId int32
	EncryptionID               string
	Payload                    []byte
	PayloadEncrypted           string
}

func backfillCampaignRecipients(db *gorm.DB) (int, error) {
	var rows []recipientBackfillRow
	if err := db.Raw("SELECT id, campaign_id, customer_id, encryption_id, payload, payload_encrypted FROM customer.campaign_recipients").Scan(&rows).Error; err != nil {
		return 0, err
	}
	updated := 0
	for _, row := range rows {
		values := map[string]any{}
		if row.EncryptionID == "" {
			row.EncryptionID = uuid.NewString()
			values["encryption_id"] = row.EncryptionID
		}
		if len(row.Payload) > 0 && string(row.Payload) != "null" {
			encrypted, err := encryptBackfillValue(string(row.Payload), "customer.campaign_recipients:payload:"+row.EncryptionID)
			if err != nil {
				return updated, err
			}
			values["payload_encrypted"] = encrypted
		}
		if err := applyBackfillUpdate(db, "customer.campaign_recipients", row.Id, values); err != nil {
			return updated, err
		}
		if len(values) > 0 {
			updated++
		}
	}
	return updated, nil
}
