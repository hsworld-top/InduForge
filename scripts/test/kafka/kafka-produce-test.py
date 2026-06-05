#!/usr/bin/env python3
from __future__ import annotations

import argparse
import json
import math
import random
import signal
import subprocess
import sys
import time
from datetime import datetime
from typing import Any


DEFAULT_CONTAINER = "induforge-test-redpanda"
DEFAULT_BROKER = "127.0.0.1:9092"
DEFAULT_TOPICS = {
    "telemetry": "induforge.kafka.telemetry",
    "batch": "induforge.kafka.batch",
    "event": "induforge.kafka.event",
}
QUALITY_CODES = [192, 192, 192, 192, 64, 68, 80, 84, 0, 24, 28, 36]
RUNNING = True


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser(
        description="向本地 Kafka/Redpanda 测试容器持续写入数据中心建点样本。"
    )
    parser.add_argument("--container", default=DEFAULT_CONTAINER, help=f"容器名，默认 {DEFAULT_CONTAINER}")
    parser.add_argument(
        "--broker",
        default=DEFAULT_BROKER,
        help=f"容器内 Kafka broker，默认 {DEFAULT_BROKER}",
    )
    parser.add_argument(
        "--telemetry-topic",
        default=DEFAULT_TOPICS["telemetry"],
        help=f"单设备遥测 Topic，默认 {DEFAULT_TOPICS['telemetry']}",
    )
    parser.add_argument(
        "--batch-topic",
        default=DEFAULT_TOPICS["batch"],
        help=f"批量变量 Topic，默认 {DEFAULT_TOPICS['batch']}",
    )
    parser.add_argument(
        "--event-topic",
        default=DEFAULT_TOPICS["event"],
        help=f"事件状态 Topic，默认 {DEFAULT_TOPICS['event']}",
    )
    parser.add_argument("--interval", type=float, default=1.0, help="推送间隔秒数，默认 1")
    parser.add_argument("--count", type=int, default=0, help="每个 Topic 推送次数，0 表示持续推送")
    parser.add_argument(
        "--topics",
        default="telemetry,batch,event",
        help="启用的样本类型：telemetry,batch,event，默认全启用",
    )
    parser.add_argument("--skip-create-topic", action="store_true", help="不自动创建 Topic")
    parser.add_argument("--seed", type=int, default=0, help="随机种子，默认使用当前时间")
    return parser.parse_args()


def run_command(command: list[str], *, input_text: str | None = None) -> subprocess.CompletedProcess[str]:
    return subprocess.run(
        command,
        input=input_text,
        text=True,
        stdout=subprocess.PIPE,
        stderr=subprocess.PIPE,
        check=False,
    )


def docker_exec(container: str, args: list[str], *, input_text: str | None = None) -> subprocess.CompletedProcess[str]:
    return run_command(["docker", "exec", "-i", container, *args], input_text=input_text)


def assert_container_ready(container: str, broker: str) -> None:
    result = docker_exec(container, ["rpk", "cluster", "info", "--brokers", broker])
    if result.returncode != 0:
        raise RuntimeError(
            "Kafka 测试容器不可用，请先运行 scripts/test/kafka/start-redpanda.sh。\n"
            f"stderr: {result.stderr.strip()}"
        )


def create_topic(container: str, broker: str, topic: str) -> None:
    # Topic 已存在时 rpk 会返回非 0；这里再查一次列表，避免重复创建导致脚本退出。
    list_result = docker_exec(container, ["rpk", "topic", "list", "--brokers", broker])
    if list_result.returncode == 0 and topic in list_result.stdout:
        return

    result = docker_exec(
        container,
        ["rpk", "topic", "create", topic, "--brokers", broker, "--partitions", "1"],
    )
    if result.returncode != 0 and "already exists" not in result.stderr.lower():
        raise RuntimeError(f"创建 Topic 失败：{topic}\n{result.stderr.strip()}")


def now_text() -> str:
    return datetime.now().strftime("%Y-%m-%d %H:%M:%S")


def telemetry_payload(sequence: int) -> dict[str, Any]:
    angle = sequence / 8
    temperature = 45 + math.sin(angle) * 8 + random.random()
    pressure = 1.8 + math.cos(angle / 2) * 0.35
    running = sequence % 30 not in (0, 1, 2)
    alarm_code = 2 if temperature > 52 else 0

    return {
        "source": "kafka-telemetry-sim",
        "deviceId": "pump-A",
        "sequence": sequence,
        "ts": now_text(),
        "metrics": {
            "temperature": round(temperature, 3),
            "pressure": round(pressure, 3),
            "flow": round(35 + math.sin(angle / 3) * 9, 3),
            "speed": 1450 + (sequence % 80),
            "current": round(12 + math.sin(angle) * 1.7, 3),
        },
        "status": {
            "running": running,
            "alarmCode": alarm_code,
            "quality": "good" if alarm_code == 0 else "warning",
        },
        "qualityCode": random.choice(QUALITY_CODES),
    }


