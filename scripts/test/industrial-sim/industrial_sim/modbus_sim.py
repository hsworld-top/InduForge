from __future__ import annotations

import logging
import sys
import threading
import time
from pathlib import Path

if __package__ in {None, ""}:
    sys.path.append(str(Path(__file__).resolve().parents[1]))

from pymodbus.datastore import ModbusSequentialDataBlock, ModbusServerContext, ModbusSlaveContext
from pymodbus.server import StartTcpServer

from industrial_sim.common import build_parser, configure_logging, print_table
from industrial_sim.device_model import PumpStationDevice

LOGGER = logging.getLogger("industrial_sim.modbus")
UNIT_ID = 1


def main() -> None:
    parser = build_parser("InduForge Modbus TCP 泵站模拟设备", 18502)
    args = parser.parse_args()
    configure_logging(args.verbose)
    if not args.verbose:
        logging.getLogger("pymodbus").setLevel(logging.WARNING)

    device = PumpStationDevice(args.scenario)
    store = ModbusSlaveContext(
        di=ModbusSequentialDataBlock(0, [0] * 100),
        co=ModbusSequentialDataBlock(0, [0] * 100),
        hr=ModbusSequentialDataBlock(0, [0] * 100),
        ir=ModbusSequentialDataBlock(0, [0] * 100),
        zero_mode=True,
    )
    initialize_store(store, device.tick())
    context = ModbusServerContext(slaves={UNIT_ID: store}, single=False)
    stop_event = threading.Event()
    updater = threading.Thread(
        target=update_loop,
        args=(device, context, args.update_ms / 1000.0, stop_event),
        daemon=True,
    )
    updater.start()

    print(f"Modbus TCP 模拟设备已启动: {args.host}:{args.port}, unitId={UNIT_ID}, scenario={args.scenario}")
    print_table(
        "Modbus 点表",
        [
            ("区域", "地址", "名称", "类型/缩放", "读写"),
            ("coil", 0, "运行命令", "Bool", "RW"),
            ("coil", 1, "报警复位", "Bool 脉冲", "RW"),
            ("coil", 2, "急停", "Bool", "RW"),
            ("coil", 3, "维护模式", "Bool", "RW"),
            ("discrete_input", 0, "运行中", "Bool", "R"),
            ("discrete_input", 1, "启动中", "Bool", "R"),
            ("discrete_input", 2, "故障", "Bool", "R"),
            ("discrete_input", 3, "报警激活", "Bool", "R"),
            ("holding_register", 0, "目标压力", "UInt16 / 100 bar", "RW"),
            ("holding_register", 1, "目标转速", "UInt16 rpm", "RW"),
            ("holding_register", 2, "状态码", "UInt16", "R"),
            ("input_register", 0, "压力", "UInt16 / 100 bar", "R"),
            ("input_register", 1, "温度", "UInt16 / 10 C", "R"),
            ("input_register", 2, "流量", "UInt16 / 10 m3/h", "R"),
            ("input_register", 3, "液位", "UInt16 / 10 %", "R"),
            ("input_register", 4, "电机转速", "UInt16 rpm", "R"),
            ("input_register", 5, "电流", "UInt16 / 100 A", "R"),
            ("input_register", 6, "报警码", "UInt16 bitmask", "R"),
            ("input_register", 7, "质量码", "0 Good / 1 Uncertain / 2 Bad", "R"),
        ],
    )

    try:
        StartTcpServer(context=context, address=(args.host, args.port))
    finally:
        stop_event.set()


def update_loop(device: PumpStationDevice, context: ModbusServerContext, interval: float, stop_event: threading.Event) -> None:
    slave = context[UNIT_ID]
    while not stop_event.is_set():
        # 先读取客户端写入的命令，再推进设备状态，最后刷新只读区。
        coils = list(slave.getValues(1, 0, count=4))
        holding = list(slave.getValues(3, 0, count=4))

        device.set_run_command(bool(coils[0]))
        if bool(coils[1]):
            device.reset_alarm()
            slave.setValues(1, 1, [False])
        device.set_emergency_stop(bool(coils[2]))
        device.set_maintenance_mode(bool(coils[3]))
        if holding[0] > 0:
            device.set_target_pressure(float(holding[0]) / 100.0)
        if holding[1] > 0:
            device.set_target_speed(float(holding[1]))

        snapshot = device.tick()
        if holding[0] == 0:
            slave.setValues(3, 0, [int(snapshot.target_pressure_bar * 100)])
        if holding[1] == 0:
            slave.setValues(3, 1, [int(snapshot.target_speed_rpm)])

        slave.setValues(2, 0, [
            snapshot.state == "running",
            snapshot.state == "starting",
            snapshot.state in {"fault", "estop"},
            snapshot.alarm_active,
            snapshot.sensor_fault,
            snapshot.comm_flap,
        ])
        slave.setValues(3, 2, [snapshot.state_code, snapshot.reset_counter])
        slave.setValues(4, 0, [
            int(snapshot.pressure_bar * 100),
            int(snapshot.temperature_c * 10),
            int(snapshot.flow_m3h * 10),
            int(snapshot.level_percent * 10),
            int(snapshot.speed_rpm),
            int(snapshot.current_a * 100),
            snapshot.alarm_code,
            snapshot.quality_code,
        ])
        LOGGER.debug("Modbus snapshot: %s", snapshot)
        stop_event.wait(interval)


def initialize_store(store: ModbusSlaveContext, snapshot) -> None:
    store.setValues(1, 0, [
        snapshot.run_command,
        False,
        snapshot.emergency_stop,
        snapshot.maintenance_mode,
    ])
    store.setValues(3, 0, [
        int(snapshot.target_pressure_bar * 100),
        int(snapshot.target_speed_rpm),
        snapshot.state_code,
        snapshot.reset_counter,
    ])


if __name__ == "__main__":
    main()
