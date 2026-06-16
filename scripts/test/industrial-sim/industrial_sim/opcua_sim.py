from __future__ import annotations

import asyncio
import logging
import sys
from pathlib import Path
from typing import Any

if __package__ in {None, ""}:
    sys.path.append(str(Path(__file__).resolve().parents[1]))

from asyncua import Server, ua

from industrial_sim.common import build_parser, configure_logging, print_table
from industrial_sim.device_model import PumpStationDevice

LOGGER = logging.getLogger("industrial_sim.opcua")

NS_PREFIX = "industrial-sim"


async def main_async() -> None:
    parser = build_parser("InduForge OPC UA 泵站模拟设备", 18540)
    args = parser.parse_args()
    configure_logging(args.verbose)
    if not args.verbose:
        logging.getLogger("asyncua").setLevel(logging.WARNING)

    device = PumpStationDevice(args.scenario)
    endpoint = f"opc.tcp://{args.host}:{args.port}/induforge/sim"
    server = Server()
    await server.init()
    server.set_endpoint(endpoint)
    server.set_server_name("InduForge PumpStation OPC UA Simulator")
    namespace = await server.register_namespace("urn:induforge:industrial-sim")

    nodes = await build_address_space(server, namespace, device)

    print(f"OPC UA 模拟设备已启动: {endpoint}, scenario={args.scenario}")
    print_table(
        "OPC UA 关键节点",
        [
            ("路径", "类型", "读写"),
            ("Objects/InduForgeSim/PumpA/Status/State", "String", "R"),
            ("Objects/InduForgeSim/PumpA/Telemetry/PressureBar", "Double", "R"),
            ("Objects/InduForgeSim/PumpA/Telemetry/TemperatureC", "Double", "R"),
            ("Objects/InduForgeSim/PumpA/Telemetry/FlowM3H", "Double", "R"),
            ("Objects/InduForgeSim/PumpA/Telemetry/Quality", "String", "R"),
            ("Objects/InduForgeSim/PumpA/Command/RunCommand", "Boolean", "RW"),
            ("Objects/InduForgeSim/PumpA/Command/ResetCommand", "Boolean pulse", "RW"),
            ("Objects/InduForgeSim/PumpA/Command/TargetPressureBar", "Double", "RW"),
            ("Objects/InduForgeSim/PumpA/Command/TargetSpeedRPM", "Double", "RW"),
            ("Objects/InduForgeSim/Plant01/Area-Water/Line-01/Skid-Pump-01/Pump-P101/Telemetry/PressureBar", "Double", "R"),
            ("Objects/InduForgeSim/Plant01/Area-Water/Line-01/Tank-T101/Telemetry/LevelPercent", "Double", "R"),
            ("Objects/InduForgeSim/Plant01/Area-Water/Line-01/Valve-XV101/Command/OpenCommand", "Boolean", "RW"),
        ],
    )

    async with server:
        while True:
            await sync_commands(device, nodes)
            snapshot = device.tick()
            await write_snapshot(nodes, snapshot)
            LOGGER.debug("OPC UA snapshot: %s", snapshot)
            await asyncio.sleep(args.update_ms / 1000.0)


