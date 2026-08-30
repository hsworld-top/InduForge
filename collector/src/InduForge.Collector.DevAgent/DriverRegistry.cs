using System.Reflection;
using System.Text.Json;
using InduForge.Collector.Contracts;
using InduForge.Collector.DriverHosting;
using InduForge.Collector.Drivers.AllenBradleyEtherNetIp;
using InduForge.Collector.Drivers.BeckhoffAds;
using InduForge.Collector.Drivers.Cimon;
using InduForge.Collector.Drivers.Dcs;
using InduForge.Collector.Drivers.Delta;
using InduForge.Collector.Drivers.FatekProgramTcp;
using InduForge.Collector.Drivers.Freedom;
using InduForge.Collector.Drivers.Fuji;
using InduForge.Collector.Drivers.GeSrtpTcp;
using InduForge.Collector.Drivers.Iec104;
using InduForge.Collector.Drivers.InstrumentMeters;
using InduForge.Collector.Drivers.InovanceModbusTcp;
using InduForge.Collector.Drivers.Keyence;
using InduForge.Collector.Drivers.LsisFastEnet;
using InduForge.Collector.Drivers.MegMeet;
using InduForge.Collector.Drivers.PanasonicMewtocolTcp;
using InduForge.Collector.Drivers.PanasonicMcBinary;
using InduForge.Collector.Drivers.MitsubishiMc3ETcp;
using InduForge.Collector.Drivers.ModbusRtu;
using InduForge.Collector.Drivers.ModbusTcp;
using InduForge.Collector.Drivers.MqttRpc;
using InduForge.Collector.Drivers.OmronFins;
using InduForge.Collector.Drivers.OpcUa;
using InduForge.Collector.Drivers.OrientalMotor;
using InduForge.Collector.Drivers.Robots;
using InduForge.Collector.Drivers.SiemensS7Tcp;
using InduForge.Collector.Drivers.ToyoPuc;
using InduForge.Collector.Drivers.Turck;
using InduForge.Collector.Drivers.Xinje;
using InduForge.Collector.Drivers.Vigor;
using InduForge.Collector.Drivers.Yamatake;
using InduForge.Collector.Drivers.Yaskawa;
using InduForge.Collector.Drivers.Yokogawa;

namespace InduForge.Collector.DevAgent;

