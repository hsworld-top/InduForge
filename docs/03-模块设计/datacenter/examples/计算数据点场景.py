# 使用前请在“变量”面板绑定：
# temperature / pressure / setpoint / queryPoint / computePoint / eventPoint
# 调试输入示例：{ "scenario": "read" }

def read_process_snapshot():
    temperature_result = temperature.read()
    pressure_result = pressure.read()
    if temperature_result.code != 0:
        return temperature_result
    if pressure_result.code != 0:
        return pressure_result

    temperature_sample = temperature_result.data
    pressure_sample = pressure_result.data
    if temperature_sample["quality"] != "good" or pressure_sample["quality"] != "good":
        return {
            "valid": False,
            "reason": "输入数据质量不可用",
            "quality": {
                "temperature": temperature_sample["quality"],
                "pressure": pressure_sample["quality"],
            },
        }

    return {
        "valid": True,
        "temperature": temperature_sample["value"],
        "pressure": pressure_sample["value"],
        "load": float(temperature_sample["value"]) * float(pressure_sample["value"]),
        "observedAt": temperature_sample.get("observedAt") or temperature_sample.get("timestamp"),
        "metadata": {
            "path": temperature.path,
            "name": temperature.displayName,
            "dataType": temperature.dataType,
            "unit": temperature.unit,
            "source": temperature.source,
            "attributes": temperature.attributes,
        },
    }

def main(argv, dp, ctx):
    scenario = str(ctx.args.get("scenario") or "read")
    if scenario == "read":
        return read_process_snapshot()
    if scenario == "set":
        target = ctx.args.get("target", argv[0] if len(argv) > 0 else 1200)
        return setpoint.set(float(target))
    if scenario == "execute":
        return queryPoint.execute({"deviceId": ctx.args.get("deviceId", "A01"), "limit": ctx.args.get("limit", 20)})
    if scenario == "run":
        return computePoint.run({"temperature": temperature.get().data, "requestedBy": "compute-demo"})
    if scenario == "publish":
        return eventPoint.publish({"type": "process_snapshot", "payload": read_process_snapshot()})
    if scenario == "unsupported":
        return temperature.history({"limit": 10})
    return {"code": 40032, "msg": "未知 scenario", "data": None}
