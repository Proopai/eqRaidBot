package main

import (
	"eqRaidBot/bot"
	"fmt"
	"github.com/Netflix/go-env"
	"github.com/bwmarrin/discordgo"
	"github.com/joho/godotenv"
	"log"
	"os"
	"os/signal"
	"syscall"
)

type config struct {
	DiscordToken string `env:"TOKEN"`
	Extras       env.EnvSet
}

func main() {
	conf := loadEnv()
	dg, err := discordgo.New("Bot " + conf.DiscordToken)
	defer dg.Close()

	if err != nil {
		log.Fatal(fmt.Sprintf("Error creating discord session: %s", err.Error()))
	}

	cmds := bot.NewCommands()

	dg.AddHandler(cmds.MessageCreated)

	dg.Identify.Intents = discordgo.IntentsGuildMessages + discordgo.IntentsDirectMessages

	err = dg.Open()
	if err != nil {
		log.Fatal(fmt.Sprintf("Error opening connection %s", err.Error()))
	}

	log.Printf("EqRaidBot is online. Press CTRL+C to terminate.\n")

	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt, os.Kill)
	<-sc
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
