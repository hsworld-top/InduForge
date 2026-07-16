from __future__ import annotations

import logging
import struct
import sys
import threading
import time
from pathlib import Path

if __package__ in {None, ""}:
    sys.path.append(str(Path(__file__).resolve().parents[1]))

from snap7.server import Server

try:
    from snap7.server import S7Area
except ImportError:
    S7Area = None

try:
    from snap7.type import SrvArea
except ImportError:  # python-snap7 旧版本兼容
    from snap7.types import srvAreaDB, srvAreaMK, srvAreaPA, srvAreaPE

    class SrvArea:  # type: ignore[no-redef]
        DB = srvAreaDB
        MK = srvAreaMK
        PA = srvAreaPA
        PE = srvAreaPE

from industrial_sim.common import build_parser, configure_logging, print_table
from industrial_sim.device_model import PumpStationDevice

LOGGER = logging.getLogger("industrial_sim.s7")


def main() -> None:
    parser = build_parser("InduForge S7 泵站模拟设备", 18503)
    args = parser.parse_args()
    configure_logging(args.verbose)
    if not args.verbose:
        logging.getLogger("snap7").setLevel(logging.WARNING)

    device = PumpStationDevice(args.scenario)
    server = Server()
    db1 = register_area_buffer(server, SrvArea.DB, "DB", 1, 128)
    marker = register_area_buffer(server, SrvArea.MK, "MK", 0, 64)
    inputs = register_area_buffer(server, SrvArea.PE, "PE", 0, 64)
    outputs = register_area_buffer(server, SrvArea.PA, "PA", 0, 64)
    initialize_buffers(device.tick(), db1, marker, inputs, outputs)

    stop_event = threading.Event()
    updater = threading.Thread(
        target=update_loop,
        args=(device, db1, marker, inputs, outputs, args.update_ms / 1000.0, stop_event),
        daemon=True,
    )
    updater.start()

    print(f"S7 模拟设备已启动: {args.host}:{args.port}, rack=0, slot=1, scenario={args.scenario}")
    print_table(
        "S7 DB1 点表",
        [
            ("地址", "名称", "类型", "读写"),
            ("DB1.DBX0.0", "运行命令", "Bool", "RW"),
            ("DB1.DBX0.1", "报警复位", "Bool 脉冲", "RW"),
            ("DB1.DBX0.2", "急停", "Bool", "RW"),
            ("DB1.DBX0.3", "维护模式", "Bool", "RW"),
            ("DB1.DBX1.0", "运行中", "Bool", "R"),
            ("DB1.DBX1.1", "启动中", "Bool", "R"),
            ("DB1.DBX1.2", "故障", "Bool", "R"),
            ("DB1.DBX1.3", "报警激活", "Bool", "R"),
            ("DB1.DBW2", "状态码", "Int", "R"),
            ("DB1.DBW4", "报警码", "Word", "R"),
            ("DB1.DBD8", "压力", "Real bar", "R"),
            ("DB1.DBD12", "温度", "Real C", "R"),
            ("DB1.DBD16", "流量", "Real m3/h", "R"),
            ("DB1.DBD20", "液位", "Real %", "R"),
            ("DB1.DBD24", "转速", "Real rpm", "R"),
            ("DB1.DBD28", "电流", "Real A", "R"),
            ("DB1.DBD32", "目标压力", "Real bar", "RW"),
            ("DB1.DBD36", "目标转速", "Real rpm", "RW"),
            ("DB1.DBB40", "质量码", "Byte", "R"),
        ],
    )

    try:
        start_server(server, args.host, args.port)
        while True:
            time.sleep(1)
    except KeyboardInterrupt:
        pass
    finally:
        stop_event.set()
        server.stop()


def register_area_buffer(server: Server, area, s7_area_name: str, index: int, size: int) -> bytearray:
    data = bytearray(size)
    server.register_area(area, index, data)
    # python-snap7 3.x 的纯 Python server 会复制入参，这里取回内部缓冲区，
    # 后台刷新线程才能真正更新客户端读到的 DB/M/I/Q 区数据。
    if S7Area is not None and hasattr(server, "memory_areas"):
        return server.memory_areas[(getattr(S7Area, s7_area_name), index)]
    return data


def start_server(server: Server, host: str, port: int) -> None:
    try:
        server.start(host=host, tcp_port=port)
    except TypeError:
        try:
            server.start(tcp_port=port)
        except TypeError:
            try:
                server.start(port=port)
            except TypeError:
                server.start(tcpport=port)


