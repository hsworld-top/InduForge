from __future__ import annotations

import logging
import struct
import sys
import threading
import time
from dataclasses import dataclass
from pathlib import Path

if __package__ in {None, ""}:
    sys.path.append(str(Path(__file__).resolve().parents[1]))

from pymodbus.datastore import ModbusSequentialDataBlock, ModbusServerContext, ModbusSlaveContext
from pymodbus.server import StartTcpServer

from industrial_sim.common import build_parser, configure_logging, print_table
from industrial_sim.device_model import PumpStationDevice

LOGGER = logging.getLogger("industrial_sim.modbus")

UNIT_IDS = (1, 2, 3)
STORE_SIZE = 512

COIL_COMMAND_START = 0
DISCRETE_STATUS_START = 0
HOLDING_UINT16_START = 0
HOLDING_INT16_START = 20
HOLDING_FLOAT32_STARTS = {
    "ABCD": 100,
    "BADC": 120,
    "CDAB": 140,
    "DCBA": 160,
}
INPUT_UINT16_START = 0
INPUT_INT16_START = 20
INPUT_FLOAT32_STARTS = {
    "ABCD": 100,
    "BADC": 120,
    "CDAB": 140,
    "DCBA": 160,
}
INPUT_UINT32_STARTS = {
    "ABCD": 200,
    "BADC": 220,
    "CDAB": 240,
    "DCBA": 260,
}
INPUT_INT32_STARTS = {
    "ABCD": 300,
    "BADC": 320,
    "CDAB": 340,
    "DCBA": 360,
}

BYTE_ORDERS = {
    "ABCD": (0, 1, 2, 3),
    "BADC": (1, 0, 3, 2),
    "CDAB": (2, 3, 0, 1),
    "DCBA": (3, 2, 1, 0),
}


@dataclass
class ModbusUnitRuntime:
    unit_id: int
    device: PumpStationDevice
    store: ModbusSlaveContext
    started_at: float


def main() -> None:
    parser = build_parser("InduForge Modbus TCP 泵站模拟设备", 18502)
    args = parser.parse_args()
    configure_logging(args.verbose)
    if not args.verbose:
        logging.getLogger("pymodbus").setLevel(logging.WARNING)

    runtimes = build_unit_runtimes(args.scenario)
    context = ModbusServerContext(slaves={unit_id: runtime.store for unit_id, runtime in runtimes.items()}, single=False)
    stop_event = threading.Event()
    updater = threading.Thread(
        target=update_loop,
        args=(runtimes, context, args.update_ms / 1000.0, stop_event),
        daemon=True,
    )
    updater.start()

    print(
        f"Modbus TCP 模拟设备已启动: {args.host}:{args.port}, "
        f"unitId={','.join(str(item) for item in UNIT_IDS)}, scenario={args.scenario}",
    )
    print_address_plan()

    try:
        StartTcpServer(context=context, address=(args.host, args.port))
    finally:
        stop_event.set()


