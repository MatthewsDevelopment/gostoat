package main

import (
	"log"
	"strings"
	"fmt"
	"github.com/MatthewsDevelopment/gostoat"
)

func main() {
	token := "STOATBOTTOKEN" // Replace STOATBOTTOKEN with your actual stoat.chat bot token.
	if token == "" {
		log.Fatal("STOATBOTTOKEN environment variable not set.")
	}

	client := stoat.NewClient(token)
	client.SetCommandPrefix("!") // Prefix for the OnCommand handler
	client.OnCommand("ping", handlePingCommand)
	client.OnCommand("say", handleSayCommand)

	log.Fatal(client.ConnectAndRun())
}

func handlePingCommand(c *stoat.Client, cmd stoat.Command) {
	err := c.SendMessage(cmd.ChannelID, stoat.SendMessagePayload{
		Content: stoat.Ptr("pong!"),
	})
	if err != nil {
		log.Printf("An error has occured: Failed to sendresponse: %v", err)
	}
}

func handleSayCommand(c *stoat.Client, cmd stoat.Command) {
	var response string
	if len(cmd.Args) == 0 || cmd.Args[0] == "" {
		response = "Please provide a message for me to say, like: `!say Hello everyone!`"
	} else {
		messageContent := cmd.Args[0] 
		response = fmt.Sprintf("%s", messageContent)
	}

	err := c.SendMessage(cmd.ChannelID, stoat.SendMessagePayload{
		Content: stoat.Ptr(response),
	})
	if err != nil {
		log.Printf("An error has occured: Failed to send response: %v", err)
	}
}