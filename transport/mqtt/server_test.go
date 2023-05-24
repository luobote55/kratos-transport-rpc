package mqtt

import (
	"context"
	"fmt"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/google/uuid"
	v1 "github.com/luobote55/kratos-transport-rpc/api/v1"
	"github.com/luobote55/kratos-transport-rpc/broker"
	"os"
	"os/signal"
	"reflect"
	"syscall"
	"testing"
	"time"
)

const (
	EmqxBroker        = "tcp://broker.emqx.io:1883"
	EmqxCnBroker      = "tcp://broker-cn.emqx.io:1883"
	EclipseBroker     = "tcp://mqtt.eclipseprojects.io:1883"
	MosquittoBroker   = "tcp://test.mosquitto.org:1883"
	HiveMQBroker      = "tcp://broker.hivemq.com:1883"
	LocalEmxqBroker   = "tcp://127.0.0.1:1883"
	LocalRabbitBroker = "tcp://user:bitnami@127.0.0.1:1883"

	TestTopic = "topic/bobo/helloword"
)

func TestServerSubscriberUpload(t *testing.T) {
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)

	ctx := context.Background()

	srv := NewServer(
		WithAddress([]string{EmqxCnBroker}),
		WithCodec("json"),
	)
	name := "RegisterSubscriberUpload"
	done := make(chan struct{}, 1)
	_ = srv.RegisterSubscriberUpload(ctx, TestTopic, func(ctx context.Context, msg broker.Event) (broker.Any, error) {
		if !reflect.DeepEqual(name, msg.Data().(*v1.HelloRequest).Name) {
			t.Errorf("expect %v, got %v", name, msg.Data())
			return nil, nil
		}
		done <- struct{}{}
		return nil, nil
	}, func(id string) broker.Any { return &v1.HelloRequest{} })

	if err := srv.Start(ctx); err != nil {
		panic(err)
	}

	if _, err := srv.PublishUpload(ctx, TestTopic, &v1.HelloRequest{Name: name}, func(id string) broker.Any { return &v1.HelloReply{} }); err != nil {
		t.Errorf("Publish failed, %s", err.Error())
	}
	ctx1, _ := context.WithTimeout(ctx, time.Second*10)
	select {
	case <-ctx1.Done():
		t.Errorf("RegisterSubscriber failed, timeout")
		break
	case <-done:
		break
	}

	defer func() {
		if err := srv.Stop(ctx); err != nil {
			t.Errorf("expected nil got %v", err)
		}
	}()

	//	<-interrupt
}

func TestServerSubscriberReqPublishReq(t *testing.T) {
	log.With(log.NewStdLogger(os.Stdout),
		"ts", log.DefaultTimestamp,
		"caller", log.DefaultCaller,
	)
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)

	ctx := context.Background()
	ClientId := uuid.New()

	srv := NewServer(
		WithAddress([]string{EmqxCnBroker}),
		WithCodec("json"),
		WithClientId(ClientId.String()),
	)
	name := "RegisterSubscriberReq"
	_ = srv.RegisterSubscriberReq(ctx, TestTopic, func(ctx context.Context, msg broker.Event) (broker.Any, error) {
		req := msg.Data().(*v1.HelloRequest)
		return &v1.HelloReply{
			Message: req.Name,
		}, nil
	}, func(id string) broker.Any { return &v1.HelloRequest{} })

	if err := srv.Start(ctx); err != nil {
		panic(err)
	}
	{
		ctx1, _ := context.WithTimeout(ctx, time.Second*5)
		ctx1 = context.WithValue(ctx1, broker.MeggageTo, ClientId.String())
		ctx1 = context.WithValue(ctx1, broker.Identifier, "HelloRequest")
		resp, err := srv.PublishReq(ctx1,
			TestTopic,
			&v1.HelloRequest{
				Name: name,
			}, func(id string) broker.Any { return &v1.HelloReply{} })
		if err != nil {
			t.Errorf("Publish failed, %s", err.Error())
		} else if !reflect.DeepEqual(name, resp.(*v1.HelloReply).Message) {
			t.Errorf("expect %v, got %v", name, resp.(*v1.HelloReply).Message)
		}
	}
	{
		ctx1, _ := context.WithTimeout(ctx, time.Second*500)
		ctx1 = context.WithValue(ctx1, broker.MeggageTo, "ClientId.String()")
		ctx1 = context.WithValue(ctx1, broker.Identifier, "HelloRequest")
		resp, err := srv.PublishReq(ctx1,
			TestTopic,
			&v1.HelloRequest{
				Name: name,
			}, func(id string) broker.Any { return &v1.HelloReply{} })
		if err != nil {
			if err.Error() != "context deadline exceeded" {
				t.Errorf("Publish failed, %s", err.Error())
			}
		} else if !reflect.DeepEqual(name, resp.(*v1.HelloReply).Message) {
			t.Errorf("expect %v, got %v", name, resp.(*v1.HelloReply).Message)
		}
	}

	defer func() {
		if err := srv.Stop(ctx); err != nil {
			t.Errorf("expected nil got %v", err)
		}
	}()

	//	<-interrupt
}

