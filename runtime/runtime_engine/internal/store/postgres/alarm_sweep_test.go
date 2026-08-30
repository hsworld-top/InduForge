package postgres

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/indu-forge/runtime-engine/internal/model"
)

func TestAlarmSweepAllowedItemsAreBoundedAndValidatedBeforeTransaction(t *testing.T) {
	items := makeSweepItems(model.MaxAlarmItems)
	if !validAlarmSweepItems(items) {
		t.Fatalf("V1 边界 %d 被拒绝", model.MaxAlarmItems)
	}
	if validAlarmSweepItems(append(items, AlarmSweepItem{AlarmItemID: sweepItemID(model.MaxAlarmItems), AlarmRevision: 1})) {
		t.Fatal("超过 V1 上限的 allowed 集合被接受")
	}

	message := func(items []AlarmSweepItem) AlarmSweepMessage {
		return AlarmSweepMessage{DeploymentID: "deployment", Role: "alarm", Token: ConsumerRoleToken{OwnerID: "owner", Epoch: 1}, ProducerKey: "alarm", ProducerToken: ProducerToken{OwnerID: "owner", Epoch: 1}, OccurredAt: time.Now().UTC(), Limit: 1, AlarmItems: items}
	}
	store := &Store{} // 非法输入必须在 Begin 前拒绝，不能依赖连接池。
	for _, test := range []struct {
		name  string
		items []AlarmSweepItem
	}{
		{name: "empty", items: nil},
		{name: "invalid revision", items: []AlarmSweepItem{{AlarmItemID: sweepItemID(1), AlarmRevision: 0}}},
		{name: "duplicate pair", items: []AlarmSweepItem{{AlarmItemID: sweepItemID(1), AlarmRevision: 1}, {AlarmItemID: sweepItemID(1), AlarmRevision: 1}}},
		{name: "same id different revision", items: []AlarmSweepItem{{AlarmItemID: sweepItemID(1), AlarmRevision: 1}, {AlarmItemID: sweepItemID(1), AlarmRevision: 2}}},
		{name: "over max", items: append(items, AlarmSweepItem{AlarmItemID: sweepItemID(model.MaxAlarmItems), AlarmRevision: 1})},
	} {
		t.Run(test.name, func(t *testing.T) {
			err := store.ProcessAlarmSweep(context.Background(), message(test.items), func(context.Context, *BusinessTx, string) error { return nil })
			if !errors.Is(err, ErrInvalidInput) {
				t.Fatalf("err=%v, want ErrInvalidInput", err)
			}
		})
	}
}

func makeSweepItems(count int) []AlarmSweepItem {
	items := make([]AlarmSweepItem, 0, count)
	for index := range count {
		items = append(items, AlarmSweepItem{AlarmItemID: sweepItemID(index), AlarmRevision: 1})
	}
	return items
}

func sweepItemID(index int) string {
	return fmt.Sprintf("%08x-0000-4000-8000-%012x", 0x60000000+index, index)
}