internal sealed class DriverRegistry
{
    private const string AllenBradleyEtherNetIpManifestResource = "InduForge.Collector.DevAgent.CollectorProtocols.allen-bradley.ethernet-ip.manifest.json";
    private const string AllenBradleyConnectedCipManifestResource = "InduForge.Collector.DevAgent.CollectorProtocols.allen-bradley.connected-cip.manifest.json";
    private const string AllenBradleyMicroCipManifestResource = "InduForge.Collector.DevAgent.CollectorProtocols.allen-bradley.micro-cip.manifest.json";
    private const string AllenBradleyPcccManifestResource = "InduForge.Collector.DevAgent.CollectorProtocols.allen-bradley.pccc.manifest.json";
    private const string AllenBradleySlcManifestResource = "InduForge.Collector.DevAgent.CollectorProtocols.allen-bradley.slc.manifest.json";
    private const string AllenBradleyDf1SerialManifestResource = "InduForge.Collector.DevAgent.CollectorProtocols.allen-bradley.df1-serial.manifest.json";
    private const string BeckhoffAdsManifestResource = "InduForge.Collector.DevAgent.CollectorProtocols.beckhoff.ads-tcp.manifest.json";
    private const string CimonHmiProtocolManifestResource = "InduForge.Collector.DevAgent.CollectorProtocols.cimon.hmi-protocol.manifest.json";
    private const string Cjt188SerialManifestResource = "InduForge.Collector.DevAgent.CollectorProtocols.cjt188.serial.manifest.json";
    private const string Cjt188TcpManifestResource = "InduForge.Collector.DevAgent.CollectorProtocols.cjt188.tcp.manifest.json";
    private const string Dam3601SerialManifestResource = "InduForge.Collector.DevAgent.CollectorProtocols.dam3601.serial.manifest.json";
    private const string EcFanMachineSerialManifestResource = "InduForge.Collector.DevAgent.CollectorProtocols.ec-fan.machine-serial.manifest.json";
    private const string DcsNanJingAutoManifestResource = "InduForge.Collector.DevAgent.CollectorProtocols.dcs.nanjing-auto.manifest.json";
    private const string DelixiDtsu6606ManifestResource = "InduForge.Collector.DevAgent.CollectorProtocols.delixi.dtsu6606.manifest.json";
    private const string Dlt6452007SerialManifestResource = "InduForge.Collector.DevAgent.CollectorProtocols.dlt645.2007-serial.manifest.json";
    private const string Dlt6452007OverTcpManifestResource = "InduForge.Collector.DevAgent.CollectorProtocols.dlt645.2007-over-tcp.manifest.json";
    private const string Dlt6451997SerialManifestResource = "InduForge.Collector.DevAgent.CollectorProtocols.dlt645.1997-serial.manifest.json";
    private const string Dlt6451997OverTcpManifestResource = "InduForge.Collector.DevAgent.CollectorProtocols.dlt645.1997-over-tcp.manifest.json";
    private const string Dlt698SerialManifestResource = "InduForge.Collector.DevAgent.CollectorProtocols.dlt698.serial.manifest.json";
    private const string Dlt698OverTcpManifestResource = "InduForge.Collector.DevAgent.CollectorProtocols.dlt698.over-tcp.manifest.json";
    private const string Dlt698TcpNetManifestResource = "InduForge.Collector.DevAgent.CollectorProtocols.dlt698.tcp-net.manifest.json";
    private const string DeltaTcpManifestResource = "InduForge.Collector.DevAgent.CollectorProtocols.delta.tcp.manifest.json";
    private const string DeltaRtuOverTcpManifestResource = "InduForge.Collector.DevAgent.CollectorProtocols.delta.rtu-over-tcp.manifest.json";
    private const string DeltaAsciiOverTcpManifestResource = "InduForge.Collector.DevAgent.CollectorProtocols.delta.ascii-over-tcp.manifest.json";
    private const string DeltaRtuManifestResource = "InduForge.Collector.DevAgent.CollectorProtocols.delta.rtu.manifest.json";
    private const string DeltaAsciiManifestResource = "InduForge.Collector.DevAgent.CollectorProtocols.delta.ascii.manifest.json";
    private const string FatekProgramTcpManifestResource = "InduForge.Collector.DevAgent.CollectorProtocols.fatek.program-tcp.manifest.json";
    private const string FatekProgramSerialManifestResource = "InduForge.Collector.DevAgent.CollectorProtocols.fatek.program-serial.manifest.json";
    private const string FreedomTcpManifestResource = "InduForge.Collector.DevAgent.CollectorProtocols.freedom.tcp.manifest.json";
    private const string FreedomUdpManifestResource = "InduForge.Collector.DevAgent.CollectorProtocols.freedom.udp.manifest.json";
    private const string FreedomSerialManifestResource = "InduForge.Collector.DevAgent.CollectorProtocols.freedom.serial.manifest.json";
    private const string FujiCommandSettingTcpManifestResource = "InduForge.Collector.DevAgent.CollectorProtocols.fuji.command-setting-tcp.manifest.json";
    private const string FujiSphTcpManifestResource = "InduForge.Collector.DevAgent.CollectorProtocols.fuji.sph-tcp.manifest.json";
    private const string FujiSpbOverTcpManifestResource = "InduForge.Collector.DevAgent.CollectorProtocols.fuji.spb-over-tcp.manifest.json";
    private const string FujiSpbManifestResource = "InduForge.Collector.DevAgent.CollectorProtocols.fuji.spb.manifest.json";
    private const string GeSrtpTcpManifestResource = "InduForge.Collector.DevAgent.CollectorProtocols.ge.srtp-tcp.manifest.json";
    private const string Iec104ManifestResource = "InduForge.Collector.DevAgent.CollectorProtocols.iec.60870-5-104.manifest.json";
    private const string InovanceModbusTcpManifestResource = "InduForge.Collector.DevAgent.CollectorProtocols.inovance.modbus-tcp.manifest.json";
    private const string InovanceModbusSerialManifestResource = "InduForge.Collector.DevAgent.CollectorProtocols.inovance.modbus-serial.manifest.json";
    private const string InovanceModbusRtuOverTcpManifestResource = "InduForge.Collector.DevAgent.CollectorProtocols.inovance.modbus-rtu-over-tcp.manifest.json";
    private const string InovanceConnectedCipManifestResource = "InduForge.Collector.DevAgent.CollectorProtocols.inovance.connected-cip.manifest.json";
    private const string InovanceEasyNetManifestResource = "InduForge.Collector.DevAgent.CollectorProtocols.inovance.easy-net.manifest.json";
    private const string InovanceComputerLinkManifestResource = "InduForge.Collector.DevAgent.CollectorProtocols.inovance.computer-link.manifest.json";
    private const string KeyenceMc3ETcpManifestResource = "InduForge.Collector.DevAgent.CollectorProtocols.keyence.mc-3e-tcp.manifest.json";
    private const string KeyenceMcAsciiTcpManifestResource = "InduForge.Collector.DevAgent.CollectorProtocols.keyence.mc-ascii-tcp.manifest.json";
    private const string KeyenceKvOldTcpManifestResource = "InduForge.Collector.DevAgent.CollectorProtocols.keyence.kv-old-tcp.manifest.json";
    private const string KeyenceNanoTcpManifestResource = "InduForge.Collector.DevAgent.CollectorProtocols.keyence.nano-tcp.manifest.json";
    private const string KeyenceNanoSerialManifestResource = "InduForge.Collector.DevAgent.CollectorProtocols.keyence.nano-serial.manifest.json";
    private const string KeyenceNanoSerialOverTcpManifestResource = "InduForge.Collector.DevAgent.CollectorProtocols.keyence.nano-serial-over-tcp.manifest.json";
    private const string LsisFastEnetManifestResource = "InduForge.Collector.DevAgent.CollectorProtocols.lsis.fast-enet.manifest.json";
    private const string LsisCnetManifestResource = "InduForge.Collector.DevAgent.CollectorProtocols.lsis.cnet.manifest.json";
    private const string LsisCnetOverTcpManifestResource = "InduForge.Collector.DevAgent.CollectorProtocols.lsis.cnet-over-tcp.manifest.json";
    private const string LsisCpuSerialManifestResource = "InduForge.Collector.DevAgent.CollectorProtocols.lsis.cpu-serial.manifest.json";
    private const string MegMeetTcpManifestResource = "InduForge.Collector.DevAgent.CollectorProtocols.megmeet.tcp.manifest.json";
    private const string MegMeetRtuOverTcpManifestResource = "InduForge.Collector.DevAgent.CollectorProtocols.megmeet.rtu-over-tcp.manifest.json";
    private const string MegMeetRtuManifestResource = "InduForge.Collector.DevAgent.CollectorProtocols.megmeet.rtu.manifest.json";
    private const string PanasonicMewtocolTcpManifestResource = "InduForge.Collector.DevAgent.CollectorProtocols.panasonic.mewtocol-tcp.manifest.json";
    private const string PanasonicMewtocolSerialManifestResource = "InduForge.Collector.DevAgent.CollectorProtocols.panasonic.mewtocol-serial.manifest.json";
    private const string PanasonicMcBinaryTcpManifestResource = "InduForge.Collector.DevAgent.CollectorProtocols.panasonic.mc-binary-tcp.manifest.json";
    private const string OpcUaManifestResource = "InduForge.Collector.DevAgent.CollectorProtocols.opcua.standard.manifest.json";
    private const string MitsubishiMc3ETcpManifestResource = "InduForge.Collector.DevAgent.CollectorProtocols.mitsubishi.mc-3e-tcp.manifest.json";
    private const string MitsubishiA1EAsciiTcpManifestResource = "InduForge.Collector.DevAgent.CollectorProtocols.mitsubishi.a1e-ascii-tcp.manifest.json";
    private const string MitsubishiA1EBinaryTcpManifestResource = "InduForge.Collector.DevAgent.CollectorProtocols.mitsubishi.a1e-binary-tcp.manifest.json";
    private const string MitsubishiMcAsciiTcpManifestResource = "InduForge.Collector.DevAgent.CollectorProtocols.mitsubishi.mc-ascii-tcp.manifest.json";
    private const string MitsubishiMcAsciiUdpManifestResource = "InduForge.Collector.DevAgent.CollectorProtocols.mitsubishi.mc-ascii-udp.manifest.json";
    private const string MitsubishiMcBinaryUdpManifestResource = "InduForge.Collector.DevAgent.CollectorProtocols.mitsubishi.mc-binary-udp.manifest.json";
    private const string MitsubishiMcRBinaryTcpManifestResource = "InduForge.Collector.DevAgent.CollectorProtocols.mitsubishi.mc-r-binary-tcp.manifest.json";
    private const string MitsubishiCipManifestResource = "InduForge.Collector.DevAgent.CollectorProtocols.mitsubishi.cip.manifest.json";
    private const string MitsubishiA3CSerialManifestResource = "InduForge.Collector.DevAgent.CollectorProtocols.mitsubishi.a3c-serial.manifest.json";
    private const string MitsubishiA3CSerialOverTcpManifestResource = "InduForge.Collector.DevAgent.CollectorProtocols.mitsubishi.a3c-serial-over-tcp.manifest.json";
    private const string MitsubishiFxLinksSerialManifestResource = "InduForge.Collector.DevAgent.CollectorProtocols.mitsubishi.fx-links-serial.manifest.json";
    private const string MitsubishiFxLinksOverTcpManifestResource = "InduForge.Collector.DevAgent.CollectorProtocols.mitsubishi.fx-links-over-tcp.manifest.json";
    private const string MitsubishiFxSerialManifestResource = "InduForge.Collector.DevAgent.CollectorProtocols.mitsubishi.fx-serial.manifest.json";
    private const string MitsubishiFxSerialOverTcpManifestResource = "InduForge.Collector.DevAgent.CollectorProtocols.mitsubishi.fx-serial-over-tcp.manifest.json";
    private const string ModbusRtuManifestResource = "InduForge.Collector.DevAgent.CollectorProtocols.modbus.rtu.manifest.json";
    private const string ModbusRtuOverTcpManifestResource = "InduForge.Collector.DevAgent.CollectorProtocols.modbus.rtu-over-tcp.manifest.json";
    private const string ModbusAsciiManifestResource = "InduForge.Collector.DevAgent.CollectorProtocols.modbus.ascii.manifest.json";
    private const string ModbusAsciiOverTcpManifestResource = "InduForge.Collector.DevAgent.CollectorProtocols.modbus.ascii-over-tcp.manifest.json";
    private const string ModbusTcpManifestResource = "InduForge.Collector.DevAgent.CollectorProtocols.modbus.tcp.manifest.json";
    private const string ModbusUdpManifestResource = "InduForge.Collector.DevAgent.CollectorProtocols.modbus.udp.manifest.json";
    private const string MqttRpcDeviceManifestResource = "InduForge.Collector.DevAgent.CollectorProtocols.mqtt.rpc-device.manifest.json";
    private const string OmronFinsTcpManifestResource = "InduForge.Collector.DevAgent.CollectorProtocols.omron.fins-tcp.manifest.json";
    private const string OmronFinsUdpManifestResource = "InduForge.Collector.DevAgent.CollectorProtocols.omron.fins-udp.manifest.json";
    private const string OmronCipManifestResource = "InduForge.Collector.DevAgent.CollectorProtocols.omron.cip.manifest.json";
    private const string OmronConnectedCipManifestResource = "InduForge.Collector.DevAgent.CollectorProtocols.omron.connected-cip.manifest.json";
    private const string OmronHostLinkManifestResource = "InduForge.Collector.DevAgent.CollectorProtocols.omron.hostlink.manifest.json";
    private const string OmronHostLinkOverTcpManifestResource = "InduForge.Collector.DevAgent.CollectorProtocols.omron.hostlink-over-tcp.manifest.json";
    private const string OmronHostLinkCModeManifestResource = "InduForge.Collector.DevAgent.CollectorProtocols.omron.hostlink-cmode.manifest.json";
    private const string OmronHostLinkCModeOverTcpManifestResource = "InduForge.Collector.DevAgent.CollectorProtocols.omron.hostlink-cmode-over-tcp.manifest.json";
    private const string OrientalMotorEipManifestResource = "InduForge.Collector.DevAgent.CollectorProtocols.oriental-motor.eip.manifest.json";
    private const string RkcTemperatureControllerSerialManifestResource = "InduForge.Collector.DevAgent.CollectorProtocols.rkc.temperature-controller-serial.manifest.json";
    private const string RkcTemperatureControllerTcpManifestResource = "InduForge.Collector.DevAgent.CollectorProtocols.rkc.temperature-controller-tcp.manifest.json";
    private const string EstunRobotManifestResource = "InduForge.Collector.DevAgent.CollectorProtocols.robot.estun-tcp.manifest.json";
    private const string FanucRobotManifestResource = "InduForge.Collector.DevAgent.CollectorProtocols.robot.fanuc-interface.manifest.json";
    private const string SiemensS7TcpManifestResource = "InduForge.Collector.DevAgent.CollectorProtocols.siemens.s7-tcp.manifest.json";
    private const string SiemensS7PlusManifestResource = "InduForge.Collector.DevAgent.CollectorProtocols.siemens.s7-plus.manifest.json";
    private const string SiemensPpiManifestResource = "InduForge.Collector.DevAgent.CollectorProtocols.siemens.ppi.manifest.json";
    private const string SiemensPpiOverTcpManifestResource = "InduForge.Collector.DevAgent.CollectorProtocols.siemens.ppi-over-tcp.manifest.json";
    private const string SiemensMpiManifestResource = "InduForge.Collector.DevAgent.CollectorProtocols.siemens.mpi.manifest.json";
    private const string SiemensFetchWriteManifestResource = "InduForge.Collector.DevAgent.CollectorProtocols.siemens.fetch-write.manifest.json";
    private const string SiemensWebApiManifestResource = "InduForge.Collector.DevAgent.CollectorProtocols.siemens.web-api.manifest.json";
    private const string ToyoPucManifestResource = "InduForge.Collector.DevAgent.CollectorProtocols.toyo.puc.manifest.json";
    private const string TurckReaderTcpManifestResource = "InduForge.Collector.DevAgent.CollectorProtocols.turck.reader-tcp.manifest.json";
    private const string XinjeTcpManifestResource = "InduForge.Collector.DevAgent.CollectorProtocols.xinje.tcp.manifest.json";
    private const string XinjeRtuOverTcpManifestResource = "InduForge.Collector.DevAgent.CollectorProtocols.xinje.rtu-over-tcp.manifest.json";
    private const string XinjeRtuManifestResource = "InduForge.Collector.DevAgent.CollectorProtocols.xinje.rtu.manifest.json";
    private const string XinjeInternalTcpManifestResource = "InduForge.Collector.DevAgent.CollectorProtocols.xinje.internal-tcp.manifest.json";
    private const string VigorSerialOverTcpManifestResource = "InduForge.Collector.DevAgent.CollectorProtocols.vigor.serial-over-tcp.manifest.json";
    private const string VigorSerialManifestResource = "InduForge.Collector.DevAgent.CollectorProtocols.vigor.serial.manifest.json";
    private const string YamatakeDigitronSerialManifestResource = "InduForge.Collector.DevAgent.CollectorProtocols.yamatake.digitron-serial.manifest.json";
    private const string YamatakeDigitronTcpManifestResource = "InduForge.Collector.DevAgent.CollectorProtocols.yamatake.digitron-tcp.manifest.json";
    private const string YuDianAiBusManifestResource = "InduForge.Collector.DevAgent.CollectorProtocols.yudian.ai-bus.manifest.json";
    private const string YaskawaMemobusTcpManifestResource = "InduForge.Collector.DevAgent.CollectorProtocols.yaskawa.memobus-tcp.manifest.json";
    private const string YaskawaMemobusUdpManifestResource = "InduForge.Collector.DevAgent.CollectorProtocols.yaskawa.memobus-udp.manifest.json";
    private const string YokogawaLinkTcpManifestResource = "InduForge.Collector.DevAgent.CollectorProtocols.yokogawa.link-tcp.manifest.json";
    private static readonly JsonSerializerOptions ManifestJsonOptions = new(JsonSerializerDefaults.Web);
    private readonly DriverHostRegistry _registry;

