package repository

import (
	"context"
	"errors"

	"bimbi-backend/internal/domain"

	"gorm.io/gorm"
)

type postgresChildRepo struct {
	db *gorm.DB
}

func NewPostgresChildRepo(db *gorm.DB) domain.ChildRepository {
	return &postgresChildRepo{db: db}
}

func (r *postgresChildRepo) Create(ctx context.Context, child *domain.Child) error {
	return r.db.WithContext(ctx).Create(child).Error
}

func (r *postgresChildRepo) GetByID(ctx context.Context, id string) (*domain.Child, error) {
	var child domain.Child
	if err := r.db.WithContext(ctx).Where("id = ?", id).First(&child).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("child not found")
		}
		return nil, err
	}
	return &child, nil
}

func (r *postgresChildRepo) GetByParentID(ctx context.Context, parentID string) ([]*domain.Child, error) {
	var children []*domain.Child
	if err := r.db.WithContext(ctx).Where("parent_id = ?", parentID).Find(&children).Error; err != nil {
		return nil, err
	}
	return children, nil
}

func (r *postgresChildRepo) Update(ctx context.Context, child *domain.Child) error {
	return r.db.WithContext(ctx).Save(child).Error
}

func (r *postgresChildRepo) Delete(ctx context.Context, id string) error {
	return r.db.WithContext(ctx).Where("id = ?", id).Delete(&domain.Child{}).Error
}

type postgresAssessmentRepo struct {
	db *gorm.DB
}

func NewPostgresAssessmentRepo(db *gorm.DB) domain.AssessmentRepository {
	return &postgresAssessmentRepo{db: db}
}

func (r *postgresAssessmentRepo) Create(ctx context.Context, assessment *domain.Assessment) error {
	return r.db.WithContext(ctx).Create(assessment).Error
}

func (r *postgresAssessmentRepo) GetLastAssessmentByChildID(ctx context.Context, childID string) (*domain.Assessment, error) {
	var assessment domain.Assessment
	if err := r.db.WithContext(ctx).Where("child_id = ?", childID).Order("created_at desc").First(&assessment).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil // Return nil, nil when no past assessment exists
		}
		return nil, err
	}
	return &assessment, nil
}

func (r *postgresAssessmentRepo) GetByChildID(ctx context.Context, childID string) ([]*domain.Assessment, error) {
	var assessments []*domain.Assessment
	if err := r.db.WithContext(ctx).Where("child_id = ?", childID).Order("created_at desc").Find(&assessments).Error; err != nil {
		return nil, err
	}
	return assessments, nil
}

func (r *postgresAssessmentRepo) GetAssessmentsByChildID(ctx context.Context, childID string) ([]*domain.Assessment, error) {
	var assessments []*domain.Assessment
	if err := r.db.WithContext(ctx).Where("child_id = ?", childID).Order("assessment_date asc").Find(&assessments).Error; err != nil {
		return nil, err
	}
	return assessments, nil
}

func (r *postgresAssessmentRepo) GetByID(ctx context.Context, id string) (*domain.Assessment, error) {
	var assessment domain.Assessment
	if err := r.db.WithContext(ctx).Where("id = ?", id).First(&assessment).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("assessment not found")
		}
		return nil, err
	}
	return &assessment, nil
}

func (r *postgresAssessmentRepo) Delete(ctx context.Context, id string) error {
	return r.db.WithContext(ctx).Where("id = ?", id).Delete(&domain.Assessment{}).Error
}