async def build_address_space(server: Server, namespace: int, device: PumpStationDevice) -> dict[str, ua.NodeId]:
    objects = server.nodes.objects
    root = await add_object(objects, namespace, "InduForgeSim", "InduForgeSim")
    pump = await add_object(root, namespace, "InduForgeSim.PumpA", "PumpA")
    status = await add_object(pump, namespace, "InduForgeSim.PumpA.Status", "Status")
    telemetry = await add_object(pump, namespace, "InduForgeSim.PumpA.Telemetry", "Telemetry")
    command = await add_object(pump, namespace, "InduForgeSim.PumpA.Command", "Command")
    snapshot = device.tick()

    nodes = {
        "state": await add_variable(status, namespace, "InduForgeSim.PumpA.Status.State", "State", snapshot.state, "String", "设备状态"),
        "state_code": await add_variable(status, namespace, "InduForgeSim.PumpA.Status.StateCode", "StateCode", snapshot.state_code, "Int64", "设备状态码"),
        "alarm_code": await add_variable(status, namespace, "InduForgeSim.PumpA.Status.AlarmCode", "AlarmCode", snapshot.alarm_code, "Int64", "报警 bitmask"),
        "alarm_active": await add_variable(status, namespace, "InduForgeSim.PumpA.Status.AlarmActive", "AlarmActive", snapshot.alarm_active, "Boolean", "报警激活"),
        "sensor_fault": await add_variable(status, namespace, "InduForgeSim.PumpA.Status.SensorFault", "SensorFault", snapshot.sensor_fault, "Boolean", "传感器故障"),
        "comm_flap": await add_variable(status, namespace, "InduForgeSim.PumpA.Status.CommunicationFlap", "CommunicationFlap", snapshot.comm_flap, "Boolean", "通信质量波动"),
        "pressure": await add_variable(telemetry, namespace, "InduForgeSim.PumpA.Telemetry.PressureBar", "PressureBar", snapshot.pressure_bar, "Double", "出口压力", unit="bar", eu_range=(0, 10)),
        "temperature": await add_variable(telemetry, namespace, "InduForgeSim.PumpA.Telemetry.TemperatureC", "TemperatureC", snapshot.temperature_c, "Double", "电机温度", unit="degC", eu_range=(0, 120)),
        "flow": await add_variable(telemetry, namespace, "InduForgeSim.PumpA.Telemetry.FlowM3H", "FlowM3H", snapshot.flow_m3h, "Double", "瞬时流量", unit="m3/h", eu_range=(0, 120)),
        "level": await add_variable(telemetry, namespace, "InduForgeSim.PumpA.Telemetry.LevelPercent", "LevelPercent", snapshot.level_percent, "Double", "水池液位", unit="%", eu_range=(0, 100)),
        "speed": await add_variable(telemetry, namespace, "InduForgeSim.PumpA.Telemetry.SpeedRPM", "SpeedRPM", snapshot.speed_rpm, "Double", "电机转速", unit="rpm", eu_range=(0, 3000)),
        "current": await add_variable(telemetry, namespace, "InduForgeSim.PumpA.Telemetry.CurrentA", "CurrentA", snapshot.current_a, "Double", "电机电流", unit="A", eu_range=(0, 30)),
        "quality": await add_variable(telemetry, namespace, "InduForgeSim.PumpA.Telemetry.Quality", "Quality", snapshot.quality, "String", "采集质量"),
        "timestamp": await add_variable(telemetry, namespace, "InduForgeSim.PumpA.Telemetry.Timestamp", "Timestamp", snapshot.timestamp, "String", "设备侧时间戳"),
        "run_command": await add_variable(command, namespace, "InduForgeSim.PumpA.Command.RunCommand", "RunCommand", snapshot.run_command, "Boolean", "运行命令", writable=True),
        "reset_command": await add_variable(command, namespace, "InduForgeSim.PumpA.Command.ResetCommand", "ResetCommand", False, "Boolean", "报警复位脉冲", writable=True),
        "emergency_stop": await add_variable(command, namespace, "InduForgeSim.PumpA.Command.EmergencyStop", "EmergencyStop", snapshot.emergency_stop, "Boolean", "急停命令", writable=True),
        "maintenance_mode": await add_variable(command, namespace, "InduForgeSim.PumpA.Command.MaintenanceMode", "MaintenanceMode", snapshot.maintenance_mode, "Boolean", "维护模式", writable=True),
        "target_pressure": await add_variable(command, namespace, "InduForgeSim.PumpA.Command.TargetPressureBar", "TargetPressureBar", snapshot.target_pressure_bar, "Double", "目标压力", writable=True, unit="bar", eu_range=(0.5, 9.5)),
        "target_speed": await add_variable(command, namespace, "InduForgeSim.PumpA.Command.TargetSpeedRPM", "TargetSpeedRPM", snapshot.target_speed_rpm, "Double", "目标转速", writable=True, unit="rpm", eu_range=(300, 2900)),
    }

    await add_industrial_browse_tree(root, namespace, nodes, snapshot)
    return nodes


