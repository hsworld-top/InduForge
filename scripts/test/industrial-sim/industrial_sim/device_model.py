from __future__ import annotations

import math
import random
import threading
import time
from dataclasses import dataclass
from typing import Any

from industrial_sim.common import clamp


STATE_CODES = {
    "stopped": 0,
    "starting": 1,
    "running": 2,
    "fault": 3,
    "estop": 4,
    "maintenance": 5,
}

QUALITY_CODES = {
    "Good": 0,
    "Uncertain": 1,
    "Bad": 2,
}

ALARM_HIGH_TEMP = 1
ALARM_HIGH_PRESSURE = 2
ALARM_LOW_LEVEL = 4
ALARM_SENSOR_FAULT = 8
ALARM_ESTOP = 16


@dataclass(frozen=True)
class DeviceSnapshot:
    state: str
    state_code: int
    run_command: bool
    reset_counter: int
    emergency_stop: bool
    maintenance_mode: bool
    target_pressure_bar: float
    target_speed_rpm: float
    pressure_bar: float
    temperature_c: float
    flow_m3h: float
    level_percent: float
    speed_rpm: float
    current_a: float
    alarm_code: int
    alarm_active: bool
    sensor_fault: bool
    comm_flap: bool
    quality: str
    quality_code: int
    timestamp: str

    def as_dict(self) -> dict[str, Any]:
        return self.__dict__.copy()


