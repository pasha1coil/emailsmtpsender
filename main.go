package main

import (
	"bufio"
	"emailsender/client"
	"fmt"
	"github.com/gofiber/fiber/v2"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"log"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
)

type Config struct {
	Auth         client.PlainAuth `yaml:"auth"`
	Sender       string           `yaml:"sender"`
	ApiUrl       string           `yaml:"apiUrl"`
	ApiKey       string           `yaml:"apiKey"`
	Subject      string           `yaml:"subject"`
	Body         string           `yaml:"body"`
	MailListFile string           `yaml:"mailListFilePath"`
}

func loadConfig(filename string) (Config, error) {
	var config Config

	file, err := os.Open(filename)
	if err != nil {
		return config, err
	}
	defer file.Close()

	data, err := ioutil.ReadAll(file)
	if err != nil {
		return config, err
	}

	err = yaml.Unmarshal(data, &config)
	if err != nil {
		return config, err
	}

	return config, nil

}

func loadMailList(filePath string) ([]string, error) {
	var mailList []string

	file, err := os.Open(filePath)
	if err != nil {
		return mailList, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		email := strings.TrimSpace(scanner.Text())
		if email != "" {
			mailList = append(mailList, email)
		}
	}

	if err := scanner.Err(); err != nil {
		return mailList, err
	}

	return mailList, nil

}

func loadHTMLBody(filePath string) (string, error) {
	data, err := ioutil.ReadFile(filePath)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func main() {
	config, err := loadConfig("config.yaml")
	if err != nil {
		log.Fatal("Error loading config:", err)
	}

	htmlBody, err := loadHTMLBody(config.Body)
	if err != nil {
		log.Fatal("Error loading HTML body:", err)
	}

	mailList, err := loadMailList(config.MailListFile)
	if err != nil {
		log.Fatal("Error loading mail list:", err)
	}

	mailClient := client.NewMailClient(client.MailClientDeps{
		ApiUrl:      config.ApiUrl,
		ApiKey:      config.ApiKey,
		FiberClient: fiber.AcquireClient(),
		Auth:        config.Auth,
		Sender:      config.Sender,
	})

	var wg sync.WaitGroup
	errorChan := make(chan error, len(mailList))
	done := make(chan struct{})

	for _, mailAddress := range mailList {
		wg.Add(1)
		go func(address string) {
			defer wg.Done()
			err := mailClient.SendMessage(config.Subject, htmlBody, address)
			if err != nil {
				errorChan <- fmt.Errorf("Error sending message to %s: %v", address, err)
			} else {
				fmt.Println("Message sent successfully to", address)
			}
		}(mailAddress)
	}

	go func() {
		wg.Wait()
		close(errorChan)
		close(done)
	}()

	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, syscall.SIGINT, syscall.SIGTERM)

	select {
	case <-done:
		for err := range errorChan {
			fmt.Println(err)
		}
		fmt.Println("Exit")
	case <-signalChan:
		fmt.Println("Termination signal. Exit")
	}
}
