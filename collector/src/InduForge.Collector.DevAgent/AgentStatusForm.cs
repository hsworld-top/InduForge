using System.Drawing;
using System.Globalization;

namespace InduForge.Collector.DevAgent;

internal sealed class AgentStatusForm : Form
{
    private readonly Label _stateValue = CreateValueLabel();
    private readonly Label _centerValue = CreateValueLabel();
    private readonly Label _nameValue = CreateValueLabel();
    private readonly Label _agentIdValue = CreateValueLabel();
    private readonly Label _taskValue = CreateValueLabel();
    private readonly Label _messageValue = CreateValueLabel();
    private readonly Label _heartbeatValue = CreateValueLabel();
    private readonly TextBox _centerInput = new() { Dock = DockStyle.Fill };
    private readonly TextBox _nameInput = new() { Dock = DockStyle.Fill };
    private readonly TextBox _registrationCodeInput = new() { Dock = DockStyle.Fill };
    private readonly Button _registerButton = new() { Text = "注册并连接", AutoSize = true };
    private readonly Button _connectionButton = new() { Text = "断开中心", AutoSize = true };
    private readonly Button _clearButton = new() { Text = "清除注册", AutoSize = true };
    private readonly TextBox _opcUaEndpoint = new() { Dock = DockStyle.Fill };
    private readonly Button _selfTestButton = new() { Text = "OPC UA 本地自检", AutoSize = true };
    private readonly GroupBox _registrationGroup;
    private bool _allowClose;

    public AgentStatusForm(AgentLocalSettings settings)
    {
        Text = "InduForge 采集调试代理";
        StartPosition = FormStartPosition.CenterScreen;
        MinimumSize = new Size(720, 620);
        Size = new Size(820, 680);
        Font = new Font("Microsoft YaHei UI", 9F);
        Icon = SystemIcons.Application;

        var content = new Panel { Dock = DockStyle.Fill, Padding = new Padding(24), AutoScroll = true };
        var title = new Label { Text = "InduForge 采集调试代理", Dock = DockStyle.Top, Font = new Font(Font, FontStyle.Bold), AutoSize = true, Padding = new Padding(0, 0, 0, 12) };
        var description = new Label { Text = "开发态短链调试组件。关闭窗口后继续驻留系统托盘，只有托盘“退出”才会停止程序。", Dock = DockStyle.Top, AutoSize = true, ForeColor = Color.DimGray, Padding = new Padding(0, 0, 0, 16) };
        var statusTable = new TableLayoutPanel { Dock = DockStyle.Top, AutoSize = true, ColumnCount = 2, Padding = new Padding(0, 0, 0, 16) };
        statusTable.ColumnStyles.Add(new ColumnStyle(SizeType.Absolute, 110));
        statusTable.ColumnStyles.Add(new ColumnStyle(SizeType.Percent, 100));
        AddRow(statusTable, "状态", _stateValue);
        AddRow(statusTable, "中心地址", _centerValue);
        AddRow(statusTable, "代理名称", _nameValue);
        AddRow(statusTable, "Agent ID", _agentIdValue);
        AddRow(statusTable, "当前任务", _taskValue);
        AddRow(statusTable, "最近消息", _messageValue);
        AddRow(statusTable, "最后心跳", _heartbeatValue);

        _registrationGroup = new GroupBox { Text = "中心注册", Dock = DockStyle.Top, AutoSize = true, Padding = new Padding(12) };
        var registrationLayout = new TableLayoutPanel { Dock = DockStyle.Fill, AutoSize = true, ColumnCount = 3 };
        registrationLayout.ColumnStyles.Add(new ColumnStyle(SizeType.Absolute, 90));
        registrationLayout.ColumnStyles.Add(new ColumnStyle(SizeType.Percent, 100));
        registrationLayout.ColumnStyles.Add(new ColumnStyle(SizeType.AutoSize));
        _centerInput.Text = settings.CenterUrl;
        _nameInput.Text = settings.AgentName;
        AddInputRow(registrationLayout, 0, "中心地址", _centerInput);
        AddInputRow(registrationLayout, 1, "代理名称", _nameInput);
        AddInputRow(registrationLayout, 2, "注册码", _registrationCodeInput);
        var registrationButtons = new FlowLayoutPanel { Dock = DockStyle.Fill, AutoSize = true, FlowDirection = FlowDirection.LeftToRight };
        registrationButtons.Controls.Add(_registerButton);
        registrationButtons.Controls.Add(_connectionButton);
        registrationButtons.Controls.Add(_clearButton);
        registrationLayout.Controls.Add(registrationButtons, 1, 3);
        registrationLayout.SetColumnSpan(registrationButtons, 2);
        _registrationGroup.Controls.Add(registrationLayout);

        var testGroup = new GroupBox { Text = "本地协议自检", Dock = DockStyle.Top, AutoSize = true, Padding = new Padding(12) };
        var testLayout = new TableLayoutPanel { Dock = DockStyle.Fill, AutoSize = true, ColumnCount = 2 };
        testLayout.ColumnStyles.Add(new ColumnStyle(SizeType.Percent, 100));
        testLayout.ColumnStyles.Add(new ColumnStyle(SizeType.AutoSize));
        _opcUaEndpoint.Text = settings.OpcUaEndpoint;
        testLayout.Controls.Add(_opcUaEndpoint, 0, 0);
        testLayout.Controls.Add(_selfTestButton, 1, 0);
        testGroup.Controls.Add(testLayout);

        _registerButton.Click += (_, _) => RegisterRequested?.Invoke(this, new AgentRegistrationFormValue(_centerInput.Text.Trim(), _nameInput.Text.Trim(), _registrationCodeInput.Text.Trim()));
        _connectionButton.Click += (_, _) => ConnectionToggleRequested?.Invoke(this, EventArgs.Empty);
        _clearButton.Click += (_, _) => ClearRegistrationRequested?.Invoke(this, EventArgs.Empty);
        _selfTestButton.Click += (_, _) => SelfTestRequested?.Invoke(this, OpcUaEndpoint);

        content.Controls.Add(testGroup);
        content.Controls.Add(_registrationGroup);
        content.Controls.Add(statusTable);
        content.Controls.Add(description);
        content.Controls.Add(title);
        Controls.Add(content);
        FormClosing += OnFormClosing;
    }

