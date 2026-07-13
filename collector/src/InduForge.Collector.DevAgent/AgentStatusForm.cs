using System.Drawing;
using System.Globalization;

namespace InduForge.Collector.DevAgent;

internal sealed class AgentStatusForm : Form
{
    private readonly Label _stateValue = CreateValueLabel();
    private readonly Label _centerValue = CreateValueLabel();
    private readonly Label _nameValue = CreateValueLabel();
    private readonly Label _taskValue = CreateValueLabel();
    private readonly Label _messageValue = CreateValueLabel();
    private readonly Label _updatedValue = CreateValueLabel();
    private readonly TextBox _opcUaEndpoint = new() { Dock = DockStyle.Fill };
    private readonly Button _selfTestButton = new() { Text = "OPC UA 本地自检", AutoSize = true };
    private bool _allowClose;

    public AgentStatusForm(AgentLocalSettings settings)
    {
        Text = "InduForge 采集调试代理";
        StartPosition = FormStartPosition.CenterScreen;
        MinimumSize = new Size(680, 440);
        Size = new Size(760, 500);
        Font = new Font("Microsoft YaHei UI", 9F);
        Icon = SystemIcons.Application;

        var content = new Panel { Dock = DockStyle.Fill, Padding = new Padding(24) };
        var title = new Label
        {
            Text = "InduForge 采集调试代理",
            Dock = DockStyle.Top,
            Font = new Font(Font, FontStyle.Bold),
            AutoSize = true,
            Padding = new Padding(0, 0, 0, 12),
        };
        var description = new Label
        {
            Text = "开发态短链调试组件。关闭窗口后继续驻留系统托盘，只有托盘“退出”才会停止程序。",
            Dock = DockStyle.Top,
            AutoSize = true,
            ForeColor = Color.DimGray,
            Padding = new Padding(0, 0, 0, 16),
        };
        var statusTable = new TableLayoutPanel
        {
            Dock = DockStyle.Top,
            AutoSize = true,
            ColumnCount = 2,
            Padding = new Padding(0, 0, 0, 16),
        };
        statusTable.ColumnStyles.Add(new ColumnStyle(SizeType.Absolute, 110));
        statusTable.ColumnStyles.Add(new ColumnStyle(SizeType.Percent, 100));
        AddRow(statusTable, "状态", _stateValue);
        AddRow(statusTable, "中心地址", _centerValue);
        AddRow(statusTable, "代理名称", _nameValue);
        AddRow(statusTable, "当前任务", _taskValue);
        AddRow(statusTable, "最近消息", _messageValue);
        AddRow(statusTable, "更新时间", _updatedValue);

        var testGroup = new GroupBox
        {
            Text = "本地协议自检",
            Dock = DockStyle.Top,
            AutoSize = true,
            Padding = new Padding(12),
        };
        var testLayout = new TableLayoutPanel { Dock = DockStyle.Fill, AutoSize = true, ColumnCount = 2 };
        testLayout.ColumnStyles.Add(new ColumnStyle(SizeType.Percent, 100));
        testLayout.ColumnStyles.Add(new ColumnStyle(SizeType.AutoSize));
        _opcUaEndpoint.Text = settings.OpcUaEndpoint;
        _selfTestButton.Click += (_, _) => SelfTestRequested?.Invoke(this, OpcUaEndpoint);
        testLayout.Controls.Add(_opcUaEndpoint, 0, 0);
        testLayout.Controls.Add(_selfTestButton, 1, 0);
        testGroup.Controls.Add(testLayout);

        var footer = new Label
        {
            Text = "当前版本尚未接入中心注册；本地自检用于验证 Windows 进程与 OPC UA 驱动。",
            Dock = DockStyle.Bottom,
            AutoSize = true,
            ForeColor = Color.DarkOrange,
            Padding = new Padding(0, 16, 0, 0),
        };
        content.Controls.Add(footer);
        content.Controls.Add(testGroup);
        content.Controls.Add(statusTable);
        content.Controls.Add(description);
        content.Controls.Add(title);
        Controls.Add(content);
        FormClosing += OnFormClosing;
    }

    public event EventHandler<string>? SelfTestRequested;

    public string OpcUaEndpoint => _opcUaEndpoint.Text.Trim();

    public void UpdateStatus(AgentStatusSnapshot snapshot)
    {
        _stateValue.Text = snapshot.ConnectionState switch
        {
            AgentConnectionState.Connected => "已连接",
            AgentConnectionState.Disconnected => "已断开",
            _ => "未注册",
        };
        _stateValue.ForeColor = snapshot.ConnectionState == AgentConnectionState.Connected ? Color.ForestGreen : Color.DarkOrange;
        _centerValue.Text = snapshot.CenterUrl;
        _nameValue.Text = snapshot.AgentName;
        _taskValue.Text = snapshot.CurrentTask;
        _messageValue.Text = snapshot.LastMessage;
        _updatedValue.Text = snapshot.UpdatedAt.LocalDateTime.ToString("yyyy-MM-dd HH:mm:ss", CultureInfo.GetCultureInfo("zh-CN"));
    }

    public void SetSelfTestRunning(bool running)
    {
        _selfTestButton.Enabled = !running;
        _selfTestButton.Text = running ? "正在测试..." : "OPC UA 本地自检";
    }

    public void ExitApplication()
    {
        _allowClose = true;
        Close();
    }

    private void OnFormClosing(object? sender, FormClosingEventArgs eventArgs)
    {
        if (_allowClose || eventArgs.CloseReason == CloseReason.WindowsShutDown)
        {
            return;
        }

        eventArgs.Cancel = true;
        Hide();
    }

    private static Label CreateValueLabel() => new()
    {
        AutoSize = true,
        Dock = DockStyle.Fill,
        Padding = new Padding(0, 5, 0, 5),
    };

    private static void AddRow(TableLayoutPanel table, string name, Control value)
    {
        var row = table.RowCount++;
        table.RowStyles.Add(new RowStyle(SizeType.AutoSize));
        table.Controls.Add(new Label
        {
            Text = name,
            AutoSize = true,
            Dock = DockStyle.Fill,
            ForeColor = Color.DimGray,
            Padding = new Padding(0, 5, 0, 5),
        }, 0, row);
        table.Controls.Add(value, 1, row);
    }
}
