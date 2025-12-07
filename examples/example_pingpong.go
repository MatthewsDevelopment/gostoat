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
	client.OnMessage(handlePingResponse)

	log.Fatal(client.ConnectAndRun())
}

// The bot responds with pong when someone says ping
func handlePingResponse(c *stoat.Client, m stoat.Message) {
	if strings.ToLower(strings.TrimSpace(m.Content)) == "ping" {
		content := "pong!"
		err := c.SendMessage(m.ChannelID, stoat.SendMessagePayload{
			Content: &content,
		})
		if err != nil {
			log.Printf("An error has occured: Failed to send 'pong' message: %v", err)
		}
	}
}