    public DriverRegistry(IEnumerable<Func<IIndustrialDriver>> factories, IReadOnlyDictionary<string, DriverManifest>? manifests = null)
    {
        _registry = new DriverHostRegistry(
            factories,
            manifests,
            manifests is null ? null : new DriverHostPlatform("devAgent", "windows-x64"));
    }

    public IReadOnlyCollection<DriverDescriptor> Descriptors => _registry.Descriptors;

    public bool UsesTransport(string driverId, string transport) =>
        _registry.UsesTransport(driverId, transport);

    public DriverDescriptor Describe(string driverId) => _registry.IsRegistered(driverId)
        ? _registry.Describe(driverId)
        : throw new CollectorTaskExecutionException("COLLECTOR_DRIVER_UNSUPPORTED", "当前 Agent 未安装该驱动", false);

    public IIndustrialDriver Create(string driverId)
    {
        if (!_registry.IsRegistered(driverId))
        {
            throw new CollectorTaskExecutionException("COLLECTOR_DRIVER_UNSUPPORTED", "当前 Agent 未安装该驱动", false);
        }
        return _registry.Create(driverId);
    }

    public static DriverRegistry CreateDefault()
    {
        var assembly = Assembly.GetExecutingAssembly();
        var allenBradleyEtherNetIpManifest = LoadManifest(assembly, AllenBradleyEtherNetIpManifestResource);
        var allenBradleyConnectedCipManifest = LoadManifest(assembly, AllenBradleyConnectedCipManifestResource);
        var allenBradleyMicroCipManifest = LoadManifest(assembly, AllenBradleyMicroCipManifestResource);
        var allenBradleyPcccManifest = LoadManifest(assembly, AllenBradleyPcccManifestResource);
        var allenBradleySlcManifest = LoadManifest(assembly, AllenBradleySlcManifestResource);
        var allenBradleyDf1SerialManifest = LoadManifest(assembly, AllenBradleyDf1SerialManifestResource);
        var beckhoffAdsManifest = LoadManifest(assembly, BeckhoffAdsManifestResource);
        var cimonHmiProtocolManifest = LoadManifest(assembly, CimonHmiProtocolManifestResource);
        var cjt188SerialManifest = LoadManifest(assembly, Cjt188SerialManifestResource);
        var cjt188TcpManifest = LoadManifest(assembly, Cjt188TcpManifestResource);
        var dam3601SerialManifest = LoadManifest(assembly, Dam3601SerialManifestResource);
        var ecFanMachineSerialManifest = LoadManifest(assembly, EcFanMachineSerialManifestResource);
        var dcsNanJingAutoManifest = LoadManifest(assembly, DcsNanJingAutoManifestResource);
        var delixiDtsu6606Manifest = LoadManifest(assembly, DelixiDtsu6606ManifestResource);
        var dlt6452007SerialManifest = LoadManifest(assembly, Dlt6452007SerialManifestResource);
        var dlt6452007OverTcpManifest = LoadManifest(assembly, Dlt6452007OverTcpManifestResource);
        var dlt6451997SerialManifest = LoadManifest(assembly, Dlt6451997SerialManifestResource);
        var dlt6451997OverTcpManifest = LoadManifest(assembly, Dlt6451997OverTcpManifestResource);
        var dlt698SerialManifest = LoadManifest(assembly, Dlt698SerialManifestResource);
        var dlt698OverTcpManifest = LoadManifest(assembly, Dlt698OverTcpManifestResource);
        var dlt698TcpNetManifest = LoadManifest(assembly, Dlt698TcpNetManifestResource);
        var deltaTcpManifest = LoadManifest(assembly, DeltaTcpManifestResource);
        var deltaRtuOverTcpManifest = LoadManifest(assembly, DeltaRtuOverTcpManifestResource);
        var deltaAsciiOverTcpManifest = LoadManifest(assembly, DeltaAsciiOverTcpManifestResource);
        var deltaRtuManifest = LoadManifest(assembly, DeltaRtuManifestResource);
        var deltaAsciiManifest = LoadManifest(assembly, DeltaAsciiManifestResource);
        var fatekProgramTcpManifest = LoadManifest(assembly, FatekProgramTcpManifestResource);
        var fatekProgramSerialManifest = LoadManifest(assembly, FatekProgramSerialManifestResource);
        var freedomTcpManifest = LoadManifest(assembly, FreedomTcpManifestResource);
        var freedomUdpManifest = LoadManifest(assembly, FreedomUdpManifestResource);
        var freedomSerialManifest = LoadManifest(assembly, FreedomSerialManifestResource);
        var fujiCommandSettingTcpManifest = LoadManifest(assembly, FujiCommandSettingTcpManifestResource);
        var fujiSphTcpManifest = LoadManifest(assembly, FujiSphTcpManifestResource);
        var fujiSpbOverTcpManifest = LoadManifest(assembly, FujiSpbOverTcpManifestResource);
        var fujiSpbManifest = LoadManifest(assembly, FujiSpbManifestResource);
        var geSrtpTcpManifest = LoadManifest(assembly, GeSrtpTcpManifestResource);
        var iec104Manifest = LoadManifest(assembly, Iec104ManifestResource);
        var inovanceModbusTcpManifest = LoadManifest(assembly, InovanceModbusTcpManifestResource);
        var inovanceModbusSerialManifest = LoadManifest(assembly, InovanceModbusSerialManifestResource);
        var inovanceModbusRtuOverTcpManifest = LoadManifest(assembly, InovanceModbusRtuOverTcpManifestResource);
        var inovanceConnectedCipManifest = LoadManifest(assembly, InovanceConnectedCipManifestResource);
        var inovanceEasyNetManifest = LoadManifest(assembly, InovanceEasyNetManifestResource);
        var inovanceComputerLinkManifest = LoadManifest(assembly, InovanceComputerLinkManifestResource);
        var keyenceMc3ETcpManifest = LoadManifest(assembly, KeyenceMc3ETcpManifestResource);
        var keyenceMcAsciiTcpManifest = LoadManifest(assembly, KeyenceMcAsciiTcpManifestResource);
        var keyenceKvOldTcpManifest = LoadManifest(assembly, KeyenceKvOldTcpManifestResource);
        var keyenceNanoTcpManifest = LoadManifest(assembly, KeyenceNanoTcpManifestResource);
        var keyenceNanoSerialManifest = LoadManifest(assembly, KeyenceNanoSerialManifestResource);
        var keyenceNanoSerialOverTcpManifest = LoadManifest(assembly, KeyenceNanoSerialOverTcpManifestResource);
        var lsisFastEnetManifest = LoadManifest(assembly, LsisFastEnetManifestResource);
        var lsisCnetManifest = LoadManifest(assembly, LsisCnetManifestResource);
        var lsisCnetOverTcpManifest = LoadManifest(assembly, LsisCnetOverTcpManifestResource);
        var lsisCpuSerialManifest = LoadManifest(assembly, LsisCpuSerialManifestResource);
        var megMeetTcpManifest = LoadManifest(assembly, MegMeetTcpManifestResource);
        var megMeetRtuOverTcpManifest = LoadManifest(assembly, MegMeetRtuOverTcpManifestResource);
        var megMeetRtuManifest = LoadManifest(assembly, MegMeetRtuManifestResource);
        var panasonicMewtocolTcpManifest = LoadManifest(assembly, PanasonicMewtocolTcpManifestResource);
        var panasonicMewtocolSerialManifest = LoadManifest(assembly, PanasonicMewtocolSerialManifestResource);
        var panasonicMcBinaryTcpManifest = LoadManifest(assembly, PanasonicMcBinaryTcpManifestResource);
        var opcUaManifest = LoadManifest(assembly, OpcUaManifestResource);
        var mitsubishiMc3ETcpManifest = LoadManifest(assembly, MitsubishiMc3ETcpManifestResource);
        var mitsubishiA1EAsciiTcpManifest = LoadManifest(assembly, MitsubishiA1EAsciiTcpManifestResource);
        var mitsubishiA1EBinaryTcpManifest = LoadManifest(assembly, MitsubishiA1EBinaryTcpManifestResource);
        var mitsubishiMcAsciiTcpManifest = LoadManifest(assembly, MitsubishiMcAsciiTcpManifestResource);
        var mitsubishiMcAsciiUdpManifest = LoadManifest(assembly, MitsubishiMcAsciiUdpManifestResource);
        var mitsubishiMcBinaryUdpManifest = LoadManifest(assembly, MitsubishiMcBinaryUdpManifestResource);
        var mitsubishiMcRBinaryTcpManifest = LoadManifest(assembly, MitsubishiMcRBinaryTcpManifestResource);
        var mitsubishiCipManifest = LoadManifest(assembly, MitsubishiCipManifestResource);
        var mitsubishiA3CSerialManifest = LoadManifest(assembly, MitsubishiA3CSerialManifestResource);
        var mitsubishiA3CSerialOverTcpManifest = LoadManifest(assembly, MitsubishiA3CSerialOverTcpManifestResource);
        var mitsubishiFxLinksSerialManifest = LoadManifest(assembly, MitsubishiFxLinksSerialManifestResource);
        var mitsubishiFxLinksOverTcpManifest = LoadManifest(assembly, MitsubishiFxLinksOverTcpManifestResource);
        var mitsubishiFxSerialManifest = LoadManifest(assembly, MitsubishiFxSerialManifestResource);
        var mitsubishiFxSerialOverTcpManifest = LoadManifest(assembly, MitsubishiFxSerialOverTcpManifestResource);
        var modbusRtuManifest = LoadManifest(assembly, ModbusRtuManifestResource);
        var modbusRtuOverTcpManifest = LoadManifest(assembly, ModbusRtuOverTcpManifestResource);
        var modbusAsciiManifest = LoadManifest(assembly, ModbusAsciiManifestResource);
        var modbusAsciiOverTcpManifest = LoadManifest(assembly, ModbusAsciiOverTcpManifestResource);
        var modbusTcpManifest = LoadManifest(assembly, ModbusTcpManifestResource);
        var modbusUdpManifest = LoadManifest(assembly, ModbusUdpManifestResource);
        var mqttRpcDeviceManifest = LoadManifest(assembly, MqttRpcDeviceManifestResource);
        var omronFinsTcpManifest = LoadManifest(assembly, OmronFinsTcpManifestResource);
        var omronFinsUdpManifest = LoadManifest(assembly, OmronFinsUdpManifestResource);
        var omronCipManifest = LoadManifest(assembly, OmronCipManifestResource);
        var omronConnectedCipManifest = LoadManifest(assembly, OmronConnectedCipManifestResource);
        var omronHostLinkManifest = LoadManifest(assembly, OmronHostLinkManifestResource);
        var omronHostLinkOverTcpManifest = LoadManifest(assembly, OmronHostLinkOverTcpManifestResource);
        var omronHostLinkCModeManifest = LoadManifest(assembly, OmronHostLinkCModeManifestResource);
        var omronHostLinkCModeOverTcpManifest = LoadManifest(assembly, OmronHostLinkCModeOverTcpManifestResource);
        var orientalMotorEipManifest = LoadManifest(assembly, OrientalMotorEipManifestResource);
        var rkcTemperatureControllerSerialManifest = LoadManifest(assembly, RkcTemperatureControllerSerialManifestResource);
        var rkcTemperatureControllerTcpManifest = LoadManifest(assembly, RkcTemperatureControllerTcpManifestResource);
        var estunRobotManifest = LoadManifest(assembly, EstunRobotManifestResource);
        var fanucRobotManifest = LoadManifest(assembly, FanucRobotManifestResource);
        var siemensS7TcpManifest = LoadManifest(assembly, SiemensS7TcpManifestResource);
        var siemensS7PlusManifest = LoadManifest(assembly, SiemensS7PlusManifestResource);
        var siemensPpiManifest = LoadManifest(assembly, SiemensPpiManifestResource);
        var siemensPpiOverTcpManifest = LoadManifest(assembly, SiemensPpiOverTcpManifestResource);
        var siemensMpiManifest = LoadManifest(assembly, SiemensMpiManifestResource);
        var siemensFetchWriteManifest = LoadManifest(assembly, SiemensFetchWriteManifestResource);
        var siemensWebApiManifest = LoadManifest(assembly, SiemensWebApiManifestResource);
        var toyoPucManifest = LoadManifest(assembly, ToyoPucManifestResource);
        var turckReaderTcpManifest = LoadManifest(assembly, TurckReaderTcpManifestResource);
        var xinjeTcpManifest = LoadManifest(assembly, XinjeTcpManifestResource);
        var xinjeRtuOverTcpManifest = LoadManifest(assembly, XinjeRtuOverTcpManifestResource);
        var xinjeRtuManifest = LoadManifest(assembly, XinjeRtuManifestResource);
        var xinjeInternalTcpManifest = LoadManifest(assembly, XinjeInternalTcpManifestResource);
        var vigorSerialOverTcpManifest = LoadManifest(assembly, VigorSerialOverTcpManifestResource);
        var vigorSerialManifest = LoadManifest(assembly, VigorSerialManifestResource);
        var yamatakeDigitronSerialManifest = LoadManifest(assembly, YamatakeDigitronSerialManifestResource);
        var yamatakeDigitronTcpManifest = LoadManifest(assembly, YamatakeDigitronTcpManifestResource);
        var yuDianAiBusManifest = LoadManifest(assembly, YuDianAiBusManifestResource);
        var yaskawaMemobusTcpManifest = LoadManifest(assembly, YaskawaMemobusTcpManifestResource);
        var yaskawaMemobusUdpManifest = LoadManifest(assembly, YaskawaMemobusUdpManifestResource);
        var yokogawaLinkTcpManifest = LoadManifest(assembly, YokogawaLinkTcpManifestResource);
        return new DriverRegistry(
        [
            () => new AllenBradleyEtherNetIpDriver(),
            () => new AllenBradleyConnectedCipDriver(),
            () => new AllenBradleyMicroCipDriver(),
            () => new AllenBradleyPcccDriver(),
            () => new AllenBradleySlcDriver(),
            () => new AllenBradleyDf1SerialDriver(),
            () => new BeckhoffAdsDriver(),
            () => new CimonDriver(),
            () => new Cjt188SerialDriver(),
            () => new Cjt188TcpDriver(),
            () => new Dam3601SerialDriver(),
            () => new EcFanMachineDriver(),
            () => new DcsNanJingAutoDriver(),
            () => new DelixiDtsu6606Driver(),
            () => new Dlt645SerialDriver(),
            () => new Dlt645OverTcpDriver(),
            () => new Dlt645With1997SerialDriver(),
            () => new Dlt645With1997OverTcpDriver(),
            () => new Dlt698SerialDriver(),
            () => new Dlt698OverTcpDriver(),
            () => new Dlt698TcpNetDriver(),
            () => new DeltaTcpDriver(),
            () => new DeltaRtuOverTcpDriver(),
            () => new DeltaAsciiOverTcpDriver(),
            () => new DeltaRtuDriver(),
            () => new DeltaAsciiDriver(),
            () => new FatekProgramTcpDriver(),
            () => new FatekProgramSerialDriver(),
            () => new FreedomTcpDriver(),
            () => new FreedomUdpDriver(),
            () => new FreedomSerialDriver(),
            () => new FujiCommandSettingTypeTcpDriver(),
            () => new FujiSphTcpDriver(),
            () => new FujiSpbOverTcpDriver(),
            () => new FujiSpbDriver(),
            () => new GeSrtpTcpDriver(),
            () => new Iec104Driver(),
            () => new InovanceModbusTcpDriver(),
            () => new InovanceModbusSerialDriver(),
            () => new InovanceModbusRtuOverTcpDriver(),
            () => new InovanceConnectedCipDriver(),
            () => new InovanceEasyNetDriver(),
            () => new InovanceComputerLinkDriver(),
            () => new KeyenceMc3ETcpDriver(),
            () => new KeyenceMcAsciiTcpDriver(),
            () => new KeyenceKvOldTcpDriver(),
            () => new KeyenceNanoTcpDriver(),
            () => new KeyenceNanoSerialDriver(),
            () => new KeyenceNanoSerialOverTcpDriver(),
            () => new LsisFastEnetDriver(),
            () => new LsisCnetDriver(),
            () => new LsisCnetOverTcpDriver(),
            () => new LsisCpuSerialDriver(),
            () => new MegMeetTcpDriver(),
            () => new MegMeetRtuOverTcpDriver(),
            () => new MegMeetRtuDriver(),
            () => new PanasonicMewtocolTcpDriver(),
            () => new PanasonicMewtocolSerialDriver(),
            () => new PanasonicMcBinaryDriver(),
            () => new OpcUaDriver(),
            () => new MitsubishiMc3ETcpDriver(),
            () => new MitsubishiA1EAsciiTcpDriver(),
            () => new MitsubishiA1EBinaryTcpDriver(),
            () => new MitsubishiMcAsciiTcpDriver(),
            () => new MitsubishiMcAsciiUdpDriver(),
            () => new MitsubishiMcBinaryUdpDriver(),
            () => new MitsubishiMcRBinaryTcpDriver(),
            () => new MitsubishiCipDriver(),
            () => new MitsubishiA3CSerialDriver(),
            () => new MitsubishiA3CSerialOverTcpDriver(),
            () => new MitsubishiFxLinksSerialDriver(),
            () => new MitsubishiFxLinksOverTcpDriver(),
            () => new MitsubishiFxSerialDriver(),
            () => new MitsubishiFxSerialOverTcpDriver(),
            () => new ModbusRtuDriver(),
            () => new ModbusRtuOverTcpDriver(),
            () => new ModbusAsciiDriver(),
            () => new ModbusAsciiOverTcpDriver(),
            () => new ModbusTcpDriver(),
            () => new ModbusUdpDriver(),
            () => new MqttRpcDeviceDriver(),
            () => new OmronFinsTcpDriver(),
            () => new OmronFinsUdpDriver(),
            () => new OmronCipDriver(),
            () => new OmronConnectedCipDriver(),
            () => new OmronHostLinkDriver(),
            () => new OmronHostLinkOverTcpDriver(),
            () => new OmronHostLinkCModeDriver(),
            () => new OmronHostLinkCModeOverTcpDriver(),
            () => new OrientalMotorEipDriver(),
            () => new RkcTemperatureControllerSerialDriver(),
            () => new RkcTemperatureControllerTcpDriver(),
            () => new EstunRobotDriver(),
            () => new FanucRobotDriver(),
            () => new SiemensS7TcpDriver(),
            () => new SiemensS7PlusDriver(),
            () => new SiemensPpiDriver(),
            () => new SiemensPpiOverTcpDriver(),
            () => new SiemensMpiDriver(),
            () => new SiemensFetchWriteDriver(),
            () => new SiemensWebApiDriver(),
            () => new ToyoPucDriver(),
            () => new TurckReaderDriver(),
            () => new XinjeTcpDriver(),
            () => new XinjeRtuOverTcpDriver(),
            () => new XinjeRtuDriver(),
            () => new XinjeInternalTcpDriver(),
            () => new VigorSerialOverTcpDriver(),
            () => new VigorSerialDriver(),
            () => new YamatakeDigitronSerialDriver(),
            () => new YamatakeDigitronTcpDriver(),
            () => new YuDianAiBusDriver(),
            () => new YaskawaMemobusTcpDriver(),
            () => new YaskawaMemobusUdpDriver(),
            () => new YokogawaLinkTcpDriver(),
        ],
        new Dictionary<string, DriverManifest>(StringComparer.Ordinal)
        {
            [allenBradleyEtherNetIpManifest.DriverId] = allenBradleyEtherNetIpManifest,
            [allenBradleyConnectedCipManifest.DriverId] = allenBradleyConnectedCipManifest,
            [allenBradleyMicroCipManifest.DriverId] = allenBradleyMicroCipManifest,
            [allenBradleyPcccManifest.DriverId] = allenBradleyPcccManifest,
            [allenBradleySlcManifest.DriverId] = allenBradleySlcManifest,
            [allenBradleyDf1SerialManifest.DriverId] = allenBradleyDf1SerialManifest,
            [beckhoffAdsManifest.DriverId] = beckhoffAdsManifest,
            [cimonHmiProtocolManifest.DriverId] = cimonHmiProtocolManifest,
            [cjt188SerialManifest.DriverId] = cjt188SerialManifest,
            [cjt188TcpManifest.DriverId] = cjt188TcpManifest,
            [dam3601SerialManifest.DriverId] = dam3601SerialManifest,
            [ecFanMachineSerialManifest.DriverId] = ecFanMachineSerialManifest,
            [dcsNanJingAutoManifest.DriverId] = dcsNanJingAutoManifest,
            [delixiDtsu6606Manifest.DriverId] = delixiDtsu6606Manifest,
            [dlt6452007SerialManifest.DriverId] = dlt6452007SerialManifest,
            [dlt6452007OverTcpManifest.DriverId] = dlt6452007OverTcpManifest,
            [dlt6451997SerialManifest.DriverId] = dlt6451997SerialManifest,
            [dlt6451997OverTcpManifest.DriverId] = dlt6451997OverTcpManifest,
            [dlt698SerialManifest.DriverId] = dlt698SerialManifest,
            [dlt698OverTcpManifest.DriverId] = dlt698OverTcpManifest,
            [dlt698TcpNetManifest.DriverId] = dlt698TcpNetManifest,
            [deltaTcpManifest.DriverId] = deltaTcpManifest,
            [deltaRtuOverTcpManifest.DriverId] = deltaRtuOverTcpManifest,
            [deltaAsciiOverTcpManifest.DriverId] = deltaAsciiOverTcpManifest,
            [deltaRtuManifest.DriverId] = deltaRtuManifest,
            [deltaAsciiManifest.DriverId] = deltaAsciiManifest,
            [fatekProgramTcpManifest.DriverId] = fatekProgramTcpManifest,
            [fatekProgramSerialManifest.DriverId] = fatekProgramSerialManifest,
            [freedomTcpManifest.DriverId] = freedomTcpManifest,
            [freedomUdpManifest.DriverId] = freedomUdpManifest,
            [freedomSerialManifest.DriverId] = freedomSerialManifest,
            [fujiCommandSettingTcpManifest.DriverId] = fujiCommandSettingTcpManifest,
            [fujiSphTcpManifest.DriverId] = fujiSphTcpManifest,
            [fujiSpbOverTcpManifest.DriverId] = fujiSpbOverTcpManifest,
            [fujiSpbManifest.DriverId] = fujiSpbManifest,
            [geSrtpTcpManifest.DriverId] = geSrtpTcpManifest,
            [iec104Manifest.DriverId] = iec104Manifest,
            [inovanceModbusTcpManifest.DriverId] = inovanceModbusTcpManifest,
            [inovanceModbusSerialManifest.DriverId] = inovanceModbusSerialManifest,
            [inovanceModbusRtuOverTcpManifest.DriverId] = inovanceModbusRtuOverTcpManifest,
            [inovanceConnectedCipManifest.DriverId] = inovanceConnectedCipManifest,
            [inovanceEasyNetManifest.DriverId] = inovanceEasyNetManifest,
            [inovanceComputerLinkManifest.DriverId] = inovanceComputerLinkManifest,
            [keyenceMc3ETcpManifest.DriverId] = keyenceMc3ETcpManifest,
            [keyenceMcAsciiTcpManifest.DriverId] = keyenceMcAsciiTcpManifest,
            [keyenceKvOldTcpManifest.DriverId] = keyenceKvOldTcpManifest,
            [keyenceNanoTcpManifest.DriverId] = keyenceNanoTcpManifest,
            [keyenceNanoSerialManifest.DriverId] = keyenceNanoSerialManifest,
            [keyenceNanoSerialOverTcpManifest.DriverId] = keyenceNanoSerialOverTcpManifest,
            [lsisFastEnetManifest.DriverId] = lsisFastEnetManifest,
            [lsisCnetManifest.DriverId] = lsisCnetManifest,
            [lsisCnetOverTcpManifest.DriverId] = lsisCnetOverTcpManifest,
            [lsisCpuSerialManifest.DriverId] = lsisCpuSerialManifest,
            [megMeetTcpManifest.DriverId] = megMeetTcpManifest,
            [megMeetRtuOverTcpManifest.DriverId] = megMeetRtuOverTcpManifest,
            [megMeetRtuManifest.DriverId] = megMeetRtuManifest,
            [panasonicMewtocolTcpManifest.DriverId] = panasonicMewtocolTcpManifest,
            [panasonicMewtocolSerialManifest.DriverId] = panasonicMewtocolSerialManifest,
            [panasonicMcBinaryTcpManifest.DriverId] = panasonicMcBinaryTcpManifest,
            [opcUaManifest.DriverId] = opcUaManifest,
            [mitsubishiMc3ETcpManifest.DriverId] = mitsubishiMc3ETcpManifest,
            [mitsubishiA1EAsciiTcpManifest.DriverId] = mitsubishiA1EAsciiTcpManifest,
            [mitsubishiA1EBinaryTcpManifest.DriverId] = mitsubishiA1EBinaryTcpManifest,
            [mitsubishiMcAsciiTcpManifest.DriverId] = mitsubishiMcAsciiTcpManifest,
            [mitsubishiMcAsciiUdpManifest.DriverId] = mitsubishiMcAsciiUdpManifest,
            [mitsubishiMcBinaryUdpManifest.DriverId] = mitsubishiMcBinaryUdpManifest,
            [mitsubishiMcRBinaryTcpManifest.DriverId] = mitsubishiMcRBinaryTcpManifest,
            [mitsubishiCipManifest.DriverId] = mitsubishiCipManifest,
            [mitsubishiA3CSerialManifest.DriverId] = mitsubishiA3CSerialManifest,
            [mitsubishiA3CSerialOverTcpManifest.DriverId] = mitsubishiA3CSerialOverTcpManifest,
            [mitsubishiFxLinksSerialManifest.DriverId] = mitsubishiFxLinksSerialManifest,
            [mitsubishiFxLinksOverTcpManifest.DriverId] = mitsubishiFxLinksOverTcpManifest,
            [mitsubishiFxSerialManifest.DriverId] = mitsubishiFxSerialManifest,
            [mitsubishiFxSerialOverTcpManifest.DriverId] = mitsubishiFxSerialOverTcpManifest,
            [modbusRtuManifest.DriverId] = modbusRtuManifest,
            [modbusRtuOverTcpManifest.DriverId] = modbusRtuOverTcpManifest,
            [modbusAsciiManifest.DriverId] = modbusAsciiManifest,
            [modbusAsciiOverTcpManifest.DriverId] = modbusAsciiOverTcpManifest,
            [modbusTcpManifest.DriverId] = modbusTcpManifest,
            [modbusUdpManifest.DriverId] = modbusUdpManifest,
            [mqttRpcDeviceManifest.DriverId] = mqttRpcDeviceManifest,
            [omronFinsTcpManifest.DriverId] = omronFinsTcpManifest,
            [omronFinsUdpManifest.DriverId] = omronFinsUdpManifest,
            [omronCipManifest.DriverId] = omronCipManifest,
            [omronConnectedCipManifest.DriverId] = omronConnectedCipManifest,
            [omronHostLinkManifest.DriverId] = omronHostLinkManifest,
            [omronHostLinkOverTcpManifest.DriverId] = omronHostLinkOverTcpManifest,
            [omronHostLinkCModeManifest.DriverId] = omronHostLinkCModeManifest,
            [omronHostLinkCModeOverTcpManifest.DriverId] = omronHostLinkCModeOverTcpManifest,
            [orientalMotorEipManifest.DriverId] = orientalMotorEipManifest,
            [rkcTemperatureControllerSerialManifest.DriverId] = rkcTemperatureControllerSerialManifest,
            [rkcTemperatureControllerTcpManifest.DriverId] = rkcTemperatureControllerTcpManifest,
            [estunRobotManifest.DriverId] = estunRobotManifest,
            [fanucRobotManifest.DriverId] = fanucRobotManifest,
            [siemensS7TcpManifest.DriverId] = siemensS7TcpManifest,
            [siemensS7PlusManifest.DriverId] = siemensS7PlusManifest,
            [siemensPpiManifest.DriverId] = siemensPpiManifest,
            [siemensPpiOverTcpManifest.DriverId] = siemensPpiOverTcpManifest,
            [siemensMpiManifest.DriverId] = siemensMpiManifest,
            [siemensFetchWriteManifest.DriverId] = siemensFetchWriteManifest,
            [siemensWebApiManifest.DriverId] = siemensWebApiManifest,
            [toyoPucManifest.DriverId] = toyoPucManifest,
            [turckReaderTcpManifest.DriverId] = turckReaderTcpManifest,
            [xinjeTcpManifest.DriverId] = xinjeTcpManifest,
            [xinjeRtuOverTcpManifest.DriverId] = xinjeRtuOverTcpManifest,
            [xinjeRtuManifest.DriverId] = xinjeRtuManifest,
            [xinjeInternalTcpManifest.DriverId] = xinjeInternalTcpManifest,
            [vigorSerialOverTcpManifest.DriverId] = vigorSerialOverTcpManifest,
            [vigorSerialManifest.DriverId] = vigorSerialManifest,
            [yamatakeDigitronSerialManifest.DriverId] = yamatakeDigitronSerialManifest,
            [yamatakeDigitronTcpManifest.DriverId] = yamatakeDigitronTcpManifest,
            [yuDianAiBusManifest.DriverId] = yuDianAiBusManifest,
            [yaskawaMemobusTcpManifest.DriverId] = yaskawaMemobusTcpManifest,
            [yaskawaMemobusUdpManifest.DriverId] = yaskawaMemobusUdpManifest,
            [yokogawaLinkTcpManifest.DriverId] = yokogawaLinkTcpManifest,
        });
    }

    private static DriverManifest LoadManifest(Assembly assembly, string resourceName)
    {
        using var stream = assembly.GetManifestResourceStream(resourceName)
            ?? throw new InvalidOperationException($"缺少内嵌驱动 Manifest：{resourceName}");
        return JsonSerializer.Deserialize<DriverManifest>(stream, ManifestJsonOptions)
            ?? throw new InvalidOperationException($"无法解析内嵌驱动 Manifest：{resourceName}");
    }
}
