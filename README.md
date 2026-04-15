# gostoat

gostoat is an API wrapper that allows you to create stoat.chat bots as well as use stoat webhooks

If you need help with using the gostoat package, join the [gstoat stoat.chat server](https://stt.gg/xHhH0zv7). 

**NOTICE**: This library is still a heavy work in progress. Expect unfinished and buggy features.

This Go package is not officially endorsed by or affiliated with stoat.chat

## Getting Started

This assumes you already have a Go environment on your system. If not, download Go [from here](https://go.dev/dl). Make sure you have Go 1.21 or newer.

Install gostoat by using the following command.

```sh
go get github.com/MatthewsDevelopment/gostoat
```

After installing the package, import the gofluxer package using this within your code.

```go
import (
	"log"
	"strings"
	"fmt"
	"github.com/MatthewsDevelopment/gostoat"
)
```

Here is a basic example of a ping pong bot.

```go
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
```

You can refer to the examples directory for more examples to see how to use this Go package

# gostoat Update Notes:

### Version 0.0.1 - December 7th, 2025

- The first early release version of gostoat. This is a testing version so I can get suggestions on what to add or fix.
- Supports message and command handlers.
- Supports NSFW channel and bot owner checks.
- Supports using webhooks
- Uses MIT license.
- Added a example_pingpong.go example, a example_commands.go example, and a example_webhooks.go example.