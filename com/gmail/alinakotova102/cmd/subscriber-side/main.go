package main

import (
	"context" // пакет для управления жизненным циклом: остановка, отмена, таймауты
	"fmt"
	"os"        // работа с операционной системой (сигналы, переменные окружения)
	"os/signal" // ловим сигналы от системы (Ctrl+C, kill и т.д.)
	"sync"      // примитивы синхронизации (WaitGroup)
	"syscall"   // системные сигналы Unix (SIGTERM и т.д.)

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

const (
	// Адрес MQTT брокера
	// tcp:// — протокол
	// localhost — брокер на этом же компьютере
	// 1883 — стандартный порт MQTT
	broker = "tcp://localhost:1883"

	// clientID — уникальное имя клиента в MQTT
	// Брокер использует его, чтобы отличать клиентов
	clientID = "go-mqtt-subscriber"

	// topic — топик (канал), на который мы подписываемся
	topic = "iot-messages"
)

// chan mqtt.Message — канал, который передаёт MQTT сообщения
var mqttMsgChan = make(chan mqtt.Message)

// messagePubHandler — обработчик входящих сообщений
// Он вызывается каждый раз, когда брокер присылает сообщение
var messagePubHandler mqtt.MessageHandler = func(client mqtt.Client, msg mqtt.Message) {
	mqttMsgChan <- msg // Кладём полученное сообщение в канал
}

// функция обработки сообщений
func processMsg(ctx context.Context, input <-chan mqtt.Message) chan mqtt.Message {
	out := make(chan mqtt.Message) // выходной канал
	go func() {                    // Запускаем горутину (параллельный поток)
		defer close(out) // Закрываем выходной канал при завершении
		for {
			select {

			// Получаем сообщение из входного канала
			case msg, ok := <-input:
				if !ok { // ok == false → канал закрыт
					return
				}

				// Выводим payload (тело сообщения) и топик
				fmt.Printf("Received message: %s from topic: %s\n", msg.Payload(), msg.Topic())

				out <- msg // Передаём сообщение дальше
			case <-ctx.Done(): // срабатывает при cancel()
				return // Аккуратно выходим из горутины
			}
		}
	}()
	return out
}

// connectHandler — вызывается при успешном подключении
var connectHandler mqtt.OnConnectHandler = func(c mqtt.Client) {
	fmt.Println("Connected to MQTT Broker")
}

// connectLostHandler — вызывается при потере соединения
var connectLostHandler mqtt.ConnectionLostHandler = func(client mqtt.Client, err error) {
	fmt.Println("Connection lost: %v", err)
}

func main() {
	// Создаём объект с настройками MQTT клиента
	opts := mqtt.NewClientOptions()
	opts.AddBroker(broker)                           // Добавляем адрес брокера
	opts.SetClientID(clientID)                       // Устанавливаем clientID
	opts.SetDefaultPublishHandler(messagePubHandler) // Устанавливаем обработчик входящих сообщений по умолчанию
	opts.OnConnect = connectHandler                  // Обработчик успешного подключения
	opts.OnConnectionLost = connectLostHandler       // Обработчик потери соединения

	// Создаём MQTT клиента с этими настройками
	client := mqtt.NewClient(opts)

	// Подключаемся к брокеру
	if token := client.Connect(); token.Wait() && token.Error() != nil { // token — объект, который отслеживает асинхронную операцию
		panic(token.Error())
	}

	// Создаём context с возможностью отмены
	ctx, cancel := context.WithCancel(context.Background())

	var wg sync.WaitGroup // WaitGroup — чтобы дождаться завершения горутины
	wg.Add(1)             // Добавляем 1 горутину в счётчик

	// Запускаем горутину обработки сообщений
	go func() {
		defer wg.Done()                           // Уменьшаем счётчик при выходе
		finalChan := processMsg(ctx, mqttMsgChan) // Запускаем обработку сообщений

		// Просто читаем канал, пока он не закроется
		for range finalChan {
			// Здесь могла бы быть логика обработки
		}
	}()

	// Подписываемся на топик
	// QoS = 1 → доставить хотя бы один раз с подтверждением
	token := client.Subscribe(topic, 1, nil)
	token.Wait()
	fmt.Printf("Subscribed to topic: %s\n", topic)

	// Канал для системных сигналов
	sigChan := make(chan os.Signal, 1) // Подписываемся на Ctrl+C и SIGTERM
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	<-sigChan // Блокируем main, пока не придёт сигнал

	// Отменяем context → останавливаем горутины
	cancel()

	fmt.Println("Unsubscribing and disconnecting...")
	client.Unsubscribe(topic) // Отписываемся от топика

	// Отключаемся от брокера
	// 250 — время ожидания отправки сообщений (мс)
	client.Disconnect(250)

	wg.Wait() // Ждём завершения всех горутин
	fmt.Println("Goroutine terminated, exiting...")
}