def batch_payload(sequence: int) -> dict[str, Any]:
    names = [
        "temperature",
        "pressure",
        "flow",
        "speed",
        "current",
        "level",
        "vibration",
        "energy",
    ]
    values = []
    for index, name in enumerate(names):
        base = ((sequence + index * 7) % 100) + 1
        value: int | float
        if index in (0, 1, 2, 4, 6):
            value = round(base + random.random(), 4)
        else:
            value = base
        values.append(
            {
                "N": name,
                "V": value,
                "T": now_text(),
                "Q": random.choice(QUALITY_CODES),
            }
        )
    return {
        "source": "kafka-batch-sim",
        "deviceId": "pump-A",
        "sequence": sequence,
        "values": values,
    }


def event_payload(sequence: int) -> dict[str, Any]:
    event_types = ["state-change", "alarm", "operator-command", "quality-change"]
    event_type = event_types[sequence % len(event_types)]
    return {
        "source": "kafka-event-sim",
        "eventId": f"evt-{sequence:06d}",
        "eventType": event_type,
        "deviceId": "pump-A",
        "severity": "warning" if event_type == "alarm" else "info",
        "message": {
            "state-change": "设备运行状态变化",
            "alarm": "温度接近上限",
            "operator-command": "收到远程启动命令",
            "quality-change": "数据质量发生变化",
        }[event_type],
        "ts": now_text(),
        "payload": {
            "sequence": sequence,
            "ackRequired": event_type == "alarm",
            "value": round(20 + random.random() * 80, 3),
        },
    }


def produce_one(container: str, broker: str, topic: str, payload: dict[str, Any]) -> None:
    line = json.dumps(payload, ensure_ascii=False, separators=(",", ":")) + "\n"
    result = docker_exec(container, ["rpk", "topic", "produce", topic, "--brokers", broker], input_text=line)
    if result.returncode != 0:
        raise RuntimeError(f"写入 Topic 失败：{topic}\n{result.stderr.strip()}")


def handle_signal(signum: int, _frame: object) -> None:
    global RUNNING
    RUNNING = False
    print(f"\n收到退出信号 {signum}，准备停止推送...")


def main() -> int:
    args = parse_args()
    enabled = {item.strip() for item in args.topics.split(",") if item.strip()}
    unknown = enabled - set(DEFAULT_TOPICS)
    if unknown:
        print(f"不支持的样本类型：{', '.join(sorted(unknown))}", file=sys.stderr)
        return 2

    if args.seed:
        random.seed(args.seed)

    signal.signal(signal.SIGINT, handle_signal)
    signal.signal(signal.SIGTERM, handle_signal)

    topics = {
        "telemetry": args.telemetry_topic,
        "batch": args.batch_topic,
        "event": args.event_topic,
    }

    assert_container_ready(args.container, args.broker)
    if not args.skip_create_topic:
        for kind in sorted(enabled):
            create_topic(args.container, args.broker, topics[kind])

    print("Kafka 模拟数据开始推送：")
    for kind in sorted(enabled):
        print(f"  {kind}: {topics[kind]}")
    print("按 Ctrl+C 停止。")

    sequence = 1
    while RUNNING and (args.count <= 0 or sequence <= args.count):
        if "telemetry" in enabled:
            produce_one(args.container, args.broker, topics["telemetry"], telemetry_payload(sequence))
        if "batch" in enabled:
            produce_one(args.container, args.broker, topics["batch"], batch_payload(sequence))
        if "event" in enabled:
            produce_one(args.container, args.broker, topics["event"], event_payload(sequence))

        print(f"[{now_text()}] 已推送第 {sequence} 轮")
        sequence += 1
        if args.count <= 0 or sequence <= args.count:
            time.sleep(max(args.interval, 0.1))

    print("Kafka 模拟数据推送结束。")
    return 0


if __name__ == "__main__":
    try:
        raise SystemExit(main())
    except KeyboardInterrupt:
        print("\nKafka 模拟数据推送结束。")
        raise SystemExit(0)
    except Exception as exc:
        print(f"错误：{exc}", file=sys.stderr)
        raise SystemExit(1)
