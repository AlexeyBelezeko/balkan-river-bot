// Package api provides handlers for external APIs and interfaces
package api

import (
	"fmt"
	"log"
	"strings"

	"github.com/abelzeko/water-bot/internal/usecases"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// TelegramBot handles interactions with the Telegram API
type TelegramBot struct {
	bot     *tgbotapi.BotAPI
	useCase *usecases.RiverUseCase
}

// NewTelegramBot creates a new Telegram bot handler
func NewTelegramBot(botToken string, useCase *usecases.RiverUseCase) (*TelegramBot, error) {
	bot, err := tgbotapi.NewBotAPI(botToken)
	if err != nil {
		return nil, fmt.Errorf("failed to create bot: %v", err)
	}

	return &TelegramBot{
		bot:     bot,
		useCase: useCase,
	}, nil
}

// Start begins listening for and handling Telegram messages
func (t *TelegramBot) Start() {
	log.Printf("Authorized on Telegram account %s", t.bot.Self.UserName)

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	updates := t.bot.GetUpdatesChan(u)
	log.Println("Bot is now listening for messages...")

	for update := range updates {
		if update.Message == nil {
			continue
		}

		// Log incoming messages
		log.Printf("Received message from %s (ID: %d): %s",
			update.Message.From.UserName,
			update.Message.From.ID,
			update.Message.Text)

		t.handleMessage(update)
	}
}

// handleMessage processes a Telegram message update
func (t *TelegramBot) handleMessage(update tgbotapi.Update) {
	msg := tgbotapi.NewMessage(update.Message.Chat.ID, "")

	switch {
	case update.Message.IsCommand():
		t.handleCommand(update.Message, &msg)
	default:
		t.handleNonCommand(update.Message, &msg)
	}

	log.Printf("Sending response to user %s", update.Message.From.UserName)
	if _, err := t.bot.Send(msg); err != nil {
		log.Printf("Error sending message: %v", err)
	}
}

// handleCommand processes commands like /start, /help, etc.
func (t *TelegramBot) handleCommand(message *tgbotapi.Message, msg *tgbotapi.MessageConfig) {
	switch message.Command() {
	case "start":
		log.Printf("Handling /start command for user %s", message.From.UserName)
		msg.Text = "Welcome to the Water Bot! Use /rivers to see the list of available rivers or /help for more information."

	case "help":
		log.Printf("Handling /help command for user %s", message.From.UserName)
		msg.Text = "Available commands:\n" +
			"/start - Start the bot\n" +
			"/rivers - Show the list of rivers\n" +
			"/river [name] - Show information for a specific river\n" +
			"/help - Show this help message"

	case "rivers":
		log.Printf("Handling /rivers command for user %s", message.From.UserName)
		t.handleRiversCommand(msg)

	case "river":
		args := message.CommandArguments()
		log.Printf("Handling /river command with args '%s' for user %s", args, message.From.UserName)
		t.handleRiverCommand(args, msg)

	default:
		log.Printf("Received unknown command /%s from user %s", message.Command(), message.From.UserName)
		msg.Text = "Unknown command. Use /help to see available commands."
	}
}

// handleRiversCommand processes the /rivers command
func (t *TelegramBot) handleRiversCommand(msg *tgbotapi.MessageConfig) {
	// Get unique rivers from repository
	rivers, err := t.useCase.GetAvailableRivers()
	if err != nil {
		msg.Text = "Error fetching river data. Please try again later."
		log.Printf("Error fetching river data: %v", err)
		return
	}

	lastUpdate, _ := t.useCase.GetLastUpdateTime()

	msg.Text = "Available rivers:\n\n"
	for _, river := range rivers {
		msg.Text += "• " + river + "\n"
	}
	msg.Text += "\nUse /river [name] to get detailed information."
	msg.Text += fmt.Sprintf("\n\n🕒 Last update: %s", lastUpdate.Format("2006-01-02 15:04:05"))
}

// handleRiverCommand processes the /river [name] command
func (t *TelegramBot) handleRiverCommand(args string, msg *tgbotapi.MessageConfig) {
	if args == "" {
		msg.Text = "Please specify a river name. Example: /river ДУНАВ"
		return
	}

	// Get river data from repository
	riverData, err := t.useCase.GetRiverDataByName(args)
	if err != nil {
		msg.Text = "Error fetching river data. Please try again later."
		log.Printf("Error fetching river data: %v", err)
		return
	}

	if len(riverData) == 0 {
		msg.Text = fmt.Sprintf("No information found for river '%s'. Use /rivers to see the available rivers.", args)
		return
	}

	lastUpdate, _ := t.useCase.GetLastUpdateTime()
	msg.Text = t.useCase.FormatRiverInfo(riverData, lastUpdate)
}

// handleNonCommand processes regular messages
func (t *TelegramBot) handleNonCommand(message *tgbotapi.Message, msg *tgbotapi.MessageConfig) {
	log.Printf("Received non-command message from user %s: %s", message.From.UserName, message.Text)

	if strings.HasPrefix(message.Text, "/river ") {
		riverName := strings.TrimPrefix(message.Text, "/river ")
		t.handleRiverCommand(riverName, msg)
		return
	}

	// Fallback response with bonus info about a default river
	riverData, err := t.useCase.GetRiverDataByName("ГРАДАЦ")
	if err != nil {
		// Fallback to default message if error fetching data
		msg.Text = "I don't understand. Use /help to see available commands."
		log.Printf("Error fetching river data: %v", err)
		return
	}

	// Default response with bonus info about ГРАДАЦ
	var response strings.Builder
	response.WriteString("I don't understand. Use /help to see available commands.\n\n")
	response.WriteString("ЈФУИ (Just For Your Information):\n")

	if len(riverData) > 0 {
		lastUpdate, _ := t.useCase.GetLastUpdateTime()
		response.WriteString(t.useCase.FormatRiverInfo(riverData, lastUpdate))
	} else {
		response.WriteString("No information available for river ГРАДАЦ at the moment.")
	}

	msg.Text = response.String()
}