def print_address_plan() -> None:
    print_table(
        "Modbus 地址段",
        [
            ("区域", "起始地址", "变量数/寄存器数", "类型/字节序", "说明"),
            ("coil", COIL_COMMAND_START, 8, "Bool", "运行、复位、急停、维护等命令位，RW"),
            ("discrete_input", DISCRETE_STATUS_START, 16, "Bool", "状态位、从站标识、质量位，R"),
            ("holding_register", HOLDING_UINT16_START, 12, "UInt16", "目标压力/转速与状态镜像，0/1 可写"),
            ("holding_register", HOLDING_INT16_START, 8, "Int16", "有符号偏差/修正量镜像，R"),
            ("holding_register", 100, "4/8", "Float32 ABCD", "目标/状态浮点镜像，R"),
            ("holding_register", 120, "4/8", "Float32 BADC", "目标/状态浮点镜像，R"),
            ("holding_register", 140, "4/8", "Float32 CDAB", "目标/状态浮点镜像，R"),
            ("holding_register", 160, "4/8", "Float32 DCBA", "目标/状态浮点镜像，R"),
            ("input_register", INPUT_UINT16_START, 12, "UInt16", "过程量缩放值，R"),
            ("input_register", INPUT_INT16_START, 8, "Int16", "过程偏差有符号值，R"),
            ("input_register", 100, "8/16", "Float32 ABCD", "过程量浮点值，R"),
            ("input_register", 120, "8/16", "Float32 BADC", "过程量浮点值，R"),
            ("input_register", 140, "8/16", "Float32 CDAB", "过程量浮点值，R"),
            ("input_register", 160, "8/16", "Float32 DCBA", "过程量浮点值，R"),
            ("input_register", 200, "5/10", "UInt32 ABCD", "运行秒数/累计量，R"),
            ("input_register", 220, "5/10", "UInt32 BADC", "运行秒数/累计量，R"),
            ("input_register", 240, "5/10", "UInt32 CDAB", "运行秒数/累计量，R"),
            ("input_register", 260, "5/10", "UInt32 DCBA", "运行秒数/累计量，R"),
            ("input_register", 300, "5/10", "Int32 ABCD", "有符号诊断量，R"),
            ("input_register", 320, "5/10", "Int32 BADC", "有符号诊断量，R"),
            ("input_register", 340, "5/10", "Int32 CDAB", "有符号诊断量，R"),
            ("input_register", 360, "5/10", "Int32 DCBA", "有符号诊断量，R"),
        ],
    )


def build_unit_runtimes(scenario: str) -> dict[int, ModbusUnitRuntime]:
    runtimes: dict[int, ModbusUnitRuntime] = {}
    for unit_id in UNIT_IDS:
        started_at = time.monotonic()
        device = create_unit_device(scenario, unit_id)
        store = create_slave_store()
        initialize_store(store, device.tick(), unit_id, started_at)
        runtimes[unit_id] = ModbusUnitRuntime(unit_id=unit_id, device=device, store=store, started_at=started_at)
    return runtimes


def create_slave_store() -> ModbusSlaveContext:
    return ModbusSlaveContext(
        di=ModbusSequentialDataBlock(0, [0] * STORE_SIZE),
        co=ModbusSequentialDataBlock(0, [0] * STORE_SIZE),
        hr=ModbusSequentialDataBlock(0, [0] * STORE_SIZE),
        ir=ModbusSequentialDataBlock(0, [0] * STORE_SIZE),
        zero_mode=True,
    )


def create_unit_device(scenario: str, unit_id: int) -> PumpStationDevice:
    unit_scenario = scenario
    if scenario == "normal" and unit_id == 2:
        unit_scenario = "startup"
    elif scenario == "normal" and unit_id == 3:
        unit_scenario = "noisy"

    device = PumpStationDevice(unit_scenario, seed=23 + unit_id)
    if unit_id == 2:
        device.set_target_pressure(5.6)
        device.set_target_speed(1780.0)
    elif unit_id == 3:
        device.set_target_pressure(3.4)
        device.set_target_speed(960.0)
        device.set_run_command(True)
    return device


def update_loop(runtimes: dict[int, ModbusUnitRuntime], context: ModbusServerContext, interval: float, stop_event: threading.Event) -> None:
    while not stop_event.is_set():
        for runtime in runtimes.values():
            update_unit(runtime, context[runtime.unit_id])
        stop_event.wait(interval)


def update_unit(runtime: ModbusUnitRuntime, slave: ModbusSlaveContext) -> None:
    # 命令只从连续命令段读取，便于前端按地址段批量生成控制变量。
    coils = list(slave.getValues(1, COIL_COMMAND_START, count=8))
    holding = list(slave.getValues(3, HOLDING_UINT16_START, count=4))

    runtime.device.set_run_command(bool(coils[0]))
    if bool(coils[1]):
        runtime.device.reset_alarm()
        slave.setValues(1, COIL_COMMAND_START + 1, [False])
    runtime.device.set_emergency_stop(bool(coils[2]))
    runtime.device.set_maintenance_mode(bool(coils[3]))
    if holding[0] > 0:
        runtime.device.set_target_pressure(float(holding[0]) / 100.0)
    if holding[1] > 0:
        runtime.device.set_target_speed(float(holding[1]))

    snapshot = runtime.device.tick()
    write_snapshot(slave, snapshot, runtime.unit_id, runtime.started_at)
    LOGGER.debug("Modbus unit %s snapshot: %s", runtime.unit_id, snapshot)


