package dagon

import (
    "github.com/streadway/amqp"
    "golang.org/x/net/context"
    "bytes"
    "github.com/Sirupsen/logrus"
    "time"
)

var log *logrus.Logger

func init() {
    log = logrus.New()
    log.Formatter = new(logrus.JSONFormatter)
    log.Level = logrus.DebugLevel
}

func serviceNamer(ctx context.Context, post ...string) string {
    var res bytes.Buffer
    res.WriteString(ctx.Value("service_name").(string))
    for _, next := range post {
        res.WriteString(".")
        res.WriteString(next)
    }
    return res.String()
}

func SetLogger(myLog *logrus.Logger){
    log = myLog
}

type Service struct {
    Name string
    Ctx context.Context
    cancel context.CancelFunc

    connError chan *amqp.Error
    reconnected chan bool

    registry map[string]*Queue

    instanceID func() string
}

func NewService(name string, ctx context.Context) *Service{
    new_ctx, cancel := context.WithCancel(context.WithValue(ctx, "service_name", name))

    return &Service{
        Name: name,
        Ctx: new_ctx,
        cancel: cancel,
        connError: make(chan *amqp.Error),
        reconnected: make(chan bool),
        registry: make(map[string]*Queue),
        instanceID: func () string { return "testy1"},
    }
}

func (s *Service) SetInstanceID(f func() string){
    s.instanceID = f
}


func (s *Service) Stop(){
    log.Debugf("Stopping Service: %s", s.Name)
    s.cancel()
}

func (s *Service) Start(uri string) error{
    s.Ctx = context.WithValue(s.Ctx, "service_instance", s.instanceID())
    conn, err := dial(uri)
    if err != nil {
        return err
    }
    ServiceConn = conn

    if err := s.setupExchanges(); err != nil{
        return err
    }
    if err := s.setupQueues(); err != nil{
        return err
    }

    s.connError = make(chan *amqp.Error)
    s.reconnected = make(chan bool)
    ServiceConn.redial(s.connError, s.reconnected)
    log.Debugln("paused fo 60 seconds")
    time.Sleep(time.Duration(60) * time.Second)

    //Main service loop.
    go func() {
        for {
            select {
            case cerr := <-s.connError:
                if cerr == nil {
                    s.cancel()
                    return
                }
                //Communicate to anything listening that the connection is down.
                //They should block until reconnected.
                log.Error(cerr)
                //s.connectionError(cerr)
            case <-s.Ctx.Done():
                log.Debug("Service is ending: closing connection.")
                ServiceConn.Close()
                return
            }
            reconnect:
            select {
            case cerr := <-s.connError:
                log.Error(cerr)
                break reconnect
            case <-s.reconnected:
                log.Info("Reconnected")
                if err := s.setupExchanges(); err != nil{
                    log.Error("Error reestablishing service after reconnecting.")
                    return
                }
                if err := s.setupQueues(); err != nil{
                    log.Error("Error reestablishing service after reconnecting.")
                    return
                }
            case <-s.Ctx.Done():
                log.Debug("Service is ending: closing connection.")
                ServiceConn.Close()
                return
            }
        }
    }()
    return nil
}

func (s *Service) setupExchanges() error {
    // Ensure that the dagon exchange exists
    ch, err := ServiceConn.Channel()
    if err != nil {
        return err
    }

    if err := ch.ExchangeDeclare("main", "topic", true, false, false, false, amqp.Table{}); err != nil{
        return err
    }

    //Declare service exchange
    if err := ch.ExchangeDeclare(s.Name, "topic", false, true, false, false, amqp.Table{}); err != nil {
        return err
    }


    if err :=ch.ExchangeBind(s.Name, serviceNamer(s.Ctx, "*"), "main", false, amqp.Table{}); err != nil {
        return err
    }
    return nil
}

func (s *Service) setupQueues() error {
    log.Debug("Setting up Queues")
    for _, q := range s.registry{
        ch, err := q.setup(s.Ctx)
        if err != nil {
            return err
        }
        err = q.consume(context.WithValue(s.Ctx, "queue", q.Name), ch)
        if err != nil {
            return err
        }
    }
    return nil
}

/*
This will accept any number of queues and will register them into a map.
If there is an existing queue in the registry with the same name as one being added it will be overwritten.
 */
func (s *Service) RegisterQueues(queues ...*Queue){
    for _, q := range queues{
        name := q.Name
        s.registry[name] = q
    }
}
/* Removes a registered queue */
func (s *Service) RemoveQueue(qName string){
    delete(s.registry, qName)
}