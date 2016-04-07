package dagon

//The connection handling
import (
	"github.com/streadway/amqp"
	"time"
)

var (
	ServiceConn *Connection
)

type Connection struct {
	*amqp.Connection

	/*
	   This should be a closure that will contain the uri of the rabbitmq server.
	*/
	dialer   func() error
	attempts uint8
}

/*
Creates a new connection.
*/
func dial(uri string) (*Connection, error) {
	conn := &Connection{}

	conn.dialer = func() error {
		var err error

		if conn.Connection, err = amqp.Dial(uri); err != nil {
			return err
		}
		return nil
	}

	if err := conn.dialer(); err != nil {
		return nil, err
	}
	return conn, nil
}

/*
This is a redial function.  It registers a channel with the connection that will receive an error if it is disrupted.
Then starts a goroutine that will read off this channel, this will block until the channel closes or an error is sent.
It outputs this error to a channel that is passed to it and then reattempts to connect infinitely.
Once it's reconnected `true` is sent on the done channel to signal that the connection is once again available. Before
this is done the routine calls redial set it's
*/
func (conn *Connection) redial(outError chan *amqp.Error, done chan bool) {
	connClosed := make(chan *amqp.Error)
	connClosed = conn.Connection.NotifyClose(connClosed)

	go func() {

		err := <-connClosed //This will block until an error is sent or it is closed.
		if err == nil {
			return //Connection was not severed
		}
		log.Error("Connection was reset.")
		for {
			outError <- err
			time.Sleep(time.Duration(conn.attempts) * time.Second)
			if connErr := conn.dialer(); connErr != nil {
				if conn.attempts < 60 {
					conn.attempts++
				}
			} else {
				conn.attempts = 0
				break
			}
		}

		conn.redial(outError, done)
		done <- true
		return
	}()
}
