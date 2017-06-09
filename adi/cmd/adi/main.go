package main

import (
	"runtime"

	"github.com/henkman/slackbot/adi"
	_ "github.com/henkman/slackbot/adi/module/level"
	_ "github.com/henkman/slackbot/adi/module/misc"
	_ "github.com/henkman/slackbot/adi/module/points"
	_ "github.com/henkman/slackbot/adi/module/proxycommands"
	_ "github.com/henkman/slackbot/adi/module/web"
	_ "github.com/henkman/slackbot/adi/module/web/duckduckgo"
	_ "github.com/henkman/slackbot/adi/module/web/google"
	_ "github.com/henkman/slackbot/adi/module/web/poll"
	_ "github.com/henkman/slackbot/adi/module/web/yahoo"
)

func main() {
	runtime.GOMAXPROCS(runtime.NumCPU())
	adi.Run()
}
