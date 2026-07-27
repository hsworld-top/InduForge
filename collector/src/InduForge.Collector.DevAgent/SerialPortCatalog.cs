using System.IO.Ports;

namespace InduForge.Collector.DevAgent;

internal sealed class SerialPortCatalog(
    Func<string[]>? portNameProvider = null,
    AgentFileLogger? logger = null)
{
    private readonly Func<string[]> _portNameProvider = portNameProvider ?? SerialPort.GetPortNames;

    public IReadOnlyList<string> List()
    {
        try
        {
            return _portNameProvider()
                .Where(portName => !string.IsNullOrWhiteSpace(portName))
                .Select(portName => portName.Trim())
                .Distinct(StringComparer.OrdinalIgnoreCase)
                .OrderBy(GetSortKey)
                .ToArray();
        }
        catch (Exception exception)
        {
            // 串口枚举属于辅助资源，失败不能中断 Agent 注册、心跳或任务处理。
            logger?.Warn("serial.ports.enumeration.failed", "无法枚举本机串口，已上报空列表", exception);
            return [];
        }
    }

    private static (int Category, int Number, string Name) GetSortKey(string portName)
    {
        if (portName.StartsWith("COM", StringComparison.OrdinalIgnoreCase) &&
            int.TryParse(portName.AsSpan(3), out var number))
        {
            return (0, number, portName);
        }
        return (1, int.MaxValue, portName);
    }
}
