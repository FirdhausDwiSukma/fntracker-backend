package repositories

import "finance-tracker/models"

type UserRepository interface {
	Create(user *models.User) error
	FindByEmail(email string) (*models.User, error)
	FindByID(id uint) (*models.User, error)
}

type CategoryRepository interface {
	FindAllByUser(userID uint) ([]models.Category, error)
	FindByIDAndUser(id, userID uint) (*models.Category, error)
	Create(category *models.Category) error
	Update(category *models.Category) error
	Delete(id, userID uint) error
	ExistsByNameTypeUser(name, categoryType string, userID uint) (bool, error)
}