def initialize_store(store: ModbusSlaveContext, snapshot, unit_id: int, started_at: float) -> None:
    store.setValues(1, COIL_COMMAND_START, [
        snapshot.run_command,
        False,
        snapshot.emergency_stop,
        snapshot.maintenance_mode,
        False,
        False,
        False,
        False,
    ])
    write_snapshot(store, snapshot, unit_id, started_at)


def write_snapshot(slave: ModbusSlaveContext, snapshot, unit_id: int, started_at: float) -> None:
    elapsed = max(0, int(time.monotonic() - started_at))
    total_flow_x10 = max(0, int(snapshot.flow_m3h * max(1, elapsed) / 360.0 * 10))

    slave.setValues(2, DISCRETE_STATUS_START, build_discrete_status(snapshot, unit_id, elapsed))
    slave.setValues(3, HOLDING_UINT16_START, build_holding_uint16(snapshot, unit_id, elapsed))
    slave.setValues(3, HOLDING_INT16_START, build_int16_values(snapshot))
    write_float32_blocks(slave, 3, HOLDING_FLOAT32_STARTS, build_holding_float_values(snapshot))
    slave.setValues(4, INPUT_UINT16_START, build_input_uint16(snapshot, unit_id, elapsed))
    slave.setValues(4, INPUT_INT16_START, build_int16_values(snapshot))
    write_float32_blocks(slave, 4, INPUT_FLOAT32_STARTS, build_input_float_values(snapshot))
    write_uint32_blocks(slave, 4, INPUT_UINT32_STARTS, build_uint32_values(snapshot, elapsed, total_flow_x10))
    write_int32_blocks(slave, 4, INPUT_INT32_STARTS, build_int32_values(snapshot, unit_id))


def build_discrete_status(snapshot, unit_id: int, elapsed: int) -> list[bool]:
    return [
        snapshot.state == "running",
        snapshot.state == "starting",
        snapshot.state in {"fault", "estop"},
        snapshot.alarm_active,
        snapshot.sensor_fault,
        snapshot.comm_flap,
        snapshot.quality_code == 0,
        unit_id == 1,
        unit_id == 2,
        unit_id == 3,
        snapshot.pressure_bar > snapshot.target_pressure_bar,
        snapshot.level_percent < 20,
        elapsed % 2 == 0,
        snapshot.run_command,
        snapshot.state in {"starting", "running"},
        snapshot.maintenance_mode,
    ]


def build_holding_uint16(snapshot, unit_id: int, elapsed: int) -> list[int]:
    return [
        int(snapshot.target_pressure_bar * 100),
        int(snapshot.target_speed_rpm),
        snapshot.state_code,
        snapshot.reset_counter,
        unit_id,
        snapshot.quality_code,
        snapshot.alarm_code,
        elapsed & 0xFFFF,
        int(snapshot.pressure_bar * 100),
        int(snapshot.flow_m3h * 10),
        int(snapshot.level_percent * 10),
        int(snapshot.current_a * 100),
    ]


def build_input_uint16(snapshot, unit_id: int, elapsed: int) -> list[int]:
    return [
        int(snapshot.pressure_bar * 100),
        int(snapshot.temperature_c * 10),
        int(snapshot.flow_m3h * 10),
        int(snapshot.level_percent * 10),
        int(snapshot.speed_rpm),
        int(snapshot.current_a * 100),
        snapshot.alarm_code,
        snapshot.quality_code,
        snapshot.state_code,
        unit_id,
        int(snapshot.target_pressure_bar * 100),
        elapsed & 0xFFFF,
    ]