async def add_industrial_browse_tree(
    root: ua.NodeId,
    namespace: int,
    nodes: dict[str, ua.NodeId],
    snapshot,
) -> None:
    """创建更接近现场设备的 OPC UA 浏览树。

    输入为现有服务器根节点和设备快照；输出是挂载到地址空间的一组对象/变量。
    异常由 asyncua 原样抛出，让启动阶段快速失败，避免工作台连接到半成品地址空间。
    """

    plant = await add_object(root, namespace, "Plant01", "Plant01", "生产厂站")
    water_area = await add_object(plant, namespace, "Plant01.Area-Water", "Area-Water", "供水工艺区")
    utilities = await add_object(plant, namespace, "Plant01.Area-Utilities", "Area-Utilities", "公辅系统")
    line = await add_object(water_area, namespace, "Plant01.Area-Water.Line-01", "Line-01", "一号供水线")
    pump_skid = await add_object(line, namespace, "Plant01.Area-Water.Line-01.Skid-Pump-01", "Skid-Pump-01", "泵组撬")

    await add_pump_device(pump_skid, namespace, nodes, "P101", "P101", snapshot, primary=True)
    await add_pump_device(pump_skid, namespace, nodes, "P102", "P102", snapshot, primary=False)
    await add_tank_device(line, namespace, nodes, snapshot)
    await add_valve_device(line, namespace, nodes, snapshot)
    await add_instrument_device(line, namespace, nodes, snapshot)
    await add_utilities_tree(utilities, namespace, nodes, snapshot)


