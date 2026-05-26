from __future__ import annotations

import argparse
import logging
from typing import Sequence


SCENARIOS = ("normal", "startup", "alarm", "noisy", "intermittent")


def configure_logging(verbose: bool = False) -> None:
    level = logging.DEBUG if verbose else logging.INFO
    logging.basicConfig(
        level=level,
        format="%(asctime)s %(levelname)s %(name)s - %(message)s",
        datefmt="%Y-%m-%d %H:%M:%S",
    )


def build_parser(description: str, default_port: int) -> argparse.ArgumentParser:
    parser = argparse.ArgumentParser(description=description)
    parser.add_argument("--host", default="127.0.0.1", help="监听地址，默认 127.0.0.1")
    parser.add_argument("--port", type=int, default=default_port, help=f"监听端口，默认 {default_port}")
    parser.add_argument(
        "--scenario",
        default="normal",
        choices=SCENARIOS,
        help="设备场景：normal/startup/alarm/noisy/intermittent",
    )
    parser.add_argument("--update-ms", type=int, default=500, help="设备刷新周期，默认 500ms")
    parser.add_argument("--verbose", action="store_true", help="输出调试日志")
    return parser


def print_table(title: str, rows: Sequence[Sequence[object]]) -> None:
    print(f"\n{title}")
    if not rows:
        return
    widths = [max(len(str(row[index])) for row in rows) for index in range(len(rows[0]))]
    for row_index, row in enumerate(rows):
        line = "  ".join(str(value).ljust(widths[index]) for index, value in enumerate(row))
        print(line)
        if row_index == 0:
            print("  ".join("-" * width for width in widths))


def clamp(value: float, minimum: float, maximum: float) -> float:
    return max(minimum, min(maximum, value))

