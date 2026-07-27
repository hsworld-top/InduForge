using System.Text.Json;using InduForge.Collector.Contracts;
namespace InduForge.Collector.Drivers.MqttRpc.Tests;
public sealed class MqttRpcDriverTests
{
 [Fact]public void UsesDemoDefaults(){var options=MqttRpcOptions.Parse(Profile(new{host="127.0.0.1",clientId="",deviceTopic="",useRsa=false}));Assert.Equal(1883,options.Port);Assert.False(options.UseRsa);}
 [Theory][InlineData("100")][InlineData("DB1.0")][InlineData("D100")]
 public void AcceptsRemoteDeviceAddress(string value)=>Assert.Equal(value,MqttRpcAddress.Parse(Point(value)).Address);
 static ConnectionProfile Profile(object config)=>new("mqtt-rpc",JsonSerializer.SerializeToElement(config),JsonSerializer.SerializeToElement(new{}));static PointReadRequest Point(string address)=>new(address,JsonSerializer.SerializeToElement(new{address}),"int16",1,JsonSerializer.SerializeToElement(new{}));
}