async def add_pump_device(
    parent: ua.NodeId,
    namespace: int,
    nodes: dict[str, ua.NodeId],
    tag: str,
    node_segment: str,
    snapshot,
    *,
    primary: bool,
) -> None:
    device = await add_object(parent, namespace, f"Plant01.Area-Water.Line-01.Skid-Pump-01.Pump-{node_segment}", f"Pump-{tag}", f"离心泵 {tag}")
    status = await add_object(device, namespace, f"Plant01.Area-Water.Line-01.Skid-Pump-01.Pump-{node_segment}.Status", "Status")
    telemetry = await add_object(device, namespace, f"Plant01.Area-Water.Line-01.Skid-Pump-01.Pump-{node_segment}.Telemetry", "Telemetry")
    command = await add_object(device, namespace, f"Plant01.Area-Water.Line-01.Skid-Pump-01.Pump-{node_segment}.Command", "Command")
    diagnostics = await add_object(device, namespace, f"Plant01.Area-Water.Line-01.Skid-Pump-01.Pump-{node_segment}.Diagnostics", "Diagnostics")

    if primary:
        nodes["plant_p101_state"] = await add_variable(status, namespace, "Plant01.Area-Water.Line-01.Skid-Pump-01.Pump-P101.Status.State", "State", snapshot.state, "String", "泵运行状态")
        nodes["plant_p101_alarm_code"] = await add_variable(status, namespace, "Plant01.Area-Water.Line-01.Skid-Pump-01.Pump-P101.Status.AlarmCode", "AlarmCode", snapshot.alarm_code, "Int64", "泵报警 bitmask")
        nodes["plant_p101_alarm_active"] = await add_variable(status, namespace, "Plant01.Area-Water.Line-01.Skid-Pump-01.Pump-P101.Status.AlarmActive", "AlarmActive", snapshot.alarm_active, "Boolean", "泵报警激活")
        nodes["plant_p101_pressure"] = await add_variable(telemetry, namespace, "Plant01.Area-Water.Line-01.Skid-Pump-01.Pump-P101.Telemetry.PressureBar", "PressureBar", snapshot.pressure_bar, "Double", "泵出口压力", unit="bar", eu_range=(0, 10))
        nodes["plant_p101_temperature"] = await add_variable(telemetry, namespace, "Plant01.Area-Water.Line-01.Skid-Pump-01.Pump-P101.Telemetry.TemperatureC", "TemperatureC", snapshot.temperature_c, "Double", "泵电机温度", unit="degC", eu_range=(0, 120))
        nodes["plant_p101_flow"] = await add_variable(telemetry, namespace, "Plant01.Area-Water.Line-01.Skid-Pump-01.Pump-P101.Telemetry.FlowM3H", "FlowM3H", snapshot.flow_m3h, "Double", "泵出口流量", unit="m3/h", eu_range=(0, 120))
        nodes["plant_p101_speed"] = await add_variable(telemetry, namespace, "Plant01.Area-Water.Line-01.Skid-Pump-01.Pump-P101.Telemetry.SpeedRPM", "SpeedRPM", snapshot.speed_rpm, "Double", "泵电机转速", unit="rpm", eu_range=(0, 3000))
        nodes["plant_p101_current"] = await add_variable(telemetry, namespace, "Plant01.Area-Water.Line-01.Skid-Pump-01.Pump-P101.Telemetry.CurrentA", "CurrentA", snapshot.current_a, "Double", "泵电机电流", unit="A", eu_range=(0, 30))
        nodes["plant_p101_run_command"] = await add_variable(command, namespace, "Plant01.Area-Water.Line-01.Skid-Pump-01.Pump-P101.Command.RunCommand", "RunCommand", snapshot.run_command, "Boolean", "泵运行命令", writable=True)
        nodes["plant_p101_reset_command"] = await add_variable(command, namespace, "Plant01.Area-Water.Line-01.Skid-Pump-01.Pump-P101.Command.ResetCommand", "ResetCommand", False, "Boolean", "泵报警复位脉冲", writable=True)
        nodes["plant_p101_target_pressure"] = await add_variable(command, namespace, "Plant01.Area-Water.Line-01.Skid-Pump-01.Pump-P101.Command.TargetPressureBar", "TargetPressureBar", snapshot.target_pressure_bar, "Double", "目标压力", writable=True, unit="bar", eu_range=(0.5, 9.5))
        nodes["plant_p101_quality"] = await add_variable(diagnostics, namespace, "Plant01.Area-Water.Line-01.Skid-Pump-01.Pump-P101.Diagnostics.Quality", "Quality", snapshot.quality, "String", "采集质量")
        nodes["plant_p101_timestamp"] = await add_variable(diagnostics, namespace, "Plant01.Area-Water.Line-01.Skid-Pump-01.Pump-P101.Diagnostics.Timestamp", "Timestamp", snapshot.timestamp, "String", "设备侧时间戳")
        return

    await add_variable(status, namespace, "Plant01.Area-Water.Line-01.Skid-Pump-01.Pump-P102.Status.State", "State", "standby", "String", "备用泵状态")
    nodes["plant_p102_ready"] = await add_variable(status, namespace, "Plant01.Area-Water.Line-01.Skid-Pump-01.Pump-P102.Status.Ready", "Ready", True, "Boolean", "备用泵就绪")
    nodes["plant_p102_auto_available"] = await add_variable(status, namespace, "Plant01.Area-Water.Line-01.Skid-Pump-01.Pump-P102.Status.AutoAvailable", "AutoAvailable", True, "Boolean", "自动投入允许")
    nodes["plant_p102_runtime_hours"] = await add_variable(telemetry, namespace, "Plant01.Area-Water.Line-01.Skid-Pump-01.Pump-P102.Telemetry.RuntimeHours", "RuntimeHours", 1286.4, "Double", "累计运行小时", unit="h")
    nodes["plant_p102_bearing_temperature"] = await add_variable(telemetry, namespace, "Plant01.Area-Water.Line-01.Skid-Pump-01.Pump-P102.Telemetry.BearingTemperatureC", "BearingTemperatureC", 32.5, "Double", "轴承温度", unit="degC", eu_range=(0, 100))
    nodes["plant_p102_vibration"] = await add_variable(telemetry, namespace, "Plant01.Area-Water.Line-01.Skid-Pump-01.Pump-P102.Telemetry.VibrationMMs", "VibrationMMs", 1.2, "Double", "振动速度", unit="mm/s", eu_range=(0, 20))
    nodes["plant_p102_run_command"] = await add_variable(command, namespace, "Plant01.Area-Water.Line-01.Skid-Pump-01.Pump-P102.Command.RunCommand", "RunCommand", False, "Boolean", "备用泵运行命令", writable=True)
    await add_variable(diagnostics, namespace, "Plant01.Area-Water.Line-01.Skid-Pump-01.Pump-P102.Diagnostics.LastMaintenance", "LastMaintenance", "2026-05-18 09:00:00", "String", "最近维护时间")