    public event EventHandler<AgentRegistrationFormValue>? RegisterRequested;
    public event EventHandler? ConnectionToggleRequested;
    public event EventHandler? ClearRegistrationRequested;
    public event EventHandler<string>? SelfTestRequested;
    public string OpcUaEndpoint => _opcUaEndpoint.Text.Trim();

    public void UpdateStatus(AgentStatusSnapshot snapshot)
    {
        _stateValue.Text = snapshot.ConnectionState switch { AgentConnectionState.Connected => "已连接", AgentConnectionState.Disconnected => "已断开", _ => "未注册" };
        _stateValue.ForeColor = snapshot.ConnectionState == AgentConnectionState.Connected ? Color.ForestGreen : Color.DarkOrange;
        _centerValue.Text = snapshot.CenterUrl;
        _nameValue.Text = snapshot.AgentName;
        _agentIdValue.Text = string.IsNullOrWhiteSpace(snapshot.AgentId) ? "-" : snapshot.AgentId;
        _taskValue.Text = snapshot.CurrentTask;
        _messageValue.Text = snapshot.LastMessage;
        _heartbeatValue.Text = snapshot.LastHeartbeatAt?.LocalDateTime.ToString("yyyy-MM-dd HH:mm:ss", CultureInfo.GetCultureInfo("zh-CN")) ?? "-";
        _registrationGroup.Text = snapshot.ConnectionState == AgentConnectionState.NotRegistered ? "中心注册" : "中心连接";
        _centerInput.Enabled = snapshot.ConnectionState == AgentConnectionState.NotRegistered;
        _nameInput.Enabled = snapshot.ConnectionState == AgentConnectionState.NotRegistered;
        _registrationCodeInput.Enabled = snapshot.ConnectionState == AgentConnectionState.NotRegistered;
        _registerButton.Visible = snapshot.ConnectionState == AgentConnectionState.NotRegistered;
        _connectionButton.Visible = snapshot.ConnectionState != AgentConnectionState.NotRegistered;
        _connectionButton.Text = snapshot.ConnectionState == AgentConnectionState.Connected ? "断开中心" : "重新连接";
        _clearButton.Visible = snapshot.ConnectionState != AgentConnectionState.NotRegistered;
    }

    public void SetRegistrationRunning(bool running) { _registerButton.Enabled = !running; _registerButton.Text = running ? "正在注册..." : "注册并连接"; }
    public void ClearRegistrationCode() => _registrationCodeInput.Clear();
    public void SetSelfTestRunning(bool running) { _selfTestButton.Enabled = !running; _selfTestButton.Text = running ? "正在测试..." : "OPC UA 本地自检"; }
    public void ExitApplication() { _allowClose = true; Close(); }

    private void OnFormClosing(object? sender, FormClosingEventArgs args) { if (_allowClose) return; args.Cancel = true; Hide(); }
    private static Label CreateValueLabel() => new() { Dock = DockStyle.Fill, AutoSize = true, Padding = new Padding(0, 4, 0, 4), MaximumSize = new Size(560, 0) };
    private static void AddRow(TableLayoutPanel table, string name, Control value) { var row = table.RowCount++; table.RowStyles.Add(new RowStyle(SizeType.AutoSize)); table.Controls.Add(new Label { Text = name, AutoSize = true, ForeColor = Color.DimGray, Padding = new Padding(0, 4, 0, 4) }, 0, row); table.Controls.Add(value, 1, row); }
    private static void AddInputRow(TableLayoutPanel table, int row, string name, Control input) { table.RowStyles.Add(new RowStyle(SizeType.AutoSize)); table.Controls.Add(new Label { Text = name, AutoSize = true, Padding = new Padding(0, 5, 0, 4) }, 0, row); table.Controls.Add(input, 1, row); table.SetColumnSpan(input, 2); }
}
