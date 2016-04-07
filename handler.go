package dagon

import (
	"github.com/streadway/amqp"
	"golang.org/x/net/context"
)

const (
	ServiceExchange = false
	MainExchange    = true
)

/*
   This defines how a response message can be modified by the handler.
*/
type MessageResponseWriter interface {
	AddHeader(string, interface{}) error
	SetProperty(string, interface{}) error
	Write([]byte) (int, error)
}

type Handler interface {
	HandleMessage(context.Context, MessageResponseWriter, amqp.Delivery)
}

type HandlerFunc func(context.Context, MessageResponseWriter, amqp.Delivery)

func (h HandlerFunc) HandleMessage(c context.Context, w MessageResponseWriter, d amqp.Delivery) {
	h(c, w, d)
}

type Queue struct {
	//The basic name of a queue.
	//when the queue is declared on rabbitmq it will be prefaced with the service name
	Name string
	//Bindings are the route keys to bind to a set of Exchanges.
	Bindings map[string]bool
	Opt      QueueOptions
	Handler  Handler
}

type QueueOptions struct {
	Durable    bool
	AutoDelete bool
	Exclusive  bool
	NoWait     bool
	Args       amqp.Table
}

func NewQueue(name string, bindings map[string]bool, opt QueueOptions, handler Handler) *Queue {
	return &Queue{name, bindings, opt, handler}
}

func NewSimpleQueue(name, route string, handler Handler) *Queue {
	binding := map[string]bool{route: ServiceExchange}
	opt := QueueOptions{false, true, false, false, amqp.Table{}}
	return NewQueue(name, binding, opt, handler)
}

func (q *Queue) setup(ctx context.Context) (*amqp.Channel, error) {
	ch, err := ServiceConn.Connection.Channel()
	if err != nil {
		return nil, err
	}
	_, err = ch.QueueDeclare(serviceNamer(ctx, q.Name), q.Opt.Durable, q.Opt.AutoDelete, q.Opt.Exclusive, q.Opt.NoWait, q.Opt.Args)
	if err != nil {
		return nil, err
	}

	for route, dest := range q.Bindings {
		if dest == ServiceExchange {
			route = serviceNamer(ctx, route)
			err := ch.QueueBind(serviceNamer(ctx, q.Name), route, ctx.Value("service_name").(string), false, amqp.Table{})
			if err != nil {
				return nil, err
			}
		} else {
			err := ch.QueueBind(serviceNamer(ctx, q.Name), route, "main", false, amqp.Table{})
			if err != nil {
				return nil, err
			}
		}
	}
	return ch, nil
}

//Setup and run a new thread for each consumer.
func (q *Queue) consume(ctx context.Context, ch *amqp.Channel) error {
	log.Debug(ctx)
	consumer, err := ch.Consume(serviceNamer(ctx, q.Name),
		serviceNamer(ctx,
			q.Name,
			ctx.Value("service_instance").(string)), false, false, false, false, amqp.Table{})
	if err != nil {
		return err
	}

	go func() {
		// This is loop runs so long as there are no errors.
		consuming := true
		for consuming {
			select {
			case <-ctx.Done():
				return
			case d, ok := <-consumer:
				if ok {
					log.Debugf("got a message: %s", d.Body)
					q.Handler.HandleMessage(ctx, nil, d)
					d.Ack(false)
				} else {
					log.Debug("The consumer channel closed. Assume there will be a reconnection")
					consuming = false
				}
			}
		}
	}()
	return nil
}
