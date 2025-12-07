package stoat

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/gorilla/websocket"
)

const (
	DefaultAPIBaseURL = "https://api.stoat.chat"
	DefaultCDNBaseURL = "https://cdn.stoatusercontent.com"
	DefaultWSBaseURL  = "wss://events.stoat.chat"
)
const (
	BotTokenType     = "Bot"
	SessionTokenType = "Session"
)

const heartbeatInterval = 30 * time.Second

type Client struct {
	APIBaseURL string
	CDNBaseURL string
	WSBaseURL  string

	Token     string
	TokenType string
	Prefix string

	HTTPClient *http.Client
	Conn *websocket.Conn

	UserID string

	OnMessageHandlers []func(c *Client, m Message)
	CommandHandlers   map[string]func(c *Client, cmd Command)
}

func NewClient(token string) *Client {
	return &Client{
		APIBaseURL: DefaultAPIBaseURL,
		CDNBaseURL: DefaultCDNBaseURL,
		WSBaseURL:  DefaultWSBaseURL,
		Token:      token,
		TokenType:  BotTokenType,
		Prefix:     "!", // Default prefix for command handler
		HTTPClient: &http.Client{Timeout: 10 * time.Second},
		CommandHandlers: make(map[string]func(c *Client, cmd Command)),
	}
}

func (c *Client) SetBaseURLs(api, cdn, ws string) {
	c.APIBaseURL = api
	c.CDNBaseURL = cdn
	c.WSBaseURL = ws
}

func (c *Client) SetAuthType(tokenType string) error {
	if tokenType != BotTokenType && tokenType != SessionTokenType {
		return fmt.Errorf("[gostoat] invalid token type: %s. Must be '%s' or '%s'", tokenType, BotTokenType, SessionTokenType)
	}
	c.TokenType = tokenType
	return nil
}
func (c *Client) SetCommandPrefix(prefix string) {
	c.Prefix = prefix
}

// --- Helper Functions ---
func Ptr[T any](v T) *T {
	return &v
}

func checkResponseStatus(resp *http.Response) error {
	if resp.StatusCode == http.StatusTooManyRequests {
		log.Fatalf("[gostoat] FATAL ERROR: API returned a status 429 rate limit. Stopping process.")
		return fmt.Errorf("[gostoat] API returned a status 429 rate limit. Stopping process.")
	}
	if resp.StatusCode >= 400 {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("[gostoat] API request failed with status %d: %s", resp.StatusCode, string(bodyBytes))
	}
	return nil
}

func (c *Client) performAPICall(method, url string, body io.Reader) (*http.Response, error) {
	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return nil, fmt.Errorf("[gostoat] request failed: %w", err)
	}

	authHeader := "X-Bot-Token"
	if c.TokenType == SessionTokenType {
		authHeader = "X-Session-Token"
	}
	req.Header.Set(authHeader, c.Token)

	if method == "POST" || method == "PUT" || method == "PATCH" {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("[gostoat] failed to execute request: %w", err)
	}

	if err := checkResponseStatus(resp); err != nil {
		resp.Body.Close()
		return nil, err
	}

	return resp, nil
}


// --- WebSocket and Event Loop ---
func (c *Client) ConnectAndRun() error {
	log.Println("Connecting to WebSocket...")
	wsURL := c.WSBaseURL
	
	u, err := url.Parse(wsURL)
	if err != nil {
		return fmt.Errorf("[gostoat] Invalid websocket URL: %w", err)
	}

	headers := make(http.Header)
	authHeader := "X-Bot-Token"
	if c.TokenType == SessionTokenType {
		authHeader = "X-Session-Token"
	}

	headers.Set(authHeader, c.Token)

	conn, _, err := websocket.DefaultDialer.Dial(u.String(), headers)
	if err != nil {
		return fmt.Errorf("[gostoat] Websocket connection failed: %w", err)
	}
	c.Conn = conn
	defer c.Conn.Close()
	log.Println("[gostoat] WebSocket connected.")

	if err := c.sendAuthenticate(); err != nil {
		return fmt.Errorf("[gostoat] authentication payload failed: %w", err)
	}

	go c.pingLoop()

	for {
		_, message, err := c.Conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("[gostoat] WebSocket error: %v", err)
			}
			return err
		}
		c.handleEvent(message)
	}
}

