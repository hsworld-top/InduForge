from __future__ import annotations

import asyncio
import logging
import sys
from pathlib import Path

if __package__ in {None, ""}:
    sys.path.append(str(Path(__file__).resolve().parents[1]))

from asyncua import Server, ua

from industrial_sim.common import build_parser, configure_logging, print_table
from industrial_sim.device_model import PumpStationDevice

LOGGER = logging.getLogger("industrial_sim.opcua")


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
    root = await objects.add_object(namespace, "InduForgeSim")
    pump = await root.add_object(namespace, "PumpA")
    status = await pump.add_object(namespace, "Status")
    telemetry = await pump.add_object(namespace, "Telemetry")
    command = await pump.add_object(namespace, "Command")
    snapshot = device.tick()

    nodes = {
        "state": await status.add_variable(namespace, "State", snapshot.state),
        "state_code": await status.add_variable(namespace, "StateCode", snapshot.state_code),
        "alarm_code": await status.add_variable(namespace, "AlarmCode", snapshot.alarm_code),
        "alarm_active": await status.add_variable(namespace, "AlarmActive", snapshot.alarm_active),
        "sensor_fault": await status.add_variable(namespace, "SensorFault", snapshot.sensor_fault),
        "comm_flap": await status.add_variable(namespace, "CommunicationFlap", snapshot.comm_flap),
        "pressure": await telemetry.add_variable(namespace, "PressureBar", snapshot.pressure_bar),
        "temperature": await telemetry.add_variable(namespace, "TemperatureC", snapshot.temperature_c),
        "flow": await telemetry.add_variable(namespace, "FlowM3H", snapshot.flow_m3h),
        "level": await telemetry.add_variable(namespace, "LevelPercent", snapshot.level_percent),
        "speed": await telemetry.add_variable(namespace, "SpeedRPM", snapshot.speed_rpm),
        "current": await telemetry.add_variable(namespace, "CurrentA", snapshot.current_a),
        "quality": await telemetry.add_variable(namespace, "Quality", snapshot.quality),
        "timestamp": await telemetry.add_variable(namespace, "Timestamp", snapshot.timestamp),
        "run_command": await command.add_variable(namespace, "RunCommand", snapshot.run_command),
        "reset_command": await command.add_variable(namespace, "ResetCommand", False),
        "emergency_stop": await command.add_variable(namespace, "EmergencyStop", snapshot.emergency_stop),
        "maintenance_mode": await command.add_variable(namespace, "MaintenanceMode", snapshot.maintenance_mode),
        "target_pressure": await command.add_variable(namespace, "TargetPressureBar", snapshot.target_pressure_bar),
        "target_speed": await command.add_variable(namespace, "TargetSpeedRPM", snapshot.target_speed_rpm),
    }

    # 真实 OPC UA 设备通常会在节点上给出描述，便于检查接入源是否展示元数据。
    for key, node in nodes.items():
        await node.write_attribute(
            ua.AttributeIds.Description,
            ua.DataValue(ua.LocalizedText(f"InduForge pump simulator point: {key}")),
        )
    for key in ["run_command", "reset_command", "emergency_stop", "maintenance_mode", "target_pressure", "target_speed"]:
        await nodes[key].set_writable()
    return nodes


async def sync_commands(device: PumpStationDevice, nodes: dict[str, ua.NodeId]) -> None:
    device.set_run_command(bool(await nodes["run_command"].read_value()))
    if bool(await nodes["reset_command"].read_value()):
        device.reset_alarm()
        await nodes["reset_command"].write_value(False)
    device.set_emergency_stop(bool(await nodes["emergency_stop"].read_value()))
    device.set_maintenance_mode(bool(await nodes["maintenance_mode"].read_value()))
    device.set_target_pressure(float(await nodes["target_pressure"].read_value()))
    device.set_target_speed(float(await nodes["target_speed"].read_value()))


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
    await nodes["target_pressure"].write_value(snapshot.target_pressure_bar)
    await nodes["target_speed"].write_value(snapshot.target_speed_rpm)


def main() -> None:
    asyncio.run(main_async())


if __name__ == "__main__":
    main()
