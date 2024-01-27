package main

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
	"bytes"

	"github.com/joho/godotenv"
)

type Update struct {
	UpdateId int     `json:"update_id"`
	Message  Message `json:"message"`
}

type Message struct {
	Text string `json:"text"`
	Chat Chat   `json:"chat"`
}

type Chat struct {
	Id int `json:"id"`
}
type ChatMessage struct {
    Role    string `json:"role"`
    Content string `json:"content"`
}

type Choice struct {
    Message ChatMessage `json:"message"`
}

type APIResponse struct {
    Choices []Choice `json:"choices"`
}
// getUpdates polls Telegram server for new updates
func getUpdates(offset int) ([]Update, error) {
	resp, err := http.Get("https://api.telegram.org/bot" + os.Getenv("TELEGRAM_BOT_TOKEN") + "/getUpdates?offset=" + strconv.Itoa(offset))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var updates struct {
		Result []Update `json:"result"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&updates); err != nil {
		return nil, err
	}

	return updates.Result, nil
}

// sendTextToTelegramChat sends a text message to the Telegram chat identified by its chat Id
func sendTextToTelegramChat(chatId int, text string) (string, error) {
	telegramApi := "https://api.telegram.org/bot" + os.Getenv("TELEGRAM_BOT_TOKEN") + "/sendMessage"
	response, err := http.PostForm(
		telegramApi,
		url.Values{
			"chat_id": {strconv.Itoa(chatId)},
			"text":    {text},
		})

	if err != nil {
		log.Printf("Error when posting text to the chat: %s", err.Error())
		return "", err
	}
	defer response.Body.Close()

	bodyBytes, errRead := io.ReadAll(response.Body)
	if errRead != nil {
		log.Printf("Error in reading response: %s", errRead.Error())
		return "", errRead
	}
	bodyString := string(bodyBytes)

	log.Printf("Response from Telegram: %s", bodyString)
	return bodyString, nil
}
func sendRequestToOpenAI(chatMessage string) ([]string, error) {
    url := "http://gpt4free:1337/v1/chat/completions"
    requestBody := map[string]interface{}{
        "model": "gpt-3.5-turbo-16k",
        "stream": false,
        "messages": []ChatMessage{
            {
                Role: "assistant",
                Content: chatMessage,
            },
        },
    }

    jsonValue, _ := json.Marshal(requestBody)
    response, err := http.Post(url, "application/json", bytes.NewBuffer(jsonValue))
    if err != nil {
        log.Printf("Error when posting to OpenAI: %s", err.Error())
        return nil, err
    }
    defer response.Body.Close()

    var apiResponse APIResponse
    err = json.NewDecoder(response.Body).Decode(&apiResponse)
    if err != nil {
        log.Printf("Error decoding JSON response: %s", err.Error())
        return nil, err
    }

    var responses []string
    for _, choice := range apiResponse.Choices {
        responses = append(responses, choice.Message.Content)
    }

    return responses, nil
}
func main() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	var lastUpdateId int
	chatStarted := false // Flag to check if chat with OpenAI has started

	for {
		updates, err := getUpdates(lastUpdateId + 1)
		if err != nil {
			log.Printf("Error getting updates: %s", err.Error())
			continue
		}

		for _, update := range updates {
			log.Printf("Received message: %s", update.Message.Text)

			command := strings.Split(update.Message.Text, "@")[0]
			if command == "/start" {
				chatStarted = true
				_, err = sendTextToTelegramChat(update.Message.Chat.Id, "Chat with AI started. How can I help you?")
				if err != nil {
					log.Printf("Error sending reply: %s", err.Error())
				}
			} else if chatStarted {

				// Send received message to OpenAI
				openAIResponses, err := sendRequestToOpenAI(update.Message.Text)
				if err != nil {
					log.Printf("Error getting response from OpenAI: %s", err.Error())
					continue
				}

				
				if err != nil {
					log.Printf("ChatCompletion error: %v\n", err)
					continue
				}

				// Process each response (for simplicity, sending only the first response back to Telegram)
				if len(openAIResponses) > 0 {
					_, err = sendTextToTelegramChat(update.Message.Chat.Id, openAIResponses[0])
					if err != nil {
						log.Printf("Error sending reply: %s", err.Error())
					}
				}
			}

			lastUpdateId = update.UpdateId
		}

		time.Sleep(5 * time.Second) // Poll every 5 seconds
	}
}
