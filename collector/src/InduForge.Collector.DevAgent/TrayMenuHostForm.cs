using System.Drawing;

namespace InduForge.Collector.DevAgent;

// 菜单宿主跟随托盘鼠标所在显示器，避免状态窗口位于另一块屏幕时发生 DPI 坐标错位。
internal sealed class TrayMenuHostForm : Form
{
    public TrayMenuHostForm()
    {
        FormBorderStyle = FormBorderStyle.None;
        ShowInTaskbar = false;
        StartPosition = FormStartPosition.Manual;
        Size = new Size(1, 1);
        BackColor = Color.Magenta;
        TransparencyKey = Color.Magenta;
    }

    public void ActivateAt(Point screenLocation)
    {
        Location = screenLocation;
        if (!Visible) Show();
        Activate();
    }
}
