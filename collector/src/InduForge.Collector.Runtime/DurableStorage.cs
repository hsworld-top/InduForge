using System.Runtime.InteropServices;

namespace InduForge.Collector.Runtime;

/// <summary>
/// 原子改名只保证名称切换；这里再持久化父目录，保证掉电后目录项本身也可恢复。
/// 当前仅支持本机 Linux/macOS 的目录 fsync；Windows 及其他平台拒绝伪造耐久保证。
/// 远程或 FUSE 文件系统是否兑现 fsync 仍须由部署验收确认，本阶段不维护文件系统白名单。
/// </summary>
internal static class DurableStorage
{
    public static void FlushParentDirectory(string directoryPath)
    {
        EnsurePlatformSupported(OperatingSystem.IsWindows(), OperatingSystem.IsLinux() || OperatingSystem.IsMacOS());
        if (OperatingSystem.IsLinux() || OperatingSystem.IsMacOS())
        {
            FlushUnixDirectory(directoryPath);
            return;
        }
    }

    /// <summary>Windows 尚无经过验证的目录项刷盘实现，因此必须拒绝启动而不是伪造保证。</summary>
    internal static void EnsurePlatformSupported(bool isWindows, bool isUnix)
    {
        if (isWindows) throw new PlatformNotSupportedException("Windows 尚未实现可验证的 WAL 目录持久化语义");
        if (!isUnix) throw new PlatformNotSupportedException("当前平台没有已验证的 WAL 父目录持久化实现");
    }

    /// <summary>新建嵌套目录后，从最内层向既存祖先逐级刷父目录，持久化每个目录项。</summary>
    public static void CreateDirectoryDurably(string directoryPath, Action<string>? flushedParentObserver = null)
        => CreateDirectoryDurably(directoryPath, OperatingSystem.IsWindows(), OperatingSystem.IsLinux() || OperatingSystem.IsMacOS(), flushedParentObserver);

    /// <summary>平台判断独立于宿主，供测试验证既存目录也不能绕过 fail-closed。</summary>
    internal static void CreateDirectoryDurably(string directoryPath, bool isWindows, bool isUnix, Action<string>? flushedParentObserver = null)
    {
        EnsurePlatformSupported(isWindows, isUnix);
        var fullPath = Path.GetFullPath(directoryPath);
        var created = new List<string>();
        var current = fullPath;
        while (!Directory.Exists(current))
        {
            created.Add(current);
            current = Directory.GetParent(current)?.FullName
                ?? throw new IOException("无法找到 WAL 目录的既存祖先");
        }

        if (created.Count == 0) return;
        Directory.CreateDirectory(fullPath);
        foreach (var directory in created)
        {
            var parent = Directory.GetParent(directory)?.FullName
                ?? throw new IOException("无法找到新建 WAL 目录的父目录");
            FlushUnixDirectory(parent);
            flushedParentObserver?.Invoke(parent);
        }
    }

    private static void FlushUnixDirectory(string directoryPath)
    {
        var descriptor = open(directoryPath, 0);
        if (descriptor < 0) throw new IOException("无法打开 WAL 父目录，错误码: " + Marshal.GetLastWin32Error());
        try
        {
            if (fsync(descriptor) != 0) throw new IOException("无法持久化 WAL 父目录，错误码: " + Marshal.GetLastWin32Error());
        }
        finally
        {
            _ = close(descriptor);
        }
    }

#pragma warning disable CA2101 // LPUTF8Str 显式指定 Unix 路径封送，规则无法识别该组合。
    [DllImport("libc", CharSet = CharSet.Ansi, ExactSpelling = true, SetLastError = true)]
    private static extern int open([MarshalAs(UnmanagedType.LPUTF8Str)] string pathname, int flags);
#pragma warning restore CA2101

    [DllImport("libc", SetLastError = true)]
    private static extern int fsync(int fd);

    [DllImport("libc", SetLastError = true)]
    private static extern int close(int fd);

}
