package casbin

import "gorm.io/gorm"

type Config struct {
	Db    *gorm.DB
	Model string
}
