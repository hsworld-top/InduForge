using System.Drawing;

namespace InduForge.Collector.DevAgent;

internal sealed class TrayApplicationContext : ApplicationContext
{
    private readonly AgentLocalSettings _settings = AgentLocalSettings.Default;
    private readonly AgentStatusForm _statusForm;
    private readonly NotifyIcon _notifyIcon;
    private readonly OpcUaSelfTestService _selfTestService = new();
    private readonly CancellationTokenSource _applicationCancellation = new();
    private AgentStatusSnapshot _snapshot;
    private bool _isExiting;

    public TrayApplicationContext()
    {
        _snapshot = new AgentStatusSnapshot(
            AgentConnectionState.NotRegistered,
            _settings.CenterUrl,
            _settings.AgentName,
            "空闲",
            "等待中心注册功能接入",
            DateTimeOffset.Now);
        _statusForm = new AgentStatusForm(_settings);
        _statusForm.UpdateStatus(_snapshot);
        _statusForm.SelfTestRequested += OnSelfTestRequested;

        var menu = new ContextMenuStrip();
        menu.Items.Add("打开状态", null, (_, _) => ShowStatus());
        menu.Items.Add("断开中心", null, (_, _) => SetDisconnected());
        menu.Items.Add(new ToolStripSeparator());
        menu.Items.Add("退出", null, (_, _) => ExitAgent());
        _notifyIcon = new NotifyIcon
        {
            Icon = SystemIcons.Application,
            Text = "InduForge 采集调试代理：未注册",
            Visible = true,
            ContextMenuStrip = menu,
        };
        _notifyIcon.DoubleClick += (_, _) => ShowStatus();
        ShowStatus();
    }

    protected override void ExitThreadCore()
    {
        _applicationCancellation.Cancel();
        _notifyIcon.Visible = false;
        _notifyIcon.Dispose();
        _statusForm.Dispose();
        _applicationCancellation.Dispose();
        base.ExitThreadCore();
    }

    private void ShowStatus()
    {
        if (_statusForm.Visible)
        {
            _statusForm.Activate();
            return;
        }

        _statusForm.Show();
        _statusForm.WindowState = FormWindowState.Normal;
        _statusForm.Activate();
    }

    private void SetDisconnected()
    {
        UpdateSnapshot(_snapshot with
        {
            ConnectionState = AgentConnectionState.Disconnected,
            LastMessage = "已手动断开中心；本地协议自检仍可使用",
            UpdatedAt = DateTimeOffset.Now,
        });
    }

    private async void OnSelfTestRequested(object? sender, string endpointUrl)
    {
        _statusForm.SetSelfTestRunning(true);
        UpdateSnapshot(_snapshot with
        {
            CurrentTask = "OPC UA 本地自检",
            LastMessage = $"正在连接 {endpointUrl}",
            UpdatedAt = DateTimeOffset.Now,
        });

        try
        {
            var message = await _selfTestService.RunAsync(endpointUrl, _applicationCancellation.Token);
            UpdateSnapshot(_snapshot with
            {
                CurrentTask = "空闲",
                LastMessage = message,
                UpdatedAt = DateTimeOffset.Now,
            });
        }
        catch (OperationCanceledException) when (_applicationCancellation.IsCancellationRequested)
        {
        }
        catch (Exception exception)
        {
            UpdateSnapshot(_snapshot with
            {
                CurrentTask = "空闲",
                LastMessage = exception.Message,
                UpdatedAt = DateTimeOffset.Now,
            });
        }
        finally
        {
            _statusForm.SetSelfTestRunning(false);
        }
    }

    private void UpdateSnapshot(AgentStatusSnapshot snapshot)
    {
        _snapshot = snapshot;
        _statusForm.UpdateStatus(snapshot);
        _notifyIcon.Text = snapshot.ConnectionState switch
        {
            AgentConnectionState.Connected => "InduForge 采集调试代理：已连接",
            AgentConnectionState.Disconnected => "InduForge 采集调试代理：已断开",
            _ => "InduForge 采集调试代理：未注册",
        };
    }

    private void ExitAgent()
    {
        if (_isExiting)
        {
            return;
        }

        _isExiting = true;
        _applicationCancellation.Cancel();
        _statusForm.ExitApplication();
        ExitThread();
    }
}