var SubscriberReqClientid = "SubscriberReqClientid"
var PublishClientid = "PublishClientid"

func TestBenchmarkBroker(t *testing.T) {
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)

	ctx := context.Background()

	srv := NewServer(
		WithAddress([]string{EmqxCnBroker}),
		WithCodec("json"),
		WithAuth("client", "client@0331"),
		WithClientId(PublishClientid),
	)
	name := "RegisterSubscriberReq"
	_ = srv.RegisterSubscriberReq(ctx, TestTopic, func(ctx context.Context, msg broker.Event) (broker.Any, error) {
		startTime := time.Now()
		<-time.NewTicker(time.Millisecond * 10).C // process time
		req := msg.Data().(*v1.HelloRequest)
		return &v1.HelloReply{
			Message:  req.Name,
			TimeFrom: req.TimeFrom,
			TimeRecv: startTime.UnixNano(),
			TimeTo:   time.Now().UnixNano(),
		}, nil
	}, func(identifier string) broker.Any {
		if identifier == "HelloRequest" {
			return &v1.HelloRequest{}
		}
		return nil
	})

	if err := srv.Start(ctx); err != nil {
		panic(err)
	}

	testSum := 1
	testtt := testSum * 4
	var timeIntervalSum int64
	var timeIntervalAvg int64
	var timeIntervalMax int64
	var timeIntervalMin int64 = 0x1000000000

	var timeIntervalFromSum int64
	var timeIntervalFromAvg int64
	var timeIntervalFromMax int64
	var timeIntervalFromMin int64 = 0x1000000000

	var timeIntervalRecvSum int64
	var timeIntervalRecvAvg int64
	var timeIntervalRecvMax int64
	var timeIntervalRecvMin int64 = 0x1000000000

	var timeIntervalToSum int64
	var timeIntervalToAvg int64
	var timeIntervalToMax int64
	var timeIntervalToMin int64 = 0x1000000000

	chantimeInterval := make(chan int64, 100000)
	chantimeIntervalFrom := make(chan int64, 100000)
	chantimeIntervalRecv := make(chan int64, 100000)
	chantimeIntervalTo := make(chan int64, 100000)
	done := make(chan struct{}, 1)
	doneFrom := make(chan struct{}, 1)
	doneTo := make(chan struct{}, 1)
	doneRecv := make(chan struct{}, 1)
	count := 0
	countFrom := 0
	countTo := 0
	countRecv := 0
	tongji := func(name string, c chan int64, sum, avg, max, min *int64, cc *int, d chan struct{}) {
		for {
			select {
			case interval := <-c:
				*sum += interval
				if *max <= interval {
					*max = interval
				}
				if *min >= interval {
					*min = interval
				}
				*cc++
				if *cc >= testtt {
					*avg = (*sum) / int64(testtt)
					d <- struct{}{}
					break
				}
			}
		}
	}
	go tongji("global", chantimeInterval, &timeIntervalSum, &timeIntervalAvg, &timeIntervalMax, &timeIntervalMin, &count, done)
	go tongji("from", chantimeIntervalFrom, &timeIntervalFromSum, &timeIntervalFromAvg, &timeIntervalFromMax, &timeIntervalFromMin, &countFrom, doneFrom)
	go tongji("recv", chantimeIntervalRecv, &timeIntervalRecvSum, &timeIntervalRecvAvg, &timeIntervalRecvMax, &timeIntervalRecvMin, &countRecv, doneRecv)
	go tongji("to", chantimeIntervalTo, &timeIntervalToSum, &timeIntervalToAvg, &timeIntervalToMax, &timeIntervalToMin, &countTo, doneTo)

	xxxxx := "RegisterSubscriberReqRegisterSubscriberReqRegisterSubscriberReqRegisterSubscriberReqRegisterSubscriberReqRegisterSubscriberReqRegisterSubscriberReqRegisterSubscriberReqRegisterSubscriberReqRegisterSubscriberReqRegisterSubscriberReqRegisterSubscriberReqRegisterSubscriberReqRegisterSubscriberReqRegisterSubscriberReqRegisterSubscriberReqRegisterSubscriberReqRegisterSubscriberReqRegisterSubscriberReqRegisterSubscriberReqRegisterSubscriberReqRegisterSubscriberReqRegisterSubscriberReqRegisterSubscriberReqRegisterSubscriberReqRegisterSubscriberReqRegisterSubscriberReqRegisterSubscriberReqRegisterSubscriberReqRegisterSubscriberReqRegisterSubscriberReqRegisterSubscriberReqRegisterSubscriberReqRegisterSubscriberReqRegisterSubscriberReqRegisterSubscriberReqRegisterSubscriberReqRegisterSubscriberReqRegisterSubscriberReqRegisterSubscriberReqRegisterSubscriberReqRegisterSubscriberReqRegisterSubscriberReqRegisterSubscriberReqRegisterSubscriberReqRegisterSubscriberReqRegisterSubscriberReqRegisterSubscriberReqRegisterSubscriberReqRegisterSubscriberReqRegisterSubscriberReqRegisterSubscriberReqRegisterSubscriberReqRegisterSubscriberReqRegisterSubscriberReqRegisterSubscriberReqRegisterSubscriberReqRegisterSubscriberReqRegisterSubscriberReqRegisterSubscriberReqRegisterSubscriberReqRegisterSubscriberReqRegisterSubscriberReqRegisterSubscriberReqRegisterSubscriberReqRegisterSubscriberReqRegisterSubscriberReqRegisterSubscriberReqRegisterSubscriberReqRegisterSubscriberReqRegisterSubscriberReqRegisterSubscriberReqRegisterSubscriberReqRegisterSubscriberReqRegisterSubscriberReqRegisterSubscriberReqRegisterSubscriberReqRegisterSubscriberReqRegisterSubscriberReqRegisterSubscriberReqRegisterSubscriberReqRegisterSubscriberReqRegisterSubscriberReqRegisterSubscriberReqRegisterSubscriberReqRegisterSubscriberReqRegisterSubscriberReqRegisterSubscriberReqRegisterSubscriberReqRegisterSubscriberReqRegisterSubscriberReqRegisterSubscriberReqRegisterSubscriberReqRegisterSubscriberReqRegisterSubscriberReqRegisterSubscriberReqRegisterSubscriberReqRegisterSubscriberReqRegisterSubscriberReqRegisterSubscriberReqRegisterSubscriberReqRegisterSubscriberReqRegisterSubscriberReqRegisterSubscriberReqRegisterSubscriberReqRegisterSubscriberReqRegisterSubscriberReqRegisterSubscriberReqRegisterSubscriberReqRegisterSubscriberReqRegisterSubscriberReqRegisterSubscriberReqRegisterSubscriberReqRegisterSubscriberReqRegisterSubscriberReqRegisterSubscriberReqRegisterSubscriberReqRegisterSubscriberReqRegisterSubscriberReqRegisterSubscriberReqRegisterSubscriberReqRegisterSubscriberReqRegisterSubscriberReqRegisterSubscriberReqRegisterSubscriberReqRegisterSubscriberReqRegisterSubscriberReqRegisterSubscriberReqRegisterSubscriberReqRegisterSubscriberReqRegisterSubscriberReqRegisterSubscriberReqRegisterSubscriberReqRegisterSubscriberReqRegisterSubscriberReqRegisterSubscriberReqRegisterSubscriberReqRegisterSubscriberReqRegisterSubscriberReqRegisterSubscriberReqRegisterSubscriberReqRegisterSubscriberReqRegisterSubscriberReqRegisterSubscriberReq"
	pubCount := 0
	testCase := func() {
		ctx1, _ := context.WithTimeout(ctx, time.Second*10)
		ctx1 = context.WithValue(ctx1, broker.MeggageTo, SubscriberReqClientid)
		ctx1 = context.WithValue(ctx1, broker.Identifier, "HelloRequest")
		startTime := time.Now()
		resp, err := srv.PublishReq(ctx1,
			TestTopic,
			&v1.HelloRequest{
				Name:     xxxxx,
				TimeFrom: startTime.UnixNano(),
			}, func(id string) broker.Any { return &v1.HelloReply{} })
		pubCount++
		stopTime := time.Now()
		if err != nil {
			t.Errorf("Publish failed, %s / %d", err.Error(), pubCount)
		} else if !reflect.DeepEqual(xxxxx, resp.(*v1.HelloReply).Message) {
			t.Errorf("expect %v, got %v", name, resp.(*v1.HelloReply).Message)
		} else {
			//duration := time.Since(startTime)
			time1 := int64(stopTime.UnixNano() - startTime.UnixNano())
			time2 := int64(resp.(*v1.HelloReply).TimeRecv - startTime.UnixNano())
			time3 := int64(resp.(*v1.HelloReply).TimeTo - resp.(*v1.HelloReply).TimeRecv)
			time4 := int64(stopTime.UnixNano() - resp.(*v1.HelloReply).TimeTo)
			//		fmt.Printf("%d,%d,%d,%d\n", time1/1000000, time2/1000000, time3/1000000, time4/1000000)
			chantimeInterval <- time1
			chantimeIntervalFrom <- time2
			chantimeIntervalRecv <- time3
			chantimeIntervalTo <- time4
		}
	}

	go func() {
		for looop := 0; looop < testSum; looop++ {
			testCase()
			//time.Sleep(50 * time.Microsecond)
		}
	}()

	go func() {
		for looop := 0; looop < testSum; looop++ {
			testCase()
			//time.Sleep(50 * time.Microsecond)
		}
	}()

	go func() {
		for looop := 0; looop < testSum; looop++ {
			testCase()
			//time.Sleep(50 * time.Microsecond)
		}
	}()

	go func() {
		for looop := 0; looop < testSum; looop++ {
			testCase()
			//time.Sleep(50 * time.Microsecond)
		}
	}()

	doneDone := 0
