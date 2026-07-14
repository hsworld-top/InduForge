using System.Drawing;
using System.Runtime.InteropServices;

namespace InduForge.Collector.DevAgent;

internal sealed class TrayApplicationContext : ApplicationContext
{
    private static readonly AgentProtocolCapability[] Capabilities = [new("opcua", "1.0", ["connection.test", "opcua.browse", "opcua.read"])];
    private readonly AgentLocalSettings _settings = AgentLocalSettings.Default;
    private readonly AgentCredentialStore _credentialStore = new();
    private readonly AgentStatusForm _statusForm;
    private readonly NotifyIcon _notifyIcon;
    private readonly ContextMenuStrip _trayMenu;
    private readonly TrayMenuHostForm _trayMenuHost = new();
    private readonly ToolStripMenuItem _connectionMenuItem;
    private readonly ToolStripMenuItem _clearRegistrationMenuItem;
    private readonly SingleInstanceCoordinator _singleInstance;
    private readonly OpcUaSelfTestService _selfTestService = new();
    private readonly CenterApiClient _apiClient = new(new HttpClient { Timeout = TimeSpan.FromSeconds(35) });
    private readonly AgentWorker _worker;
    private readonly CancellationTokenSource _applicationCancellation = new();
    private AgentCredentials? _credentials;
    private AgentStatusSnapshot _snapshot;
    private bool _isExiting;

    public TrayApplicationContext(SingleInstanceCoordinator singleInstance)
    {
        _singleInstance = singleInstance;
        _credentials = LoadCredentialsSafely();
        _snapshot = new AgentStatusSnapshot(
            _credentials is null ? AgentConnectionState.NotRegistered : AgentConnectionState.Disconnected,
            _credentials?.CenterUrl ?? _settings.CenterUrl,
            _credentials?.AgentName ?? _settings.AgentName,
            _credentials?.AgentId ?? string.Empty,
            "空闲",
            _credentials is null ? "请输入中心地址和一次性注册码" : "正在连接中心",
            null,
            DateTimeOffset.Now);
        _statusForm = new AgentStatusForm(_settings);
        _statusForm.UpdateStatus(_snapshot);
        _statusForm.RegisterRequested += OnRegisterRequested;
        _statusForm.ConnectionToggleRequested += OnConnectionToggleRequested;
        _statusForm.ClearRegistrationRequested += OnClearRegistrationRequested;
        _statusForm.SelfTestRequested += OnSelfTestRequested;
        _worker = new AgentWorker(_apiClient, new CollectorTaskExecutor(), OnWorkerUpdateAsync);

        _trayMenu = new ContextMenuStrip();
        _trayMenu.Items.Add("打开状态", null, (_, _) => ShowStatus());
        _connectionMenuItem = new ToolStripMenuItem("重新连接", null, (_, _) => OnConnectionToggleRequested(this, EventArgs.Empty));
        _clearRegistrationMenuItem = new ToolStripMenuItem("清除注册", null, (_, _) => OnClearRegistrationRequested(this, EventArgs.Empty));
        _trayMenu.Items.Add(_connectionMenuItem);
        _trayMenu.Items.Add(_clearRegistrationMenuItem);
        _trayMenu.Items.Add(new ToolStripSeparator());
        _trayMenu.Items.Add("退出", null, (_, _) => ExitAgent());
        _trayMenu.Closed += (_, _) => _trayMenuHost.Hide();
        _notifyIcon = new NotifyIcon { Icon = SystemIcons.Application, Text = "InduForge 采集调试代理：未注册", Visible = true };
        _notifyIcon.MouseClick += (_, args) =>
        {
            if (args.Button != MouseButtons.Right) return;
            ShowTrayMenu();
        };
        _notifyIcon.MouseDoubleClick += (_, args) => { if (args.Button == MouseButtons.Left) ShowStatus(); };
        ShowStatus();
        _singleInstance.StartListening(OnActivationRequestedAsync);
        if (_credentials is not null) _worker.Start(_credentials, _applicationCancellation.Token);
    }

