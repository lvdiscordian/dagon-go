package dagon

import (
	"bytes"
	"github.com/streadway/amqp"
)

const MAX_OUTGOING_MESSAGE = 1 << 20

type Job struct {
	handler Handler
	d       amqp.Delivery
}

type ResponseWriter struct {
	amqp.Publishing
	stage *bytes.Buffer
}

func NewResponseWriter() ResponseWriter {
	rw := ResponseWriter{}
	rw.Publishing = amqp.Publishing{Headers: amqp.Table{}}
	rw.stage = bytes.NewBuffer(make([]byte, 0, MAX_OUTGOING_MESSAGE))
	return rw
}

func (rw ResponseWriter) AddHeader(key string, value interface{}) error {
	rw.Publishing.Headers[key] = value
	if err := rw.Publishing.Headers.Validate(); err != nil {
		delete(rw.Publishing.Headers, key)
		return err
	}
	return nil
}

//SetProperty(string, interface{}) error
//Write([]byte) (int, error)
