package main

import (
	"os"
	"os/signal"

	"github.com/Logiase/MiraiGo-Template/client"

	_ "github.com/Logiase/MiraiGo-Template/plugins/ping"
)

func main() {
	client.Init()
	client.Login()
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, os.Interrupt)
	<-ch
}
