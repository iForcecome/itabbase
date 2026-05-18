package main

import (
	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/net/ghttp"

	_ "github.com/gogf/gf/contrib/drivers/sqlite/v2"

	itab "ksogit.kingsoft.net/wpsee/itabbase/server"
)

func main() {
	s := g.Server()

	k := itab.New(
		itab.WithDB(g.DB()),
		itab.WithBuiltinAuth(),
	)

	s.Group("/", func(group *ghttp.RouterGroup) {
		if err := k.Mount(group); err != nil {
			panic(err)
		}
	})

	s.Run()
}
