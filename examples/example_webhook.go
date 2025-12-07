package main

import (
	"log"

	"github.com/MatthewsDevelopment/gostoat" 
)

func main() {
	// Webhook urls should look something like this: https://stoat.chat/api/webhooks/<WEBHOOKID>/<WEBHOOKTOKEN>
	const webhookID = "WEBHOOKID" 
	const webhookToken = "WEBHOOKTOKEN"
	const apiBaseURL = "https://api.stoat.chat" 

	if webhookID == "WEBHOOKID" || webhookToken == "WEBHOOKTOKEN" {
		log.Fatal("Please replace 'WEBHOOKID' and 'WEBHOOKTOKEN' with actual values.")
	}

	messageContent := "This is an example of using the webhook function in gostoat."
	customUsername := "Webhook Client"

	simpleEmbed := stoat.Embed{
		Title:       stoat.Ptr("Embed title"),
		Description: stoat.Ptr("Embed description"),
	}

	payload := stoat.WebhookPayload{
		Content:   stoat.Ptr(messageContent),
		Username:  stoat.Ptr(customUsername),
		Embeds:    []stoat.Embed{simpleEmbed},
	}

	err := stoat.ExecuteWebhook(apiBaseURL, webhookID, webhookToken, payload)
	if err != nil {
		log.Fatalf("An error has occured: %v", err)
	}

	log.Println("Webhook send successfully")
}