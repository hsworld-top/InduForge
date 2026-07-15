using System.Security.Cryptography;
using System.Text;
using Microsoft.Win32;

namespace InduForge.Collector.DevAgent;

internal static class MachineIdentityProvider
{
    private const string MachineGuidPath = @"SOFTWARE\Microsoft\Cryptography";

    // 中心只接收不可逆哈希；原始 MachineGuid 不离开本机。
    public static string GetMachineId()
    {
        if (!OperatingSystem.IsWindows()) throw new PlatformNotSupportedException("采集调试代理仅支持 Windows 机器身份。");
        using var baseKey = RegistryKey.OpenBaseKey(RegistryHive.LocalMachine, RegistryView.Registry64);
        using var key = baseKey.OpenSubKey(MachineGuidPath, writable: false);
        var machineGuid = key?.GetValue("MachineGuid") as string;
        if (string.IsNullOrWhiteSpace(machineGuid)) throw new InvalidOperationException("无法读取 Windows MachineGuid，不能完成代理注册。");
        return HashMachineGuid(machineGuid);
    }

    internal static string HashMachineGuid(string machineGuid)
    {
        if (string.IsNullOrWhiteSpace(machineGuid)) throw new ArgumentException("MachineGuid 不能为空。", nameof(machineGuid));
        var source = Encoding.UTF8.GetBytes($"InduForge.Collector.DevAgent|{machineGuid.Trim().ToLowerInvariant()}");
        return Convert.ToHexStringLower(SHA256.HashData(source));
    }
}
