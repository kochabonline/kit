package casbin

import "gorm.io/gorm"

type Config struct {
	DB    *gorm.DB
	Model string
}