    protected override void ExitThreadCore()
    {
        _applicationCancellation.Cancel();
        _worker.DisposeAsync().AsTask().GetAwaiter().GetResult();
        _singleInstance.DisposeAsync().AsTask().GetAwaiter().GetResult();
        _notifyIcon.Visible = false;
        _notifyIcon.Dispose();
        _trayMenu.Dispose();
        _trayMenuHost.Dispose();
        _statusForm.Dispose();
        _applicationCancellation.Dispose();
        base.ExitThreadCore();
    }

    private async void OnRegisterRequested(object? sender, AgentRegistrationFormValue value)
    {
        if (string.IsNullOrWhiteSpace(value.CenterUrl) || string.IsNullOrWhiteSpace(value.AgentName) || string.IsNullOrWhiteSpace(value.RegistrationCode))
        {
            MessageBox.Show("中心地址、代理名称和注册码不能为空。", "注册失败", MessageBoxButtons.OK, MessageBoxIcon.Warning);
            return;
        }
        _statusForm.SetRegistrationRunning(true);
        try
        {
            var registration = await _apiClient.RegisterAsync(value.CenterUrl, new AgentRegistrationRequest(value.RegistrationCode, value.AgentName, "windows", RuntimeInformation.ProcessArchitecture.ToString().ToLowerInvariant(), Application.ProductVersion, Capabilities), _applicationCancellation.Token);
            _credentials = new AgentCredentials(value.CenterUrl, registration.AgentId, registration.AgentToken, registration.TenantId, value.AgentName);
            _credentialStore.Save(_credentials);
            UpdateSnapshot(_snapshot with { ConnectionState = AgentConnectionState.Disconnected, CenterUrl = value.CenterUrl, AgentName = value.AgentName, AgentId = registration.AgentId, LastMessage = "注册成功，正在连接中心", UpdatedAt = DateTimeOffset.Now });
            _worker.Start(_credentials, _applicationCancellation.Token);
        }
        catch (Exception exception)
        {
            MessageBox.Show(exception.Message, "注册失败", MessageBoxButtons.OK, MessageBoxIcon.Error);
            UpdateSnapshot(_snapshot with { LastMessage = exception.Message, UpdatedAt = DateTimeOffset.Now });
        }
        finally { _statusForm.SetRegistrationRunning(false); }
    }

    private async void OnConnectionToggleRequested(object? sender, EventArgs args)
    {
        if (_credentials is null) return;
        if (_worker.IsRunning)
        {
            await _worker.StopAsync();
            UpdateSnapshot(_snapshot with { ConnectionState = AgentConnectionState.Disconnected, LastMessage = "已手动断开中心；本地协议自检仍可使用", UpdatedAt = DateTimeOffset.Now });
        }
        else
        {
            UpdateSnapshot(_snapshot with { ConnectionState = AgentConnectionState.Disconnected, LastMessage = "正在重新连接中心", UpdatedAt = DateTimeOffset.Now });
            _worker.Start(_credentials, _applicationCancellation.Token);
        }
    }

    private async void OnClearRegistrationRequested(object? sender, EventArgs args)
    {
        if (MessageBox.Show("清除后需要新的注册码才能再次连接中心，是否继续？", "清除注册", MessageBoxButtons.YesNo, MessageBoxIcon.Warning) != DialogResult.Yes) return;
        await _worker.StopAsync();
        ClearCredentials();
        UpdateSnapshot(new AgentStatusSnapshot(AgentConnectionState.NotRegistered, _settings.CenterUrl, _settings.AgentName, string.Empty, "空闲", "本地注册已清除", null, DateTimeOffset.Now));
    }

