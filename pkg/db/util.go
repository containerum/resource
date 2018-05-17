package db

import (
	"github.com/globalsign/mgo"
)

type PageInfo struct {
	PerPage        int
	Page           int
	DefaultPerPage int
}

func (pages PageInfo) Init() (limit, offset int) {
	if pages.PerPage <= 0 {
		if pages.DefaultPerPage > 0 {
			pages.PerPage = pages.DefaultPerPage
		} else {
			pages.PerPage = 100
		}
	}
	if pages.Page <= 0 {
		pages.Page = 0
	} else {
		pages.Page--
	}
	return pages.PerPage, pages.Page * pages.PerPage
}

func Paginate(query *mgo.Query, info *PageInfo) *mgo.Query {
	if info != nil {
		var limit, offset = info.Init()
		return query.Skip(offset).Limit(limit)
	}
	return query
}