func (c *Client) sendAuthenticate() error {
	authPayload := struct {
		Type string `json:"type"`
		Token string `json:"token"`
		V int `json:"v"`
	}{
		Type: "Authenticate",
		Token: c.Token,
		V: 1,
	}

	data, err := json.Marshal(authPayload)
	if err != nil {
		return err
	}
	
	log.Println("Sending Authenticate payload...")
	return c.Conn.WriteMessage(websocket.TextMessage, data)
}

func (c *Client) pingLoop() {
	ticker := time.NewTicker(heartbeatInterval)
	defer ticker.Stop()

	for range ticker.C {
		if c.Conn == nil {
			return
		}
		pingPayload := struct {
			Type string `json:"type"`
		}{
			Type: "Ping",
		}
		data, err := json.Marshal(pingPayload)
		if err != nil {
			log.Printf("[gostoat] Error marshaling ping payload: %v", err)
			continue
		}
		if err := c.Conn.WriteMessage(websocket.TextMessage, data); err != nil {
			log.Printf("[gostoat] Error sending ping: %v", err)
			return
		}
	}
}

func (c *Client) FetchBotUser() (string, error) {
	url := fmt.Sprintf("%s/users/@me", c.APIBaseURL)
	resp, err := c.performAPICall("GET", url, nil)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var user User
	if err := json.NewDecoder(resp.Body).Decode(&user); err != nil {
		return "", fmt.Errorf("[gostoat] failed to decode user response: %w", err)
	}

	return user.ID, nil
}

func (c *Client) GetBotOwnerID() (string, error) {
	url := fmt.Sprintf("%s/users/@me", c.APIBaseURL)
	resp, err := c.performAPICall("GET", url, nil)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var user User
	if err := json.NewDecoder(resp.Body).Decode(&user); err != nil {
		return "", fmt.Errorf("[gostoat] failed to decode user response for owner ID: %w", err)
	}
	if user.Bot == nil || user.Bot.Owner == "" {
		return "", fmt.Errorf("[gostoat] Failed to retrieve bot owner ID. Most likely caused by using a user bot.")
	}

	return user.Bot.Owner, nil
}

