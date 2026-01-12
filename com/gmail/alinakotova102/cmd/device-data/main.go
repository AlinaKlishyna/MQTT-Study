package main

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"os"
	"os/signal"
	"syscall"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

const (
	broker   = "tcp://localhost:1883"
	clientID = "go-mqtt-client"
	topic    = "iot-messages"
)

// Сообщение - это то, что будет отправлено брокеру MQTT
type Message struct {
	Time       time.Time   `json:"time"`
	DeviceId   string      `json:"device_id"`   // идентификатор устройства
	DeviceType string      `json:"device_type"` // тип устройства
	DeviceData interface{} `json:"device_data"` // интерфейс для конкретных данных устройства
}

type BaseModel struct {
	ID        uint      `gorm:"primarykey" json:"id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type IoTDeviceMessage struct {
	BaseModel
	Time       time.Time       `json:"time" gorm:"index"`
	DeviceID   string          `json:"device_id"`
	DeviceType string          `json:"device_type"`
	DeviceData json.RawMessage `json:"device_data"`
}

func main() {
	// Настройки MQTT клиента
	opts := mqtt.NewClientOptions()
	opts.AddBroker(broker)
	opts.SetClientID(clientID)

	// Создаём клиента
	client := mqtt.NewClient(opts)

	// Подключаемся к брокеру
	if token := client.Connect(); token.Wait() && token.Error() != nil {
		panic(token.Error())
	}
	fmt.Println("Connected to MQTT broker")

	// Канал для Ctrl+C
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	// Основной цикл
loop:
	for {
		select {
		case <-sigChan:
			fmt.Println("Stopping publisher...")
			break loop
		default:
			msg := generateRandomMessage()

			payload, err := json.Marshal(msg)
			if err != nil {
				fmt.Println("JSON error:", err)
				continue
			}

			token := client.Publish(topic, 0, false, payload)
			token.Wait()

			fmt.Printf("Published: %+v\n", msg)
			time.Sleep(1 * time.Second)
		}
	}

	client.Disconnect(250)
	fmt.Println("Disconnected")
}

func generateRandomMessage() Message {
	return Message{
		Time:       time.Now(),
		DeviceId:   "bracelet-123",
		DeviceType: "bracelet",
		DeviceData: map[string]interface{}{
			"heart_rate": rand.Intn(40) + 60,
		},
	}
}