    private Task OnWorkerUpdateAsync(AgentWorkerUpdate update)
    {
        if (_statusForm.IsDisposed) return Task.CompletedTask;
        _statusForm.BeginInvoke(() =>
        {
            if (update.CredentialsInvalid) ClearCredentials();
            UpdateSnapshot(_snapshot with { ConnectionState = update.State, AgentId = _credentials?.AgentId ?? string.Empty, CurrentTask = update.CurrentTask, LastMessage = update.Message, LastHeartbeatAt = update.State == AgentConnectionState.Connected ? update.UpdatedAt : _snapshot.LastHeartbeatAt, UpdatedAt = update.UpdatedAt });
        });
        return Task.CompletedTask;
    }

    private AgentCredentials? LoadCredentialsSafely()
    {
        try { return _credentialStore.Load(); }
        catch { _credentialStore.Clear(); return null; }
    }

    private void ClearCredentials() { _credentialStore.Clear(); _credentials = null; }
    private Task OnActivationRequestedAsync()
    {
        if (_isExiting || _statusForm.IsDisposed) return Task.CompletedTask;
        try
        {
            if (_statusForm.InvokeRequired) _statusForm.BeginInvoke(ShowStatus);
            else ShowStatus();
        }
        catch (InvalidOperationException) when (_isExiting || _statusForm.IsDisposed)
        {
        }
        return Task.CompletedTask;
    }

    private void ShowStatus()
    {
        if (_statusForm.IsDisposed) return;
        if (!_statusForm.Visible) _statusForm.Show();
        if (_statusForm.WindowState == FormWindowState.Minimized) _statusForm.WindowState = FormWindowState.Normal;
        _statusForm.BringToFront();
        _statusForm.Activate();
        SetForegroundWindow(_statusForm.Handle);
    }

    // 托盘菜单必须绑定前台窗口显示，否则窗口可见时 Windows 可能立即收回菜单焦点。
    private void ShowTrayMenu()
    {
        if (_trayMenu.Visible || _isExiting) return;
        RefreshMenus();
        var cursorPosition = Cursor.Position;
        _trayMenuHost.ActivateAt(cursorPosition);
        SetForegroundWindow(_trayMenuHost.Handle);
        _trayMenu.Show(_trayMenuHost, Point.Empty);
    }
    private async void OnSelfTestRequested(object? sender, string endpointUrl) { _statusForm.SetSelfTestRunning(true); UpdateSnapshot(_snapshot with { CurrentTask = "OPC UA 本地自检", LastMessage = $"正在连接 {endpointUrl}", UpdatedAt = DateTimeOffset.Now }); try { var message = await _selfTestService.RunAsync(endpointUrl, _applicationCancellation.Token); UpdateSnapshot(_snapshot with { CurrentTask = "空闲", LastMessage = message, UpdatedAt = DateTimeOffset.Now }); } catch (OperationCanceledException) when (_applicationCancellation.IsCancellationRequested) { } catch (Exception exception) { UpdateSnapshot(_snapshot with { CurrentTask = "空闲", LastMessage = exception.Message, UpdatedAt = DateTimeOffset.Now }); } finally { _statusForm.SetSelfTestRunning(false); } }
    private void UpdateSnapshot(AgentStatusSnapshot snapshot) { _snapshot = snapshot; _statusForm.UpdateStatus(snapshot); _notifyIcon.Text = snapshot.ConnectionState switch { AgentConnectionState.Connected => "InduForge 采集调试代理：已连接", AgentConnectionState.Disconnected => "InduForge 采集调试代理：已断开", _ => "InduForge 采集调试代理：未注册" }; }
    private void RefreshMenus() { var registered = _credentials is not null; _connectionMenuItem.Visible = registered; _connectionMenuItem.Text = _worker?.IsRunning == true ? "断开中心" : "重新连接"; _clearRegistrationMenuItem.Visible = registered; }
    private void ExitAgent() { if (_isExiting) return; _isExiting = true; _applicationCancellation.Cancel(); _statusForm.ExitApplication(); ExitThread(); }

    [DllImport("user32.dll")]
    [return: MarshalAs(UnmanagedType.Bool)]
    private static extern bool SetForegroundWindow(IntPtr windowHandle);
}
