package db

import (
	"os"

	"github.com/globalsign/mgo"
	"github.com/sirupsen/logrus"
)

func Paginate(query *mgo.Query, pages ...int) *mgo.Query {
	switch len(pages) {
	case 0:
		return query
	case 1:
		return query.Limit(pages[0])
	case 2:
		return query.Skip(pages[1]).Limit(pages[0])
	default:
		defer func() { os.Exit(100) }()
		logrus.Panicf("[resource-service/pkg/db.GetDomainList] invalid pagination config: expected at most 2 args, got %d", len(pages))
		return nil
	}
}