async def add_tank_device(parent: ua.NodeId, namespace: int, nodes: dict[str, ua.NodeId], snapshot) -> None:
    tank = await add_object(parent, namespace, "Plant01.Area-Water.Line-01.Tank-T101", "Tank-T101", "原水缓冲罐")
    telemetry = await add_object(tank, namespace, "Plant01.Area-Water.Line-01.Tank-T101.Telemetry", "Telemetry")
    status = await add_object(tank, namespace, "Plant01.Area-Water.Line-01.Tank-T101.Status", "Status")
    nodes["plant_t101_level"] = await add_variable(telemetry, namespace, "Plant01.Area-Water.Line-01.Tank-T101.Telemetry.LevelPercent", "LevelPercent", snapshot.level_percent, "Double", "液位百分比", unit="%", eu_range=(0, 100))
    nodes["plant_t101_volume"] = await add_variable(telemetry, namespace, "Plant01.Area-Water.Line-01.Tank-T101.Telemetry.VolumeM3", "VolumeM3", snapshot.level_percent * 3.5, "Double", "估算库存体积", unit="m3", eu_range=(0, 350))
    nodes["plant_t101_inlet_flow"] = await add_variable(telemetry, namespace, "Plant01.Area-Water.Line-01.Tank-T101.Telemetry.InletFlowM3H", "InletFlowM3H", snapshot.flow_m3h * 1.08, "Double", "入口流量", unit="m3/h", eu_range=(0, 150))
    nodes["plant_t101_low_level"] = await add_variable(status, namespace, "Plant01.Area-Water.Line-01.Tank-T101.Status.LowLevelAlarm", "LowLevelAlarm", snapshot.level_percent < 15, "Boolean", "低液位报警")
    nodes["plant_t101_high_level"] = await add_variable(status, namespace, "Plant01.Area-Water.Line-01.Tank-T101.Status.HighLevelAlarm", "HighLevelAlarm", snapshot.level_percent > 92, "Boolean", "高液位报警")


