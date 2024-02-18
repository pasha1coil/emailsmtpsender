package client

import (
	"bytes"
	"errors"
	"fmt"
	"github.com/gofiber/fiber/v2"
	"mime/multipart"
)

type MailClientDeps struct {
	ApiUrl      string
	Sender      string
	Auth        PlainAuth
	ApiKey      string
	FiberClient *fiber.Client
}

type PlainAuth struct {
	Identity string
	Username string
	Password string
}

type MailClient struct {
	deps MailClientDeps
}

func NewMailClient(deps MailClientDeps) *MailClient {
	if deps.FiberClient == nil {
		deps.FiberClient = fiber.AcquireClient()
	}
	return &MailClient{
		deps: deps,
	}
}

func (receiver *MailClient) SendMessage(subject string, body string, to string) error {

	form := new(bytes.Buffer)
	writer := multipart.NewWriter(form)
	defer writer.Close()

	fields := map[string]string{
		"from":    receiver.deps.Sender,
		"to":      to,
		"subject": subject,
		"html":    body,
	}

	for key, value := range fields {
		if err := writer.WriteField(key, value); err != nil {
			return err
		}
	}

	if err := writer.Close(); err != nil {
		return err
	}

	req := receiver.deps.FiberClient.Post(receiver.deps.ApiUrl).Body(form.Bytes()).ContentType(writer.FormDataContentType())
	if receiver.deps.ApiKey != "" {
		req.Set("Authorization", receiver.deps.ApiKey)
	}

	s, _, errs := req.Bytes()
	if s != fiber.StatusOK {
		return errors.New("failed sending")
	}
	if errs != nil {
		fmt.Println(errs)
		return errs[0]
	}

	return nil
}