def build_int16_values(snapshot) -> list[int]:
    return [
        signed16((snapshot.pressure_bar - snapshot.target_pressure_bar) * 100),
        signed16((snapshot.temperature_c - 25.0) * 10),
        signed16((snapshot.flow_m3h - 30.0) * 10),
        signed16((snapshot.level_percent - 50.0) * 10),
        signed16(snapshot.speed_rpm - snapshot.target_speed_rpm),
        signed16((snapshot.current_a - 10.0) * 100),
        signed16(snapshot.quality_code - 1),
        signed16(-snapshot.alarm_code),
    ]


def build_holding_float_values(snapshot) -> list[float]:
    return [
        snapshot.target_pressure_bar,
        snapshot.target_speed_rpm,
        float(snapshot.state_code),
        float(snapshot.alarm_code),
    ]


def build_input_float_values(snapshot) -> list[float]:
    return [
        snapshot.pressure_bar,
        snapshot.temperature_c,
        snapshot.flow_m3h,
        snapshot.level_percent,
        snapshot.speed_rpm,
        snapshot.current_a,
        snapshot.target_pressure_bar,
        float(snapshot.quality_code),
    ]


def build_uint32_values(snapshot, elapsed: int, total_flow_x10: int) -> list[int]:
    return [
        elapsed,
        max(0, int(snapshot.reset_counter)),
        max(0, int(snapshot.alarm_code)),
        total_flow_x10,
        max(0, int(snapshot.speed_rpm * 10)),
    ]


def build_int32_values(snapshot, unit_id: int) -> list[int]:
    return [
        int((snapshot.pressure_bar - snapshot.target_pressure_bar) * 1000),
        int((snapshot.temperature_c - 25.0) * 100),
        int((snapshot.flow_m3h - 30.0) * 100),
        -unit_id * 100,
        int(snapshot.quality_code) - 1,
    ]


def write_float32_blocks(slave: ModbusSlaveContext, function_code: int, starts: dict[str, int], values: list[float]) -> None:
    for order_name, start in starts.items():
        slave.setValues(function_code, start, encode_values(values, order_name, "float32"))


def write_uint32_blocks(slave: ModbusSlaveContext, function_code: int, starts: dict[str, int], values: list[int]) -> None:
    for order_name, start in starts.items():
        slave.setValues(function_code, start, encode_values(values, order_name, "uint32"))


def write_int32_blocks(slave: ModbusSlaveContext, function_code: int, starts: dict[str, int], values: list[int]) -> None:
    for order_name, start in starts.items():
        slave.setValues(function_code, start, encode_values(values, order_name, "int32"))


def encode_values(values: list[float] | list[int], order_name: str, data_type: str) -> list[int]:
    registers: list[int] = []
    for value in values:
        registers.extend(encode_32bit_value(value, order_name, data_type))
    return registers


def encode_32bit_value(value: float | int, order_name: str, data_type: str) -> list[int]:
    """把一个 32 位值编码为指定字节序的两个 Modbus 寄存器。

    输入为数值、字节序名称和数据类型；输出为两个 16 位寄存器。
    不支持的类型或字节序会直接抛错，使模拟器启动/刷新阶段暴露配置问题。
    """

    if data_type == "float32":
        raw = struct.pack(">f", float(value))
    elif data_type == "uint32":
        raw = int(value).to_bytes(4, "big", signed=False)
    elif data_type == "int32":
        raw = int(value).to_bytes(4, "big", signed=True)
    else:
        raise ValueError(f"不支持的 32 位类型: {data_type}")

    order = BYTE_ORDERS[order_name]
    encoded = bytes(raw[index] for index in order)
    return [int.from_bytes(encoded[0:2], "big"), int.from_bytes(encoded[2:4], "big")]


def signed16(value: float | int) -> int:
    return max(-32768, min(32767, int(round(float(value)))))


if __name__ == "__main__":
    main()
