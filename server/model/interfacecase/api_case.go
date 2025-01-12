// 自动生成模板ApiCase
package interfacecase

import (
	"github.com/test-instructor/cheetah/server/global"
	"github.com/test-instructor/cheetah/server/model/system"
)

// ApiCase 结构体
// 如果含有time.Time 请自行import time包
type ApiCase struct {
	global.GVA_MODEL
	Name        string          `json:"name" form:"name" gorm:"column:name;comment:;"`
	Status      *bool           `json:"status" form:"status" gorm:"column:status;comment:;"`
	Describe    string          `json:"describe" form:"describe" gorm:"column:describe;comment:;"`
	RunNumber   *int            `json:"runNumber" form:"runNumber" gorm:"column:run_number;comment:;"`
	ProjectID   uint            `json:"-"`
	Project     *system.Project `json:"-"`
	RunConfig   *ApiConfig      `json:"runConfig" form:"runConfig"`
	RunConfigID uint            `json:"RunConfigID" form:"RunConfigID"`
	ApiCaseStep []ApiCaseStep   `json:"case" form:"case" gorm:"many2many:ApiCaseRelationship;"`
	FrontCase   *bool           `json:"front_case" orm:"front_case"`
	EntryID     int             `json:"-"`
	ApiMenuID   uint            `json:"-"`
	ApiMenu     *ApiMenu        `json:"-"`
	CreatedBy   system.SysUser  `json:"-"`
	CreatedByID uint            `json:"-"`
	UpdateBy    system.SysUser  `json:"-"`
	UpdateByID  uint            `json:"-"`
	DeleteBy    system.SysUser  `json:"-"`
	DeleteByID  uint            `json:"-"`
}