async def add_valve_device(parent: ua.NodeId, namespace: int, nodes: dict[str, ua.NodeId], snapshot) -> None:
    valve = await add_object(parent, namespace, "Plant01.Area-Water.Line-01.Valve-XV101", "Valve-XV101", "出口电动阀")
    status = await add_object(valve, namespace, "Plant01.Area-Water.Line-01.Valve-XV101.Status", "Status")
    command = await add_object(valve, namespace, "Plant01.Area-Water.Line-01.Valve-XV101.Command", "Command")
    telemetry = await add_object(valve, namespace, "Plant01.Area-Water.Line-01.Valve-XV101.Telemetry", "Telemetry")
    is_open = snapshot.state in {"starting", "running"}
    nodes["plant_xv101_opened"] = await add_variable(status, namespace, "Plant01.Area-Water.Line-01.Valve-XV101.Status.Opened", "Opened", is_open, "Boolean", "阀门开到位")
    nodes["plant_xv101_closed"] = await add_variable(status, namespace, "Plant01.Area-Water.Line-01.Valve-XV101.Status.Closed", "Closed", not is_open, "Boolean", "阀门关到位")
    nodes["plant_xv101_position"] = await add_variable(telemetry, namespace, "Plant01.Area-Water.Line-01.Valve-XV101.Telemetry.PositionPercent", "PositionPercent", 100.0 if is_open else 0.0, "Double", "阀门开度", unit="%", eu_range=(0, 100))
    nodes["plant_xv101_open_command"] = await add_variable(command, namespace, "Plant01.Area-Water.Line-01.Valve-XV101.Command.OpenCommand", "OpenCommand", is_open, "Boolean", "开阀命令", writable=True)
    nodes["plant_xv101_close_command"] = await add_variable(command, namespace, "Plant01.Area-Water.Line-01.Valve-XV101.Command.CloseCommand", "CloseCommand", not is_open, "Boolean", "关阀命令", writable=True)


async def add_instrument_device(parent: ua.NodeId, namespace: int, nodes: dict[str, ua.NodeId], snapshot) -> None:
    instruments = await add_object(parent, namespace, "Plant01.Area-Water.Line-01.Instruments", "Instruments", "现场仪表")
    pit = await add_object(instruments, namespace, "Plant01.Area-Water.Line-01.Instruments.PIT-101", "PIT-101", "压力变送器")
    fit = await add_object(instruments, namespace, "Plant01.Area-Water.Line-01.Instruments.FIT-101", "FIT-101", "流量计")
    lit = await add_object(instruments, namespace, "Plant01.Area-Water.Line-01.Instruments.LIT-101", "LIT-101", "液位计")
    nodes["plant_pit101_pv"] = await add_variable(pit, namespace, "Plant01.Area-Water.Line-01.Instruments.PIT-101.PV", "PV", snapshot.pressure_bar, "Double", "压力当前值", unit="bar", eu_range=(0, 10))
    nodes["plant_pit101_signal"] = await add_variable(pit, namespace, "Plant01.Area-Water.Line-01.Instruments.PIT-101.SignalMA", "SignalMA", 4.0 + snapshot.pressure_bar / 10.0 * 16.0, "Double", "4-20mA 信号", unit="mA", eu_range=(4, 20))
    nodes["plant_fit101_pv"] = await add_variable(fit, namespace, "Plant01.Area-Water.Line-01.Instruments.FIT-101.PV", "PV", snapshot.flow_m3h, "Double", "流量当前值", unit="m3/h", eu_range=(0, 120))
    nodes["plant_lit101_pv"] = await add_variable(lit, namespace, "Plant01.Area-Water.Line-01.Instruments.LIT-101.PV", "PV", snapshot.level_percent, "Double", "液位当前值", unit="%", eu_range=(0, 100))


async def add_utilities_tree(parent: ua.NodeId, namespace: int, nodes: dict[str, ua.NodeId], snapshot) -> None:
    power = await add_object(parent, namespace, "Plant01.Area-Utilities.Power", "Power", "电气系统")
    air = await add_object(parent, namespace, "Plant01.Area-Utilities.InstrumentAir", "InstrumentAir", "仪表空气")
    nodes["plant_power_voltage"] = await add_variable(power, namespace, "Plant01.Area-Utilities.Power.BusVoltageV", "BusVoltageV", 380.0, "Double", "低压母线电压", unit="V", eu_range=(320, 430))
    nodes["plant_power_current"] = await add_variable(power, namespace, "Plant01.Area-Utilities.Power.TotalCurrentA", "TotalCurrentA", snapshot.current_a + 4.5, "Double", "总电流", unit="A", eu_range=(0, 120))
    nodes["plant_air_pressure"] = await add_variable(air, namespace, "Plant01.Area-Utilities.InstrumentAir.HeaderPressureBar", "HeaderPressureBar", 6.2, "Double", "仪表空气总管压力", unit="bar", eu_range=(0, 10))


