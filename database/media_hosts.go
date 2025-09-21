package database

import (
	"fmt"
	"strings"
	"time"
)

type MediaHost struct {
	ID        int64     `json:"id" gorm:"primaryKey;column:id"`
	MediaType int       `json:"media_type" gorm:"column:media_type;not null;default:0"`
	URLCode   string    `json:"url_code" gorm:"column:url_code;type:varchar(200);not null;default:''"`
	URLName   string    `json:"url_name" gorm:"column:url_name;type:varchar(100);not null;default:''"`
	URLHost   string    `json:"url_host" gorm:"column:url_host;type:varchar(100);not null;default:''"`
	URLScheme string    `json:"url_scheme" gorm:"column:url_scheme;type:varchar(10);not null;default:''"`
	URLPath   string    `json:"url_path" gorm:"column:url_path;type:varchar(200);not null;default:''"`
	URLQuery  string    `json:"url_query" gorm:"column:url_query;type:varchar(1000);not null;default:''"`
	ReqMethod string    `json:"req_method" gorm:"column:req_method;type:varchar(10);not null;default:'GET'"`
	State     int       `json:"state" gorm:"column:state;not null;default:1"`
	CreatedAt time.Time `json:"created_at" gorm:"column:created_at;default:now()"`
	UpdatedAt time.Time `json:"updated_at" gorm:"column:updated_at;default:CURRENT_TIMESTAMP"`
}

func (m *MediaHost) RequestUrl() string {
	bashUrl := fmt.Sprintf(
		"%s://%s/%s",
		m.URLScheme,
		m.URLHost,
		strings.TrimPrefix(m.URLPath, "/"),
	)
	if m.URLQuery != "" {
		return bashUrl + "?" + m.URLQuery
	}
	return bashUrl
}
