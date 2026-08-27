package gormdb

import (
	"context"
	"errors"
	"reflect"
	"time"

	"github.com/jjcheng/wawa-go/internal/dao"
	"github.com/jjcheng/wawa-go/internal/repository"
	"github.com/jjcheng/wawa-go/internal/service"

	"gorm.io/gorm"
)

type Repository[T dao.DAO] struct {
	db         *gorm.DB
	entityName string
	logger     *service.Logger
}

// READ
// does not log errors, errors should be logged in individual repositories or feature which carries more info
func NewRepository[T dao.DAO](db *gorm.DB, logger *service.Logger) repository.Repository[T] {
	repository := Repository[T]{
		db:     db,
		logger: logger,
	}
	var object T
	t := reflect.TypeOf(object)
	repository.entityName = t.Name()
	return &repository
}

func (repository *Repository[T]) GetById(ctx context.Context, id int32) (*T, error, bool) {
	var entity T
	result := repository.db.WithContext(ctx).First(&entity, "id = ?", id)
	if err := result.Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil, true
		}
		repository.logger.ErrorFunction(err, repository.entityName, id)
		return nil, err, false
	}
	return &entity, nil, false
}

func (repository *Repository[T]) ListAll(ctx context.Context) ([]T, error) {
	var entities []T
	if err := repository.db.WithContext(ctx).Order("id").Find(&entities).Error; err != nil {
		repository.logger.ErrorFunction(err, repository.entityName)
		return nil, err
	}
	return entities, nil
}

// WRITE
func (repository *Repository[T]) Insert(ctx context.Context, entity *T) error {
	now := time.Now()
	repository.setField(entity, "EntryDate", now)
	repository.setField(entity, "LastUpdate", now)
	// Always set Id to 0 to let PostgreSQL auto-generate it
	repository.setField(entity, "Id", int32(0))
	result := repository.db.WithContext(ctx).Omit("id").Create(entity)
	if result.Error != nil {
		if !errors.Is(result.Error, gorm.ErrRecordNotFound) {
			repository.logger.ErrorFunction(result.Error, repository.entityName, entity)
		}
		return result.Error
	}
	return nil
}

// CreateBatch inserts multiple LLM logs in a single transaction
func (repository *Repository[T]) InsertBulk(ctx context.Context, entities []T) error {
	if len(entities) == 0 {
		return nil
	}
	now := time.Now()
	for i := range entities {
		repository.setField(&entities[i], "EntryDate", now)
		repository.setField(&entities[i], "LastUpdate", now)
		// Ensure Id is 0 to let PostgreSQL generate it automatically
		repository.setField(&entities[i], "Id", int32(0))
	}
	result := repository.db.WithContext(ctx).Omit("id").CreateInBatches(entities, len(entities))
	if err := result.Error; err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			repository.logger.ErrorFunction(err, repository.entityName, entities)
		}
		return err
	}
	return nil
}

func (repository *Repository[T]) Update(ctx context.Context, entity *T) error {
	now := time.Now()
	repository.setField(entity, "LastUpdate", now)
	result := repository.db.WithContext(ctx).Select("*").Where("id = ?", (*entity).Base().Id).Updates(entity) //need to select * first, otherwise zero (default) values won't update
	if result.Error != nil {
		if !errors.Is(result.Error, gorm.ErrRecordNotFound) {
			repository.logger.ErrorFunction(result.Error, repository.entityName, entity)
		}
		return result.Error
	}
	return nil
}

func (repository *Repository[T]) UpdateFields(ctx context.Context, id int32, fields map[string]any) error {
	var entity T
	fields["last_update"] = time.Now()
	result := repository.db.WithContext(ctx).Model(&entity).Where("id = ?", id).Updates(fields)
	if result.Error != nil {
		if !errors.Is(result.Error, gorm.ErrRecordNotFound) {
			repository.logger.ErrorFunction(result.Error, repository.entityName, id, fields)
		}
		return result.Error
	}
	return nil
}

func (repository *Repository[T]) Delete(ctx context.Context, entity *T) error {
	return repository.DeleteById(ctx, (*entity).Base().Id)
}

func (repository *Repository[T]) DeleteById(ctx context.Context, id int32) error {
	var entity T
	if err := repository.db.WithContext(ctx).Model(entity).Delete("id = ?", id).Error; err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			repository.logger.ErrorFunction(err, repository.entityName, id)
		}
		return err
	}
	return nil
}

func (repository *Repository[T]) DeleteAll(ctx context.Context) error {
	var entity T
	if err := repository.db.WithContext(ctx).Session(&gorm.Session{AllowGlobalUpdate: true}).Delete(&entity).Error; err != nil {
		repository.logger.ErrorFunction(err, repository.entityName)
		return err
	}
	return nil
}

func (repository *Repository[T]) setField(entity *T, fieldName string, fieldValue any) bool {
	elements := reflect.ValueOf(entity).Elem()
	if field := elements.FieldByName(fieldName); field.IsValid() && field.CanSet() {
		field.Set(reflect.ValueOf(fieldValue))
		return true
	}
	return false
}
