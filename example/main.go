package main

import (
    dagon "github.com/lvdiscordian/dagon-go"
    "golang.org/x/net/context"
    "github.com/Sirupsen/logrus"
    "github.com/streadway/amqp"
)

var log = logrus.New()

func init() {
    log.Formatter = new(logrus.JSONFormatter)
    log.Level = logrus.DebugLevel
    dagon.SetLogger(log)
}

func main(){
    var handler, handler0 dagon.HandlerFunc
    handler = func(c context.Context, w dagon.MessageResponseWriter, d amqp.Delivery){
        log.Infof("Mischief Managed %s", d.Body)
        log.Info(d.AppId)
        log.Info(d.UserId)
    }
    handler0 = func(c context.Context, w dagon.MessageResponseWriter, d amqp.Delivery){
        log.Infof("Basic Basic %s", d.Body)
    }

    bindings := make(map[string]bool)
    bindings["qTest0"] = dagon.ServiceExchange
    bindings["event.qTest0.#"] = dagon.MainExchange
    qTest0 := dagon.NewQueue("qTest0", bindings, dagon.QueueOptions{false, true, false, false, nil}, handler0)

    qTest1 := dagon.NewSimpleQueue("qTest1",
        "qTest1",
        handler)

    appCtx := context.Background()
    server := dagon.NewService("testy",appCtx )
    server.RegisterQueues(qTest0, qTest1)
    log.Debugln("Starting")
    err := server.Start("amqp://guest:guest@192.168.99.100:5672/")
    if err != nil{
        log.Info(err)
        return
    }
    log.Debug("Wait for context to end.")
    <-server.Ctx.Done()
}