class PumpStationDevice:
    """共享泵站模型。

    模型有意保留启动爬升、报警锁存、写命令反馈这些真实设备常见行为，
    用于暴露接入源在读写、质量码、地址映射和诊断方面的体验问题。
    """

    def __init__(self, scenario: str = "normal", seed: int = 23) -> None:
        self.scenario = scenario
        self._lock = threading.RLock()
        self._random = random.Random(seed)
        self._created_at = time.monotonic()
        self._last_tick = self._created_at
        self._state_changed_at = self._created_at

        self.state = "stopped"
        self.run_command = False
        self.reset_counter = 0
        self.emergency_stop = False
        self.maintenance_mode = False
        self.target_pressure_bar = 4.2
        self.target_speed_rpm = 1450.0

        self.pressure_bar = 0.25
        self.temperature_c = 24.0
        self.flow_m3h = 0.0
        self.level_percent = 78.0
        self.speed_rpm = 0.0
        self.current_a = 0.0
        self.alarm_code = 0
        self.sensor_fault = False
        self.comm_flap = False
        self.quality = "Good"

        if scenario == "startup":
            self.run_command = True
            self._set_state("starting")
        elif scenario == "alarm":
            self.run_command = True
            self.alarm_code = ALARM_HIGH_PRESSURE
            self._set_state("fault")

    def set_run_command(self, value: bool) -> None:
        with self._lock:
            self.run_command = value

    def reset_alarm(self) -> None:
        with self._lock:
            self.reset_counter += 1
            if self.state in {"fault", "estop"} and not self.emergency_stop:
                self.alarm_code = 0
                self.sensor_fault = False
                self._set_state("stopped")

    def set_emergency_stop(self, value: bool) -> None:
        with self._lock:
            self.emergency_stop = value
            if value:
                self.run_command = False
                self.alarm_code |= ALARM_ESTOP
                self._set_state("estop")

    def set_maintenance_mode(self, value: bool) -> None:
        with self._lock:
            self.maintenance_mode = value
            if value:
                self.run_command = False
                self._set_state("maintenance")
            elif self.state == "maintenance":
                self._set_state("stopped")

    def set_target_pressure(self, value: float) -> None:
        with self._lock:
            self.target_pressure_bar = clamp(value, 0.5, 9.5)

    def set_target_speed(self, value: float) -> None:
        with self._lock:
            self.target_speed_rpm = clamp(value, 300.0, 2900.0)

    def tick(self) -> DeviceSnapshot:
        with self._lock:
            now = time.monotonic()
            dt = max(0.001, min(now - self._last_tick, 2.0))
            self._last_tick = now

            self._apply_scenario(now - self._created_at)
            self._apply_state_machine(now)
            self._update_process_values(dt, now)
            self._update_alarms()
            return self.snapshot_without_tick()

    def snapshot_without_tick(self) -> DeviceSnapshot:
        state_code = STATE_CODES[self.state]
        return DeviceSnapshot(
            state=self.state,
            state_code=state_code,
            run_command=self.run_command,
            reset_counter=self.reset_counter,
            emergency_stop=self.emergency_stop,
            maintenance_mode=self.maintenance_mode,
            target_pressure_bar=round(self.target_pressure_bar, 2),
            target_speed_rpm=round(self.target_speed_rpm, 1),
            pressure_bar=round(self.pressure_bar, 2),
            temperature_c=round(self.temperature_c, 1),
            flow_m3h=round(self.flow_m3h, 1),
            level_percent=round(self.level_percent, 1),
            speed_rpm=round(self.speed_rpm, 1),
            current_a=round(self.current_a, 2),
            alarm_code=self.alarm_code,
            alarm_active=self.alarm_code != 0 or self.state in {"fault", "estop"},
            sensor_fault=self.sensor_fault,
            comm_flap=self.comm_flap,
            quality=self.quality,
            quality_code=QUALITY_CODES.get(self.quality, 2),
            timestamp=time.strftime("%Y-%m-%d %H:%M:%S", time.localtime()),
        )

    def _set_state(self, state: str) -> None:
        if self.state != state:
            self.state = state
            self._state_changed_at = time.monotonic()

    def _apply_scenario(self, elapsed: float) -> None:
        # 场景负责制造真实联调里常见的问题：噪声、质量波动、间歇通讯和自动报警。
        self.comm_flap = self.scenario == "intermittent" and int(elapsed) % 12 in {9, 10}
        self.sensor_fault = self.scenario == "noisy" and int(elapsed) % 20 in {13, 14}

        if self.scenario == "alarm" and elapsed > 3:
            self.alarm_code |= ALARM_HIGH_PRESSURE
            self._set_state("fault")

        if self.comm_flap:
            self.quality = "Bad"
        elif self.sensor_fault:
            self.quality = "Uncertain"
        else:
            self.quality = "Good"

    def _apply_state_machine(self, now: float) -> None:
        if self.emergency_stop:
            self.alarm_code |= ALARM_ESTOP
            self._set_state("estop")
            return
        if self.maintenance_mode:
            self._set_state("maintenance")
            return
        if self.alarm_code != 0 and self.state not in {"estop", "maintenance"}:
            self._set_state("fault")
            return
        if self.run_command:
            if self.state == "stopped":
                self._set_state("starting")
            elif self.state == "starting" and now - self._state_changed_at >= 5:
                self._set_state("running")
        elif self.state in {"starting", "running"}:
            self._set_state("stopped")

    def _update_process_values(self, dt: float, now: float) -> None:
        if self.state == "running":
            desired_speed = self.target_speed_rpm
            desired_pressure = self.target_pressure_bar
        elif self.state == "starting":
            desired_speed = self.target_speed_rpm * 0.55
            desired_pressure = self.target_pressure_bar * 0.45
        else:
            desired_speed = 0.0
            desired_pressure = 0.25

        noise = self._random.uniform(-0.04, 0.04) if self.scenario == "noisy" else 0.0
        self.speed_rpm = self._approach(self.speed_rpm, desired_speed, 420.0 * dt)
        self.pressure_bar = self._approach(self.pressure_bar, desired_pressure + noise, 0.7 * dt)
        self.flow_m3h = self._approach(self.flow_m3h, self.speed_rpm / 42.0, 12.0 * dt)
        self.current_a = self._approach(self.current_a, 1.8 + self.speed_rpm / 180.0, 2.0 * dt)
        self.level_percent = clamp(76.0 + math.sin(now / 13.0) * 4.5 - self.flow_m3h / 120.0, 0.0, 100.0)
        heat_target = 24.0 + self.current_a * 2.8
        self.temperature_c = self._approach(self.temperature_c, heat_target, 0.35 * dt)

    def _update_alarms(self) -> None:
        if self.temperature_c > 76:
            self.alarm_code |= ALARM_HIGH_TEMP
        if self.pressure_bar > 8.5:
            self.alarm_code |= ALARM_HIGH_PRESSURE
        if self.level_percent < 15:
            self.alarm_code |= ALARM_LOW_LEVEL
        if self.sensor_fault:
            self.alarm_code |= ALARM_SENSOR_FAULT

    @staticmethod
    def _approach(current: float, target: float, step: float) -> float:
        if current < target:
            return min(current + step, target)
        return max(current - step, target)

