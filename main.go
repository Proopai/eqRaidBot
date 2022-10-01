package main

import (
	"eqRaidBot/bot"
	"eqRaidBot/db"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/Netflix/go-env"
	"github.com/bwmarrin/discordgo"
	"github.com/joho/godotenv"
)

type config struct {
	DiscordToken string `env:"TOKEN"`
	DbURI        string `env:"DB_URI"`
	Extras       env.EnvSet
}

func main() {
	conf := loadEnv()
	dg, err := discordgo.New("Bot " + conf.DiscordToken)
	defer dg.Close()

	if err != nil {
		log.Fatal(fmt.Sprintf("Error creating discord session: %s", err.Error()))
	}

	conn, err := db.NewPgPool(conf.DbURI)
	if err != nil {
		log.Fatal(fmt.Sprintf("problem establishing connection to db: %s", err.Error()))
	}

	cmds := bot.NewCommandController(conn)

	autoAttender := bot.NewAutoAttender(conn)
	eventWatcher := bot.NewEventWatcher(conn)

	ac := make(chan struct{})
	ec := make(chan struct{})
	go autoAttender.Run(ac, 5*time.Minute)
	go eventWatcher.Run(ec, 5*time.Minute)

	//t, _ := util.GenerateDBObjects(143)
	//for _, v := range t {
	//	v.Save(conn)
	//}
	//os.Exit(1)
	//splitter := eq.NewSplitter(t, true)
	//splitter.Split(7)
	//os.Exit(0)

	dg.AddHandler(cmds.MessageCreatedHandler)

	dg.Identify.Intents = discordgo.IntentsGuildMessages + discordgo.IntentsDirectMessages

	err = dg.Open()
	if err != nil {
		log.Fatal(fmt.Sprintf("Error opening connection %s", err.Error()))
	}

	log.Printf("EqRaidBot is online. Press CTRL+C to terminate.\n")

	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt, os.Kill)
	<-sc
	ac <- struct{}{}
	ec <- struct{}{}
	log.Println("Shutting down")
}

func loadEnv() *config {
	err := godotenv.Load()
	if err != nil {
		log.Fatal(fmt.Sprintf("Could not load environment: %s", err.Error()))
	}

	var conf config
	if es, err := env.UnmarshalFromEnviron(&conf); err != nil {
		log.Fatal(fmt.Sprintf("Could not load environment: %s", err.Error()))
	} else {
		conf.Extras = es
	}

	return &conf
}
