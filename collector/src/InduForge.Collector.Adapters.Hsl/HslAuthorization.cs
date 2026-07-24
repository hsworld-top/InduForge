using HslCommunication;

namespace InduForge.Collector.Adapters.Hsl;

public enum HslAuthorizationStatus
{
    NotConfigured,
    Activated,
    Rejected,
}

public sealed record HslAuthorizationResult(HslAuthorizationStatus Status)
{
    public bool IsActivated => Status == HslAuthorizationStatus.Activated;
}

public static class HslAuthorizationInitializer
{
    // HSL 授权是进程级全局状态，必须在创建任何 HSL 通信客户端之前调用一次。
    public static HslAuthorizationResult Initialize(string? authorizationCode)
    {
        if (string.IsNullOrWhiteSpace(authorizationCode))
        {
            return new HslAuthorizationResult(HslAuthorizationStatus.NotConfigured);
        }

        var activated = Authorization.SetAuthorizationCode(authorizationCode.Trim());
        return new HslAuthorizationResult(
            activated ? HslAuthorizationStatus.Activated : HslAuthorizationStatus.Rejected);
    }
}
