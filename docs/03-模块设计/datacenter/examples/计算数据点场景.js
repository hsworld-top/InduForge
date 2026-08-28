// 使用前请在“变量”面板绑定：
// temperature / pressure / setpoint / queryPoint / computePoint / eventPoint
// 调试输入示例：{ "scenario": "read" }

function failure(result) {
  return result && result.code !== 0 ? result : null
}

function readProcessSnapshot() {
  const temperatureResult = temperature.read()
  const pressureResult = pressure.read()
  if (failure(temperatureResult)) return temperatureResult
  if (failure(pressureResult)) return pressureResult

  const temperatureSample = temperatureResult.data
  const pressureSample = pressureResult.data
  if (temperatureSample.quality !== 'good' || pressureSample.quality !== 'good') {
    return {
      valid: false,
      reason: '输入数据质量不可用',
      quality: {
        temperature: temperatureSample.quality,
        pressure: pressureSample.quality,
      },
    }
  }

  return {
    valid: true,
    temperature: temperatureSample.value,
    pressure: pressureSample.value,
    load: Number(temperatureSample.value) * Number(pressureSample.value),
    observedAt: temperatureSample.observedAt || temperatureSample.timestamp,
    metadata: {
      path: temperature.path,
      name: temperature.displayName,
      dataType: temperature.dataType,
      unit: temperature.unit,
      source: temperature.source,
      attributes: temperature.attributes,
    },
  }
}

function previewSetpoint() {
  const target = Number(ctx.args.target ?? argv[0] ?? 1200)
  if (!Number.isFinite(target)) return { code: 40032, msg: 'target 必须是数字', data: null }
  return setpoint.set(target)
}

function previewQueryExecution() {
  return queryPoint.execute({
    deviceId: ctx.args.deviceId || 'A01',
    limit: Number(ctx.args.limit || 20),
  })
}

function previewComputeRun() {
  const current = temperature.get()
  if (failure(current)) return current
  return computePoint.run({
    temperature: current.data.value,
    requestedBy: 'compute-demo',
  })
}

function previewPublish() {
  return eventPoint.publish({
    type: 'process_snapshot',
    payload: readProcessSnapshot(),
    emittedAt: new Date().toISOString(),
  })
}

switch (String(ctx.args.scenario || 'read')) {
  case 'read':
    return readProcessSnapshot()
  case 'set':
    return previewSetpoint()
  case 'execute':
    return previewQueryExecution()
  case 'run':
    return previewComputeRun()
  case 'publish':
    return previewPublish()
  case 'unsupported':
    return temperature.history({ limit: 10 })
  default:
    return { code: 40032, msg: '未知 scenario', data: null }
}