async def add_object(
    parent: ua.NodeId,
    namespace: int,
    node_id: str,
    browse_name: str,
    description: str | None = None,
) -> ua.NodeId:
    node = await parent.add_object(ua.NodeId(f"{NS_PREFIX}.{node_id}", namespace), browse_name)
    if description:
        await write_description(node, description)
    return node


async def add_variable(
    parent: ua.NodeId,
    namespace: int,
    node_id: str,
    browse_name: str,
    value: Any,
    data_type: str,
    description: str,
    *,
    writable: bool = False,
    unit: str | None = None,
    eu_range: tuple[float, float] | None = None,
) -> ua.NodeId:
    node = await parent.add_variable(ua.NodeId(f"{NS_PREFIX}.{node_id}", namespace), browse_name, value)
    await write_description(node, description)
    await add_metadata(node, namespace, data_type, unit, eu_range)
    if writable:
        await node.set_writable()
    return node


async def write_description(node: ua.NodeId, description: str) -> None:
    await node.write_attribute(
        ua.AttributeIds.Description,
        ua.DataValue(ua.LocalizedText(description)),
    )


async def add_metadata(
    node: ua.NodeId,
    namespace: int,
    data_type: str,
    unit: str | None,
    eu_range: tuple[float, float] | None,
) -> None:
    # 部分工作台会读取属性节点辅助导入，保留轻量属性即可覆盖常见元数据场景。
    await node.add_property(ua.NodeId(f"{node.nodeid.Identifier}.DataTypeName", namespace), "DataTypeName", data_type)
    if unit:
        await node.add_property(ua.NodeId(f"{node.nodeid.Identifier}.EngineeringUnit", namespace), "EngineeringUnit", unit)
    if eu_range:
        await node.add_property(ua.NodeId(f"{node.nodeid.Identifier}.EURange", namespace), "EURange", f"{eu_range[0]}..{eu_range[1]}")


async def sync_commands(device: PumpStationDevice, nodes: dict[str, ua.NodeId]) -> None:
    snapshot = device.snapshot_without_tick()
    device.set_run_command(await read_bool_command_alias(nodes, ["run_command", "plant_p101_run_command"], snapshot.run_command))
    if await read_bool_pulse_alias(nodes, ["reset_command", "plant_p101_reset_command"]):
        device.reset_alarm()
        await nodes["reset_command"].write_value(False)
        await nodes["plant_p101_reset_command"].write_value(False)
    device.set_emergency_stop(bool(await nodes["emergency_stop"].read_value()))
    device.set_maintenance_mode(bool(await nodes["maintenance_mode"].read_value()))
    device.set_target_pressure(await read_float_command_alias(nodes, ["target_pressure", "plant_p101_target_pressure"], snapshot.target_pressure_bar))
    device.set_target_speed(float(await nodes["target_speed"].read_value()))


async def read_bool_command_alias(nodes: dict[str, ua.NodeId], keys: list[str], current: bool) -> bool:
    """读取一组布尔命令别名。

    输入是多个可写节点和设备当前值；输出是本轮应写入设备模型的值。
    当某个别名被用户改成不同于当前设备值时，将它视为最新命令；随后快照会把别名同步回一致状态。
    """

    selected = current
    for key in keys:
        value = bool(await nodes[key].read_value())
        if value != current:
            selected = value
    return selected


async def read_bool_pulse_alias(nodes: dict[str, ua.NodeId], keys: list[str]) -> bool:
    """读取脉冲命令别名，任一节点为 true 都触发一次动作。"""

    for key in keys:
        if bool(await nodes[key].read_value()):
            return True
    return False