func (c *Client) GetChannel(channelID string) (*Channel, error) {
	url := fmt.Sprintf("%s/channels/%s", c.APIBaseURL, channelID)
	resp, err := c.performAPICall("GET", url, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var channel Channel
	if err := json.NewDecoder(resp.Body).Decode(&channel); err != nil {
		return nil, fmt.Errorf("[gostoat] failed to decode channel response: %w", err)
	}

	return &channel, nil
}

func (c *Client) IsChannelNSFW(channelID string) (bool, error) {
	channel, err := c.GetChannel(channelID)
	if err != nil {
		return false, err
	}
	return channel.NSFW, nil
}

func (c *Client) handleEvent(rawMessage []byte) {
	// log.Printf("[gostoat] DEBUG: Received raw WS message: %s", string(rawMessage))

	var typeOnly struct {
		Type string `json:"type"`
	}
	if err := json.Unmarshal(rawMessage, &typeOnly); err != nil {
		log.Printf("[gostoat] Failed to unmarshal event type: %v. Raw message size: %d", err, len(rawMessage))
		return
	}
	
	eventType := typeOnly.Type
	switch eventType {
	case "Authenticated":
		log.Println("[gostoat] ✅ Successfully connected to stoat.chat. Bot is Ready!")

		userID, err := c.FetchBotUser()
		if err != nil {
			log.Printf("[gostoat] Failed to fetch bot user ID: %v", err)
			return
		}
		c.UserID = userID
		log.Printf("[gostoat] Bot User ID successfully fetched: %s", c.UserID)

	case "Pong":
		// Server response to our Ping heartbeat. This is expected and normal.
	case "Ping":
		// Server is pinging us, we must respond with a Pong immediately.
		c.sendPong()
	case "Message":
		log.Println("[gostoat] DEBUG: Reached Message handler switch case.")
		var msg Message
		if err := json.Unmarshal(rawMessage, &msg); err != nil {
			log.Printf("Error unmarshaling Message data: %v", err)
			return
		}
		log.Printf("[gostoat] Received Message event from %s in channel %s: %s", msg.AuthorID, msg.ChannelID, msg.Content)
		c.handleMessage(msg)
	case "ChannelStartTyping", "ChannelStopTyping":
		// Explicitly handling typing events to avoid "unhandled event type" spam.
		log.Printf("[gostoat] DEBUG: Typing event received: %s", eventType)
	default:
		log.Printf("[gostoat] DEBUG: Received unhandled event type: %s", eventType)
	}
}

func (c *Client) sendPong() error {
	pongPayload := struct {
		Type string `json:"type"`
	}{
		Type: "Pong",
	}
	data, err := json.Marshal(pongPayload)
	if err != nil {
		return err
	}
	return c.Conn.WriteMessage(websocket.TextMessage, data)
}

// handleMessage processes a MessageCreate event.
func (c *Client) handleMessage(m Message) {
	if c.UserID != "" && m.AuthorID == c.UserID {
		return
	}

	for _, handler := range c.OnMessageHandlers {
		handler(c, m)
	}

	trimmedContent := strings.TrimSpace(m.Content)

	if strings.HasPrefix(trimmedContent, c.Prefix) {
		cmd := c.parseCommand(m)
		if handler, ok := c.CommandHandlers[cmd.Name]; ok {
			handler(c, cmd)
		} else {
			// Log for if a command is recognized but there is no handler registered. Off for official release.
			// log.Printf("Unknown command: !%s", cmd.Name)
		}
	}
}

func (c *Client) parseCommand(m Message) Command {
	text := strings.TrimSpace(strings.TrimPrefix(m.Content, c.Prefix))

	cmd := Command{
		Message: m,
		Name:    "",
		Args:    []string{},
	}

	parts := strings.Fields(text)

	if len(parts) > 0 {
		cmd.Name = parts[0]
		cmdNameLen := len(cmd.Name)
		if len(text) > cmdNameLen {
			argsStart := cmdNameLen
			if argsStart < len(text) && text[argsStart] == ' ' {
				argsStart++
			}
			argsString := strings.TrimSpace(text[argsStart:])
			if argsString != "" {
				cmd.Args = []string{argsString}
			}
		}
	}
	return cmd
}

// --- Handler Registration ---
func (c *Client) OnMessage(handler func(c *Client, m Message)) {
	c.OnMessageHandlers = append(c.OnMessageHandlers, handler)
}
func (c *Client) OnCommand(name string, handler func(c *Client, cmd Command)) {
	c.CommandHandlers[name] = handler
}


// --- API Payloads and Structs ---
type Masquerade struct {
	Name   *string `json:"name,omitempty"`
	Avatar *string `json:"avatar,omitempty"`
}

type Embed struct {
	IconURL     *string `json:"icon_url,omitempty"`
	URL         *string `json:"url,omitempty"`
	Title       *string `json:"title,omitempty"`
	Description *string `json:"description,omitempty"`
	Media       *string `json:"media,omitempty"`
	Colour      *string `json:"colour,omitempty"`
}

type SendMessagePayload struct {
	Content    *string      `json:"content,omitempty"`
	Embeds     []Embed      `json:"embeds,omitempty"`
	Masquerade *Masquerade `json:"masquerade,omitempty"`
}

func (c *Client) SendMessage(channelID string, payload SendMessagePayload) error {
	if payload.Content == nil && len(payload.Embeds) == 0 && payload.Masquerade == nil {
		return fmt.Errorf("[gostoat] message payload must contain 'content' or 'embeds''")
	}

	url := fmt.Sprintf("%s/channels/%s/messages", c.APIBaseURL, channelID)
	data, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("[gostoat] marshal message payload failed: %w", err)
	}

	resp, err := c.performAPICall("POST", url, bytes.NewBuffer(data))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("[gostoat] request failed with unexpected status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	return nil
}

func (c *Client) GetMessage(channelID, messageID string) (*Message, error) {
	url := fmt.Sprintf("%s/channels/%s/messages/%s", c.APIBaseURL, channelID, messageID)
	resp, err := c.performAPICall("GET", url, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var message Message
	if err := json.NewDecoder(resp.Body).Decode(&message); err != nil {
		return nil, fmt.Errorf("[gostoat] failed to decode message response: %w", err)
	}

	return &message, nil
}

// --- Event Structures ---
type Message struct {
	ID        string `json:"_id"`
	ChannelID string `json:"channel"`
	Content   string `json:"content"`
	AuthorID  string `json:"author"`
}

type Command struct {
	Message
	Name    string
	Args    []string
}

type Channel struct {
	ID   string `json:"_id"`
	NSFW bool   `json:"nsfw,omitempty"`
}

type BotData struct {
	Owner string `json:"owner"`
}

type User struct {
	ID  string `json:"_id"`
	Bot *BotData `json:"bot,omitempty"`
}