using System.Text.Json;using InduForge.Collector.Adapters.Hsl;using InduForge.Collector.Contracts;
namespace InduForge.Collector.Drivers.Robots.Tests;
public sealed class RobotDriverTests
{
 [Fact]public void UsesDemoDefaults(){var estun=RobotOptions.Parse(Profile("estun",new{host="127.0.0.1"}),HslSpecializedNetworkProtocol.EstunRobot);var fanuc=RobotOptions.Parse(Profile("fanuc",new{host="127.0.0.1"}),HslSpecializedNetworkProtocol.FanucRobot);Assert.Equal(502,estun.Port);Assert.Equal(HslDataFormat.CDAB,estun.DataFormat);Assert.Equal(60008,fanuc.Port);Assert.Equal(100,fanuc.RetainTimeMs);}
 [Theory][InlineData("0")][InlineData("SDO1")][InlineData("R1")][InlineData("CurrentAlarm")]
 public void AcceptsDemoAddresses(string address)=>Assert.Equal(address,RobotAddress.Parse(Point(address),HslSpecializedNetworkProtocol.FanucRobot).Address);
 static ConnectionProfile Profile(string family,object config)=>new(family,JsonSerializer.SerializeToElement(config),JsonSerializer.SerializeToElement(new{}));static PointReadRequest Point(string address)=>new(address,JsonSerializer.SerializeToElement(new{address}),"int16",1,JsonSerializer.SerializeToElement(new{}));
}