async def read_float_command_alias(nodes: dict[str, ua.NodeId], keys: list[str], current: float) -> float:
    """读取数值命令别名，返回和当前值不同的最后一个写入候选。"""

    selected = current
    for key in keys:
        value = float(await nodes[key].read_value())
        if abs(value - current) > 0.000001:
            selected = value
    return selected


async def write_snapshot(nodes: dict[str, ua.NodeId], snapshot) -> None:
    await nodes["state"].write_value(snapshot.state)
    await nodes["state_code"].write_value(snapshot.state_code)
    await nodes["alarm_code"].write_value(snapshot.alarm_code)
    await nodes["alarm_active"].write_value(snapshot.alarm_active)
    await nodes["sensor_fault"].write_value(snapshot.sensor_fault)
    await nodes["comm_flap"].write_value(snapshot.comm_flap)
    await nodes["pressure"].write_value(snapshot.pressure_bar)
    await nodes["temperature"].write_value(snapshot.temperature_c)
    await nodes["flow"].write_value(snapshot.flow_m3h)
    await nodes["level"].write_value(snapshot.level_percent)
    await nodes["speed"].write_value(snapshot.speed_rpm)
    await nodes["current"].write_value(snapshot.current_a)
    await nodes["quality"].write_value(snapshot.quality)
    await nodes["timestamp"].write_value(snapshot.timestamp)
    await nodes["run_command"].write_value(snapshot.run_command)
    await nodes["emergency_stop"].write_value(snapshot.emergency_stop)
    await nodes["maintenance_mode"].write_value(snapshot.maintenance_mode)
    await nodes["target_pressure"].write_value(snapshot.target_pressure_bar)
    await nodes["target_speed"].write_value(snapshot.target_speed_rpm)
    await nodes["plant_p101_state"].write_value(snapshot.state)
    await nodes["plant_p101_alarm_code"].write_value(snapshot.alarm_code)
    await nodes["plant_p101_alarm_active"].write_value(snapshot.alarm_active)
    await nodes["plant_p101_pressure"].write_value(snapshot.pressure_bar)
    await nodes["plant_p101_temperature"].write_value(snapshot.temperature_c)
    await nodes["plant_p101_flow"].write_value(snapshot.flow_m3h)
    await nodes["plant_p101_speed"].write_value(snapshot.speed_rpm)
    await nodes["plant_p101_current"].write_value(snapshot.current_a)
    await nodes["plant_p101_run_command"].write_value(snapshot.run_command)
    await nodes["plant_p101_target_pressure"].write_value(snapshot.target_pressure_bar)
    await nodes["plant_p101_quality"].write_value(snapshot.quality)
    await nodes["plant_p101_timestamp"].write_value(snapshot.timestamp)
    await nodes["plant_t101_level"].write_value(snapshot.level_percent)
    await nodes["plant_t101_volume"].write_value(round(snapshot.level_percent * 3.5, 1))
    await nodes["plant_t101_inlet_flow"].write_value(round(snapshot.flow_m3h * 1.08, 1))
    await nodes["plant_t101_low_level"].write_value(snapshot.level_percent < 15)
    await nodes["plant_t101_high_level"].write_value(snapshot.level_percent > 92)
    valve_open = snapshot.state in {"starting", "running"}
    await nodes["plant_xv101_opened"].write_value(valve_open)
    await nodes["plant_xv101_closed"].write_value(not valve_open)
    await nodes["plant_xv101_position"].write_value(100.0 if valve_open else 0.0)
    await nodes["plant_pit101_pv"].write_value(snapshot.pressure_bar)
    await nodes["plant_pit101_signal"].write_value(round(4.0 + snapshot.pressure_bar / 10.0 * 16.0, 2))
    await nodes["plant_fit101_pv"].write_value(snapshot.flow_m3h)
    await nodes["plant_lit101_pv"].write_value(snapshot.level_percent)
    await nodes["plant_power_current"].write_value(round(snapshot.current_a + 4.5, 2))


def main() -> None:
    asyncio.run(main_async())


if __name__ == "__main__":
    main()