def update_loop(
    device: PumpStationDevice,
    db1: bytearray,
    marker: bytearray,
    inputs: bytearray,
    outputs: bytearray,
    interval: float,
    stop_event: threading.Event,
) -> None:
    while not stop_event.is_set():
        # DB/M/Q 三个区域都接受命令，方便不同 S7 驱动习惯做联调。
        run_command = get_bit(db1, 0, 0) or get_bit(marker, 0, 0) or get_bit(outputs, 0, 0)
        reset_command = get_bit(db1, 0, 1) or get_bit(marker, 0, 1) or get_bit(outputs, 0, 1)
        emergency_stop = get_bit(db1, 0, 2) or get_bit(marker, 0, 2) or get_bit(outputs, 0, 2)
        maintenance_mode = get_bit(db1, 0, 3) or get_bit(marker, 0, 3) or get_bit(outputs, 0, 3)

        device.set_run_command(run_command)
        if reset_command:
            device.reset_alarm()
            set_bit(db1, 0, 1, False)
            set_bit(marker, 0, 1, False)
            set_bit(outputs, 0, 1, False)
        device.set_emergency_stop(emergency_stop)
        device.set_maintenance_mode(maintenance_mode)

        target_pressure = read_real(db1, 32)
        target_speed = read_real(db1, 36)
        if target_pressure > 0:
            device.set_target_pressure(target_pressure)
        if target_speed > 0:
            device.set_target_speed(target_speed)

        snapshot = device.tick()
        write_snapshot(db1, snapshot)
        write_snapshot(inputs, snapshot)
        set_bit(marker, 1, 0, snapshot.state == "running")
        set_bit(marker, 1, 1, snapshot.state == "starting")
        set_bit(marker, 1, 2, snapshot.state in {"fault", "estop"})
        set_bit(marker, 1, 3, snapshot.alarm_active)
        LOGGER.debug("S7 snapshot: %s", snapshot)
        stop_event.wait(interval)


def initialize_buffers(snapshot, db1: bytearray, marker: bytearray, inputs: bytearray, outputs: bytearray) -> None:
    for buffer in (db1, marker, outputs):
        set_bit(buffer, 0, 0, snapshot.run_command)
        set_bit(buffer, 0, 1, False)
        set_bit(buffer, 0, 2, snapshot.emergency_stop)
        set_bit(buffer, 0, 3, snapshot.maintenance_mode)
    write_snapshot(db1, snapshot)
    write_snapshot(inputs, snapshot)


def write_snapshot(buffer: bytearray, snapshot) -> None:
    set_bit(buffer, 1, 0, snapshot.state == "running")
    set_bit(buffer, 1, 1, snapshot.state == "starting")
    set_bit(buffer, 1, 2, snapshot.state in {"fault", "estop"})
    set_bit(buffer, 1, 3, snapshot.alarm_active)
    set_bit(buffer, 1, 4, snapshot.sensor_fault)
    write_int(buffer, 2, snapshot.state_code)
    write_word(buffer, 4, snapshot.alarm_code)
    write_real(buffer, 8, snapshot.pressure_bar)
    write_real(buffer, 12, snapshot.temperature_c)
    write_real(buffer, 16, snapshot.flow_m3h)
    write_real(buffer, 20, snapshot.level_percent)
    write_real(buffer, 24, snapshot.speed_rpm)
    write_real(buffer, 28, snapshot.current_a)
    write_real(buffer, 32, snapshot.target_pressure_bar)
    write_real(buffer, 36, snapshot.target_speed_rpm)
    buffer[40] = snapshot.quality_code & 0xFF


def get_bit(buffer: bytearray, byte_index: int, bit_index: int) -> bool:
    return (buffer[byte_index] & (1 << bit_index)) != 0


def set_bit(buffer: bytearray, byte_index: int, bit_index: int, value: bool) -> None:
    mask = 1 << bit_index
    if value:
        buffer[byte_index] |= mask
    else:
        buffer[byte_index] &= ~mask


def write_int(buffer: bytearray, offset: int, value: int) -> None:
    buffer[offset : offset + 2] = struct.pack(">h", int(value))


def write_word(buffer: bytearray, offset: int, value: int) -> None:
    buffer[offset : offset + 2] = struct.pack(">H", int(value) & 0xFFFF)


def write_real(buffer: bytearray, offset: int, value: float) -> None:
    buffer[offset : offset + 4] = struct.pack(">f", float(value))


def read_real(buffer: bytearray, offset: int) -> float:
    return struct.unpack(">f", bytes(buffer[offset : offset + 4]))[0]


if __name__ == "__main__":
    main()
