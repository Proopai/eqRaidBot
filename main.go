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

	ch := make(chan struct{})
	go autoAttender.Run(ch, 15*time.Second)

	//t, spread := util.GenerateDBObjects(123)
	//for k, v := range spread {
	//	fmt.Println(k, v)
	//}
	//splitter := eq.NewSplitter(t, true)
	//splitter.Split(3)
	//os.Exit(1)

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
	log.Println("Shutting down")
	ch <- struct{}{}
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
