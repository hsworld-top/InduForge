using System.Diagnostics;
using System.IO.Pipes;
using System.Runtime.InteropServices;
using System.Security.AccessControl;
using System.Security.Principal;
using System.Security.Cryptography;
using System.Text;

namespace InduForge.Collector.DevAgent;

internal sealed class SingleInstanceCoordinator : IAsyncDisposable
{
    private const string ActivationCommand = "show";
    private readonly Mutex _instanceMutex;
    private readonly string _pipeName;
    private readonly CancellationTokenSource _listenCancellation = new();
    private Task? _listenTask;
    private int _disposed;

    public SingleInstanceCoordinator(string? instanceName = null)
    {
        var identity = instanceName ?? "InduForge.Collector.DevAgent";
        var suffix = Convert.ToHexString(SHA256.HashData(Encoding.UTF8.GetBytes(identity)))[..24];
        _pipeName = $"InduForge.Collector.DevAgent.{suffix}";
        var security = new MutexSecurity();
        security.AddAccessRule(new MutexAccessRule(
            new SecurityIdentifier(WellKnownSidType.WorldSid, null),
            MutexRights.FullControl,
            AccessControlType.Allow));
        _instanceMutex = MutexAcl.Create(false, $"Global\\{_pipeName}", out var createdNew, security);
        IsPrimaryInstance = createdNew;
        ExistingInstanceInCurrentSession = !createdNew && HasMatchingProcessInCurrentSession();
    }

    public bool IsPrimaryInstance { get; }
    public bool ExistingInstanceInCurrentSession { get; }

    // 首实例监听后续进程的唤醒请求；回调由调用方负责切换到 UI 线程。
    public void StartListening(Func<Task> activationHandler)
    {
        ArgumentNullException.ThrowIfNull(activationHandler);
        ObjectDisposedException.ThrowIf(Volatile.Read(ref _disposed) != 0, this);
        if (!IsPrimaryInstance) throw new InvalidOperationException("只有首实例可以监听唤醒请求。");
        if (_listenTask is not null) throw new InvalidOperationException("单实例监听已经启动。");
        _listenTask = ListenAsync(activationHandler, _listenCancellation.Token);
    }

    // 后续实例只通知已有进程显示窗口，通知失败时也不能启动第二套 Agent。
    public async Task<bool> NotifyExistingInstanceAsync(CancellationToken cancellationToken = default)
    {
        ObjectDisposedException.ThrowIf(Volatile.Read(ref _disposed) != 0, this);
        if (IsPrimaryInstance) return false;

        try
        {
            GrantForegroundPermissionToExistingProcess();
            await using var client = new NamedPipeClientStream(".", _pipeName, PipeDirection.Out, PipeOptions.Asynchronous);
            await client.ConnectAsync(2000, cancellationToken).ConfigureAwait(false);
            await using var writer = new StreamWriter(client, Encoding.UTF8, leaveOpen: true) { AutoFlush = true };
            await writer.WriteLineAsync(ActivationCommand.AsMemory(), cancellationToken).ConfigureAwait(false);
            return true;
        }
        catch (OperationCanceledException) when (!cancellationToken.IsCancellationRequested)
        {
            return false;
        }
        catch (IOException)
        {
            return false;
        }
        catch (TimeoutException)
        {
            return false;
        }
    }

    public async ValueTask DisposeAsync()
    {
        if (Interlocked.Exchange(ref _disposed, 1) != 0) return;
        _listenCancellation.Cancel();
        if (_listenTask is not null)
        {
            try
            {
                await _listenTask.ConfigureAwait(false);
            }
            catch (OperationCanceledException)
            {
            }
        }
        _listenCancellation.Dispose();
        _instanceMutex.Dispose();
    }

    private async Task ListenAsync(Func<Task> activationHandler, CancellationToken cancellationToken)
    {
        while (!cancellationToken.IsCancellationRequested)
        {
            try
            {
                await using var server = new NamedPipeServerStream(
                    _pipeName,
                    PipeDirection.In,
                    1,
                    PipeTransmissionMode.Byte,
                    PipeOptions.Asynchronous);
                await server.WaitForConnectionAsync(cancellationToken).ConfigureAwait(false);
                using var reader = new StreamReader(server, Encoding.UTF8, leaveOpen: true);
                var command = await reader.ReadLineAsync(cancellationToken).ConfigureAwait(false);
                if (string.Equals(command, ActivationCommand, StringComparison.Ordinal))
                {
                    await activationHandler().ConfigureAwait(false);
                }
            }
            catch (OperationCanceledException) when (cancellationToken.IsCancellationRequested)
            {
                break;
            }
            catch (IOException)
            {
            }
            catch (UnauthorizedAccessException)
            {
                await Task.Delay(100, cancellationToken).ConfigureAwait(false);
            }
        }
    }

    private static bool HasMatchingProcessInCurrentSession()
    {
        if (string.IsNullOrWhiteSpace(Environment.ProcessPath)) return false;
        using var currentProcess = Process.GetCurrentProcess();
        var processName = Path.GetFileNameWithoutExtension(Environment.ProcessPath);
        foreach (var process in Process.GetProcessesByName(processName))
        {
            using (process)
            {
                if (process.Id == currentProcess.Id || process.SessionId != currentProcess.SessionId) continue;
                try
                {
                    if (string.Equals(process.MainModule?.FileName, Environment.ProcessPath, StringComparison.OrdinalIgnoreCase)) return true;
                }
                catch (InvalidOperationException)
                {
                }
                catch (System.ComponentModel.Win32Exception)
                {
                }
            }
        }
        return false;
    }

    private static void GrantForegroundPermissionToExistingProcess()
    {
        if (!OperatingSystem.IsWindows() || string.IsNullOrWhiteSpace(Environment.ProcessPath)) return;
        var currentProcess = Process.GetCurrentProcess();
        var processName = Path.GetFileNameWithoutExtension(Environment.ProcessPath);
        foreach (var process in Process.GetProcessesByName(processName))
        {
            using (process)
            {
                if (process.Id == currentProcess.Id) continue;
                try
                {
                    if (!string.Equals(process.MainModule?.FileName, Environment.ProcessPath, StringComparison.OrdinalIgnoreCase)) continue;
                    AllowSetForegroundWindow((uint)process.Id);
                    return;
                }
                catch (InvalidOperationException)
                {
                }
                catch (System.ComponentModel.Win32Exception)
                {
                }
            }
        }
    }

    // 用户主动启动第二个 EXE 时，由该进程把前台激活权限转交给已有实例。
    [DllImport("user32.dll")]
    [return: MarshalAs(UnmanagedType.Bool)]
    private static extern bool AllowSetForegroundWindow(uint processId);
}
