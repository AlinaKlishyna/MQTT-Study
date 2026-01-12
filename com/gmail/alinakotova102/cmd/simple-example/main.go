package main

import (
	"fmt"
	"math/rand" // пакет для генерации случайных чисел
	"time"

	// mqtt — это Go клиент для работы с MQTT брокером.
	mqtt "github.com/eclipse/paho.mqtt.golang" // библиотека, которая реализует протокол MQTT
)

const (
	// broker — адрес MQTT брокера
	// tcp:// — это протокол
	// 1883 — стандартный порт MQTT для TCP соединений
	broker = "tcp://localhost:1883"

	// clientID — уникальный идентификатор клиента MQTT
	// Каждый клиент должен иметь своё имя, чтобы брокер мог отличать его
	clientID = "go-mqtt-client"

	// topic — это «тема», в которую клиент публикует сообщения
	// В MQTT сообщения отправляются и принимаются через топики (топик = канал, как в чате)
	topic = "iot-messages"
)

// connectHandler — это функция, которая выполняется, когда клиент успешно подключается к брокеру
var connectHandler mqtt.OnConnectHandler = func(client mqtt.Client) {
	fmt.Println("Connected to MQTT Broker")
}

// connectLostHandler — функция, которая срабатывает, если соединение с брокером теряется
// err — ошибка, по которой можно понять причину разрыва
var connectLostHandler mqtt.ConnectionLostHandler = func(client mqtt.Client, err error) {
	fmt.Println("Connection lost: %v", err)
}

func main() {
	// opts — объект с опциями для MQTT клиента
	// Сюда мы передаем все настройки: адрес брокера, clientID, обработчики событий и т.д.
	opts := mqtt.NewClientOptions()

	// Добавляем адрес брокера в опции клиента(без этого клиент не знает, куда подключаться)
	opts.AddBroker(broker)

	// Устанавливаем clientID для клиента
	opts.SetClientID(clientID)

	opts.OnConnect = connectHandler            // Указываем, какую функцию вызывать после успешного подключения
	opts.OnConnectionLost = connectLostHandler // Указываем, какую функцию вызывать, если соединение потеряно

	// Создаём новый MQTT клиент с указанными опциями
	// client — это объект, с помощью которого мы будем публиковать и подписываться на топики
	client := mqtt.NewClient(opts)

	// Возвращается token — объект, который отслеживает выполнение операции
	// client.Connect() запускает соединение с брокером
	// token.Wait() — ждём завершения операции (Connect асинхронный)
	// token.Error() — проверяем, произошла ли ошибка
	if token := client.Connect(); token.Wait() && token.Error() != nil {
		panic(token.Error())
	}

	for {
		message := generateRandomMessage()

		// token — объект для отслеживания выполнения операции
		// client.Publish публикует сообщение в указанный топик
		token := client.Publish(
			topic,   // topic — канал, куда отправляем
			0,       // 0 — QoS (качество доставки). 0 значит «доставить хотя бы один раз, без подтверждения»
			false,   // false — retain (сообщение не сохраняется на брокере для новых подписчиков)
			message, // текст сообщения
		)
		token.Wait() // Ждём завершения публикации, чтобы убедиться, что сообщение отправлено
		fmt.Printf("Published message: %s\n", message)
		time.Sleep(1 * time.Second) // Задержка в 1 секунду между публикациями
	}
}

// Генерируем случайное сообщение, которое будем публиковать
func generateRandomMessage() string {
	messages := []string{
		"Hello, World!",
		"Greetings from Go!",
		"MQTT is awesome!",
		"Random message incoming!",
		"Go is fun!",
	}
	// rand.Intn(len(messages)) — случайное число от 0 до длины массива (не включая длину)
	// messages[...] — выбираем сообщение по индексу
	return messages[rand.Intn(len(messages))]
}