forfunc:
	for {
		select {
		case <-done:
			doneDone++
			if doneDone >= 4 {
				break forfunc
			}
		case <-doneRecv:
			doneDone++
			if doneDone >= 4 {
				break forfunc
			}
		case <-doneFrom:
			doneDone++
			if doneDone >= 4 {
				break forfunc
			}
		case <-doneTo:
			doneDone++
			if doneDone >= 4 {
				break forfunc
			}
		case <-interrupt:
			break forfunc
		}
	}
	fmt.Printf("testSum/sent/recv: %d/%d/%d, global: sum/avg/max/min:\t%3dms/\t%3dms/\t%3dms/\t%3dms \n", testtt, pubCount, count, timeIntervalSum/1000000, timeIntervalAvg/1000000, timeIntervalMax/1000000, timeIntervalMin/1000000)
	fmt.Printf("testSum/sent/recv: %d/%d/%d, recv  : sum/avg/max/min:\t%3dms/\t%3dms/\t%3dms/\t%3dms \n", testtt, pubCount, countFrom, timeIntervalFromSum/1000000, timeIntervalFromAvg/1000000, timeIntervalFromMax/1000000, timeIntervalFromMin/1000000)
	fmt.Printf("testSum/sent/recv: %d/%d/%d, proc  : sum/avg/max/min:\t%3dms/\t%3dms/\t%3dms/\t%3dms \n", testtt, pubCount, countRecv, timeIntervalRecvSum/1000000, timeIntervalRecvAvg/1000000, timeIntervalRecvMax/1000000, timeIntervalRecvMin/1000000)
	fmt.Printf("testSum/sent/recv: %d/%d/%d, to    : sum/avg/max/min:\t%3dms/\t%3dms/\t%3dms/\t%3dms \n", testtt, pubCount, countTo, timeIntervalToSum/1000000, timeIntervalToAvg/1000000, timeIntervalToMax/1000000, timeIntervalToMin/1000000)

	defer func() {
		if err := srv.Stop(ctx); err != nil {
			t.Errorf("expected nil got %v", err)
		}
	}()

	//	<-interrupt
}
