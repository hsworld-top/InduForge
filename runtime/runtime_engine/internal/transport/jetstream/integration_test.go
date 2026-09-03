package jetstream

import (
	"os"
	"testing"
	"time"

	"github.com/indu-forge/runtime-engine/internal/model"
	"github.com/nats-io/nats.go"
	js "github.com/nats-io/nats.go/jetstream"
)

// 此测试只在调用者明确提供既有 NATS 地址时运行；不启动或重启任何服务。
func TestJetStreamOptInPullAckAndMsgIDDuplicate(t *testing.T) {
	url := os.Getenv("RUNTIME_ENGINE_NATS_TEST_URL")
	if url == "" {
		t.Skip("RUNTIME_ENGINE_NATS_TEST_URL 未设置")
	}
	nc, err := nats.Connect(url)
	if err != nil {
		t.Fatal(err)
	}
	defer nc.Close()
	legacy, err := nc.JetStream()
	if err != nil {
		t.Fatal(err)
	}
	name := "RTE_IT_" + time.Now().UTC().Format("20060102150405000000000")
	subject := "rte.it." + name
	if _, err = legacy.AddStream(&nats.StreamConfig{Name: name, Subjects: []string{subject}, Storage: nats.MemoryStorage, Retention: nats.LimitsPolicy, Discard: nats.DiscardOld, MaxMsgs: 10, Duplicates: time.Minute}); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = legacy.DeleteStream(name) }()
	config := &nats.ConsumerConfig{Durable: "pull", FilterSubject: subject, AckPolicy: nats.AckExplicitPolicy, AckWait: time.Second, MaxDeliver: -1, DeliverPolicy: nats.DeliverAllPolicy, ReplayPolicy: nats.ReplayInstantPolicy, MaxAckPending: 32, MaxWaiting: 32, MaxRequestBatch: 32, MaxRequestExpires: 5 * time.Second, MaxRequestMaxBytes: 1 << 20}
	if _, err = legacy.AddConsumer(name, config); err != nil {
		t.Fatal(err)
	}
	modern, err := js.New(nc)
	if err != nil {
		t.Fatal(err)
	}
	stream, err := modern.Stream(t.Context(), name)
	if err != nil {
		t.Fatal(err)
	}
	consumer, err := stream.Consumer(t.Context(), "pull")
	if err != nil {
		t.Fatal(err)
	}
	info, err := consumer.Info(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	expected := model.Consumer{DurableName: "pull", FilterSubject: subject, AckWaitMS: 1000, BackoffMS: nil, MaxAckPending: 32, MaxWaiting: 32, MaxRequestBatch: 32, MaxRequestExpiresMS: 5000, MaxRequestMaxBytes: 1 << 20}
	if err := compareConsumer(info.Config, expected); err != nil {
		t.Logf("consumer info: %+v", info.Config)
		t.Fatalf("explicit consumer did not round-trip: %v", err)
	}
	if err := validateDurableNames(t.Context(), stream, []string{"pull"}); err != nil {
		t.Fatalf("single configured durable set must validate: %v", err)
	}
	extra := *config
	extra.Durable = "unexpected-query-v1"
	if _, err = legacy.AddConsumer(name, &extra); err != nil {
		t.Fatal(err)
	}
	if err := validateDurableNames(t.Context(), stream, []string{"pull"}); err == nil {
		t.Fatal("unexpected durable on a data stream must reject topology")
	}
	msg := nats.NewMsg(subject)
	msg.Data = []byte("immutable")
	msg.Header.Set("Nats-Msg-Id", "stable-id")
	ack, err := legacy.PublishMsg(msg)
	if err != nil {
		t.Fatal(err)
	}
	if ack.Duplicate {
		t.Fatal("first PubAck unexpectedly duplicate")
	}
	ack, err = legacy.PublishMsg(msg)
	if err != nil {
		t.Fatal(err)
	}
	if !ack.Duplicate {
		t.Fatal("stable Msg-Id duplicate must receive duplicate PubAck")
	}
	client := &Client{js: modern}
	received, err := client.Fetch(t.Context(), name, "pull", 1, time.Second)
	if err != nil {
		t.Fatal(err)
	}
	if string(received[0].Body) != "immutable" {
		t.Fatal("unexpected message bytes")
	}
	if err := received[0].Ack(t.Context()); err != nil {
		t.Fatal(err)
	}
}
