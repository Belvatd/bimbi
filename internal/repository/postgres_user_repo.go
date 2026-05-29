package repository

import (
	"errors"

	"bimbi-backend/internal/domain"
	"gorm.io/gorm"
)

// gormUser is an internal mapping struct so domain doesn't depend on GORM
type gormUser struct {
	gorm.Model
	Email        string `gorm:"uniqueIndex;not null"`
	PasswordHash string `gorm:"not null"`
}

type postgresUserRepo struct {
	db *gorm.DB
}

func NewPostgresUserRepo(db *gorm.DB) domain.UserRepository {
	// Auto migrate the internal GORM struct
	db.AutoMigrate(&gormUser{})
	return &postgresUserRepo{db: db}
}

func (r *postgresUserRepo) CreateUser(user *domain.User) error {
	gu := &gormUser{
		Email:        user.Email,
		PasswordHash: user.PasswordHash,
	}

	if err := r.db.Create(gu).Error; err != nil {
		return err
	}

	// Update domain model
	user.ID = gu.ID
	user.CreatedAt = gu.CreatedAt
	user.UpdatedAt = gu.UpdatedAt
	return nil
}

func (r *postgresUserRepo) GetUserByEmail(email string) (*domain.User, error) {
	var gu gormUser
	if err := r.db.Where("email = ?", email).First(&gu).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil // Not found
		}
		return nil, err
	}

	return &domain.User{
		ID:           gu.ID,
		Email:        gu.Email,
		PasswordHash: gu.PasswordHash,
		CreatedAt:    gu.CreatedAt,
		UpdatedAt:    gu.UpdatedAt,
	}, nil
}
